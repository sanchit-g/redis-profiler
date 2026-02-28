package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/redis/go-redis/v9"
)

// KeyRecord holds everything we know about a single Redis key
// after the worker has inspected it.
type KeyRecord struct {
	Name     string
	NameHash uint64
	Bytes    int64
	KeyType  string
	TTL      int64
}

type Worker struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Worker {
	return &Worker{rdb: rdb}
}

func (w *Worker) Run(
	ctx context.Context,
	workCh <-chan string,
	resultsCh chan<- KeyRecord,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for key := range workCh {
		record, err := w.processKey(ctx, key)
		if err != nil {
			fmt.Printf("error processing key %s: %v\n", key, err)
			continue
		}
		resultsCh <- record
	}
}

func (w *Worker) processKey(ctx context.Context, key string) (KeyRecord, error) {
	pipe := w.rdb.Pipeline()
	memCmd := pipe.MemoryUsage(ctx, key, 5)
	typeCmd := pipe.Type(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	if _, err := pipe.Exec(ctx); err != nil {
		return KeyRecord{}, fmt.Errorf("pipeline error for key %s: %w", key, err)
	}

	mem, err := memCmd.Result()
	if err != nil {
		mem = 0
	}

	keyType, err := typeCmd.Result()
	if err != nil {
		keyType = "unknown"
	}

	ttlDuration, err := ttlCmd.Result()
	if err != nil {
    	ttlDuration = time.Duration(0)
	}

	// check special values BEFORE converting to seconds
	ttlSeconds := int64(ttlDuration.Seconds())
	if ttlDuration == -1 * time.Nanosecond {
    	ttlSeconds = -1   // no expiry
	} else if ttlDuration == -2 * time.Nanosecond {
    	ttlSeconds = -2   // key does not exist
	}

	return KeyRecord{
		Name:     key,
		NameHash: xxhash.Sum64String(key),
		Bytes:    mem,
		KeyType:  keyType,
		TTL:      ttlSeconds,
	}, nil
}