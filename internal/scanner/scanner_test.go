package scanner

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/sanchit-g/redis-profiler/internal/progress"
)

func TestNew(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	
	tracker := progress.New()
	scanner := New(rdb, 100, tracker)
	
	if scanner == nil {
		t.Fatal("New() returned nil")
	}
	
	if scanner.rdb == nil {
		t.Error("Scanner rdb field is nil")
	}
	
	if scanner.batchSize != 100 {
		t.Errorf("Expected batchSize 100, got %d", scanner.batchSize)
	}
	
	if scanner.tracker == nil {
		t.Error("Scanner tracker field is nil")
	}
}

func TestNewWithDifferentBatchSizes(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	
	tests := []int64{10, 50, 100, 500, 1000}
	
	for _, batchSize := range tests {
		tracker := progress.New()
		scanner := New(rdb, batchSize, tracker)
		if scanner.batchSize != batchSize {
			t.Errorf("Expected batchSize %d, got %d", batchSize, scanner.batchSize)
		}
	}
}

func TestNewScannerZeroBatchSize(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	
	tracker := progress.New()
	scanner := New(rdb, 0, tracker)
	
	if scanner.batchSize != 0 {
		t.Errorf("Expected batchSize 0, got %d", scanner.batchSize)
	}
}

func TestNewScannerNegativeBatchSize(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	
	tracker := progress.New()
	scanner := New(rdb, -10, tracker)
	
	if scanner.batchSize != -10 {
		t.Errorf("Expected batchSize -10, got %d", scanner.batchSize)
	}
}

func TestScanner_Scan_Mocks(t *testing.T) {
	db, mock := redismock.NewClientMock()
	
	tracker := progress.New()
	scanner := New(db, 2, tracker)
	
	ctx := context.Background()
	
	// Mock SCAN behavior
	// First call: returns 2 keys, cursor 10
	mock.ExpectScan(0, "*", 2).SetVal([]string{"key1", "key2"}, 10)
	// Second call: returns 1 key, cursor 0 (done)
	mock.ExpectScan(10, "*", 2).SetVal([]string{"key3"}, 0)
	
	workCh := make(chan string, 10)
	
	err := scanner.Scan(ctx, workCh)
	if err != nil {
		t.Fatalf("Scan() unexpected error: %v", err)
	}
	
	close(workCh)
	
	// Verify keys were sent to channel
	var keys []string
	for k := range workCh {
		keys = append(keys, k)
	}
	
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}
	
	if tracker.Current() != 3 {
		t.Errorf("Expected tracker to be 3, got %d", tracker.Current())
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestScanner_Scan_Error_Mocks(t *testing.T) {
	db, mock := redismock.NewClientMock()
	
	tracker := progress.New()
	scanner := New(db, 10, tracker)
	
	ctx := context.Background()
	mockErr := errors.New("simulated network error")
	
	// Mock SCAN to return error
	mock.ExpectScan(0, "*", 10).SetErr(mockErr)
	
	workCh := make(chan string, 10)
	
	err := scanner.Scan(ctx, workCh)
	
	if err == nil {
		t.Fatal("Scan() expected error, got nil")
	}
	
	if !errors.Is(err, mockErr) && err.Error() != fmt.Sprintf("scan error at cursor 0: %v", mockErr) {
		t.Errorf("Expected wrapped network error, got: %v", err)
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
