package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/sanchit-g/redis-profiler/config"
	"github.com/sanchit-g/redis-profiler/internal/aggregator"
	"github.com/sanchit-g/redis-profiler/internal/progress"
	"github.com/sanchit-g/redis-profiler/internal/report"
	"github.com/sanchit-g/redis-profiler/internal/scanner"
	"github.com/sanchit-g/redis-profiler/internal/worker"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use: "profile",
	Short: "Profile the Redis keyspace and report memory usage",
	RunE: runProfile,
}

var cfgFile string

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "path to config file (default: ./config.yaml)")
}

func runProfile(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		return err
	}

	fmt.Printf("connecting to Redis at %s\n", cfg.Redis.Address)
	
	// connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to Redis: %v\n", err)
		return err
	}

	fmt.Println("connected successfully")

	// create progress tracker
	tracker := progress.New()
	done := make(chan struct{})

	// start progress display in background goroutine
	go tracker.Run(done)

	// create channels
	workCh := make(chan string, cfg.Scanner.BatchSize*2)
	resultsCh := make(chan worker.KeyRecord, 1000)

	// start scanner in background goroutine
	scanErrCh := make(chan error, 1)
	go func() {
		defer close(workCh)
		defer close(done)
		s := scanner.New(rdb, int64(cfg.Scanner.BatchSize), tracker)
		scanErrCh <- s.Scan(ctx, workCh)
	}()
	
	// start worker pool
	var wg sync.WaitGroup
	wrk := worker.New(rdb)
	for i := 0; i < cfg.Scanner.Workers; i++ {
		wg.Add(1)
		go wrk.Run(ctx, workCh, resultsCh, &wg)
	}

	// close results channel when workers are done
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// aggregator runs in main goroutine - blocks until resultsCh is closed
	agg := aggregator.New(cfg.Groups)
	agg.Run(resultsCh)
	
	fmt.Printf("scan complete: %d keys found\n", tracker.Current())

	// check for scan errors
	if err := <-scanErrCh; err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		return err
	}

	// print results
	report.Print(
		agg.Results(),
		cfg.Redis.Address,
		cfg.Output.CliffThreshold,
		cfg.Output.CliffWindowMins,
		cfg.Output.CliffIgnoreGroups,
	)

	return nil
}