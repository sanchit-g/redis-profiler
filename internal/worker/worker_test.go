package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestNew(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()
	
	worker := New(rdb)
	
	if worker == nil {
		t.Fatal("New() returned nil")
	}
	
	if worker.rdb == nil {
		t.Error("Worker rdb field is nil")
	}
}

func TestKeyRecordHash(t *testing.T) {
	// Test that same key produces same hash
	key1 := "test:key:123"
	key2 := "test:key:123"
	key3 := "test:key:456"
	
	hash1 := xxhash.Sum64String(key1)
	hash2 := xxhash.Sum64String(key2)
	hash3 := xxhash.Sum64String(key3)
	
	if hash1 != hash2 {
		t.Error("Same keys should produce same hash")
	}
	
	if hash1 == hash3 {
		t.Error("Different keys should produce different hashes")
	}
}

func TestWorker_processKey_Mocks(t *testing.T) {
	db, mock := redismock.NewClientMock()
	
	worker := New(db)
	ctx := context.Background()
	testKey := "mock:test:key"
	
	// Set up pipeline expectations
	mock.ExpectMemoryUsage(testKey, 5).SetVal(1024)
	mock.ExpectType(testKey).SetVal("string")
	mock.ExpectTTL(testKey).SetVal(3600 * time.Second)
	
	record, err := worker.processKey(ctx, testKey)
	
	if err != nil {
		t.Fatalf("processKey() unexpected error: %v", err)
	}
	
	if record.Name != testKey {
		t.Errorf("Expected Name %s, got %s", testKey, record.Name)
	}
	
	if record.Bytes != 1024 {
		t.Errorf("Expected Bytes 1024, got %d", record.Bytes)
	}
	
	if record.KeyType != "string" {
		t.Errorf("Expected KeyType string, got %s", record.KeyType)
	}
	
	if record.TTL != 3600 {
		t.Errorf("Expected TTL 3600, got %d", record.TTL)
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestWorker_processKey_NoExpiry_Mocks(t *testing.T) {
	db, mock := redismock.NewClientMock()
	
	worker := New(db)
	ctx := context.Background()
	testKey := "mock:test:noexpiry"
	
	// -1 * time.Nanosecond is returned by go-redis for keys with no expiry
	mock.ExpectMemoryUsage(testKey, 5).SetVal(512)
	mock.ExpectType(testKey).SetVal("hash")
	mock.ExpectTTL(testKey).SetVal(-1 * time.Nanosecond)
	
	record, err := worker.processKey(ctx, testKey)
	
	if err != nil {
		t.Fatalf("processKey() unexpected error: %v", err)
	}
	
	if record.TTL != -1 {
		t.Errorf("Expected TTL -1 for no expiry, got %d", record.TTL)
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestWorker_processKey_NonExistent_Mocks(t *testing.T) {
	db, mock := redismock.NewClientMock()
	
	worker := New(db)
	ctx := context.Background()
	testKey := "mock:test:nonexistent"
	
	// -2 * time.Nanosecond is returned for non-existent keys
	// MemoryUsage returns an error for non-existent keys.
	// But in redismock, setting an Err on a pipeline command causes pipe.Exec() to fail with that error.
	// To test the recovery, we expect Exec to return nil but memCmd.Result() to return an error.
	// Redismock supports this by using expect-but-don't-set-err and handling it specifically, or we can just expect the pipeline to fail completely and verify the error.
	
	mockErr := errors.New("ERR no such key")
	// If any command in the pipeline has an error, Exec() returns that error.
	// NOTE: Depending on redismock implementation, it might only process the first erred command in pipeline.
	mock.ExpectMemoryUsage(testKey, 5).SetErr(mockErr)
	// Do not expect Type and TTL here since the mock pipeline executor short-circuits on the first error.
	
	_, err := worker.processKey(ctx, testKey)
	
	// The processKey method does: 
	// if _, err := pipe.Exec(ctx); err != nil { return KeyRecord{}, fmt.Errorf(...) }
	// So it WILL error out if the pipeline returns an error.
	if err == nil {
		t.Fatal("processKey() expected error, got nil")
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

