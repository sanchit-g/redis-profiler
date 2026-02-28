package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "redis-profiler",
	Short: "A fast, concurrent Redis keyspace memory profiler",
	Long: `redis-profiler scans your Redis keyspace and produces a memory
breakdown by key pattern, TTL health analysis, and expiry cliff detection.

Safe to run on production — uses SCAN (never KEYS *), bounded concurrency,
and read-only commands only.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("redis-profiler: use --help to see available commands")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
