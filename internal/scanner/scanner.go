package scanner

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Scanner struct {
	rdb *redis.Client
	batchSize int64
}

func New(rdb *redis.Client, batchSize int64) *Scanner {
	return &Scanner{
		rdb: rdb,
		batchSize: batchSize,
	}
}

func (s *Scanner) Scan(ctx context.Context, workCh chan<- string) error {
	var cursor uint64
	totalKeys := 0

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, "*", s.batchSize).Result()
		if err != nil {
			return fmt.Errorf("scan error at cursor %d: %w", cursor, err)
		}

		for _, key := range keys {
			workCh <- key
			totalKeys++
		}

		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	fmt.Printf("scan complete: %d keys found\n", totalKeys)
	return nil
}