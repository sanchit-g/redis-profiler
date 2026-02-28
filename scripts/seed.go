package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Printf("could not connect to Redis: %v\n", err)
		return
	}

	fmt.Println("Connected to Redis, starting seeding")

	seedSessions(ctx, rdb)
	seedProductCache(ctx, rdb)
	seedRateLimiters(ctx, rdb)
	seedNoPattern(ctx, rdb)
	seedExpiryCliff(ctx, rdb)

	fmt.Println("Seeding done")
}

func seedSessions(ctx context.Context, rdb *redis.Client) {
	fmt.Println("seeding session keys...")

	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("session:user:%d", rand.Intn(500000))
		value := fmt.Sprintf(`{"user_id":%d,"token":"abc123","created_at":"2024-01-01"}`, rand.Intn(500000))
		ttl := time.Hour*24 + time.Duration(rand.Intn(3600))*time.Second

		if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
			fmt.Printf("could not set key %s: %v\n", key, err)
		}
	}

	fmt.Println("seeding session keys done")
}

func seedProductCache(ctx context.Context, rdb *redis.Client) {
	fmt.Println("seeding product cache...")

	for i := 0; i < 50000; i++ {
		key := fmt.Sprintf("product:cache:item-%d", rand.Intn(200000))
		value := fmt.Sprintf(`{"price":%d,"stock":%d}`, rand.Intn(500000), rand.Intn(500000))
		ttl := time.Hour + time.Duration(rand.Intn(1800))*time.Second

		if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
			fmt.Printf("could not set key %s: %v\n", key, err)
		}
	}

	fmt.Println("seeding product cache done")
}

func seedRateLimiters(ctx context.Context, rdb *redis.Client) {
	fmt.Println("seeding rate limiter keys...")

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("ratelimit:ip:192.168.%d.%d", rand.Intn(255), rand.Intn(255))
		value := fmt.Sprintf("%d", rand.Intn(100))
		ttl := time.Second*60 + time.Duration(rand.Intn(60))*time.Second

		if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
			fmt.Printf("could not set key %s: %v\n", key, err)
		}
	}

	fmt.Println("seeding rate limiter keys done")
}

func seedNoPattern(ctx context.Context, rdb *redis.Client) {
	fmt.Println("seeding orphan keys with no pattern...")

	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("orphan:unknown:%d", rand.Intn(100000))
		value := "unclassified_data"
		ttl := time.Duration(0)

		if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
			fmt.Printf("could not set key %s: %v\n", key, err)
		}
	}

	fmt.Println("seeding orphan keys done")
}

func seedExpiryCliff(ctx context.Context, rdb *redis.Client) {
	fmt.Println("seeding expiry cliff keys...")

	for i := 0; i < 20000; i++ {
		key := fmt.Sprintf("session:bulk:%d", rand.Intn(500000))
		value := "bulk_session_data"
		ttl := time.Minute * 10

		if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
			fmt.Printf("could not set key %s: %v\n", key, err)
		}
	}

	fmt.Println("seeding expiry cliff keys done")
}