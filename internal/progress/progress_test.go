package progress

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tracker := New()
	
	if tracker == nil {
		t.Fatal("New() returned nil")
	}
	
	if tracker.Current() != 0 {
		t.Errorf("Expected initial count 0, got %d", tracker.Current())
	}
}

func TestIncrement(t *testing.T) {
	tracker := New()
	
	tracker.Increment()
	
	if tracker.Current() != 1 {
		t.Errorf("Expected count 1 after Increment(), got %d", tracker.Current())
	}
	
	tracker.Increment()
	tracker.Increment()
	
	if tracker.Current() != 3 {
		t.Errorf("Expected count 3 after 3 Increment() calls, got %d", tracker.Current())
	}
}

func TestIncrementConcurrent(t *testing.T) {
	tracker := New()
	
	// Run 100 goroutines, each incrementing 100 times
	numGoroutines := 100
	incrementsPerGoroutine := 100
	expectedTotal := numGoroutines * incrementsPerGoroutine
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				tracker.Increment()
			}
		}()
	}
	
	wg.Wait()
	
	if tracker.Current() != int64(expectedTotal) {
		t.Errorf("Expected count %d after concurrent increments, got %d", expectedTotal, tracker.Current())
	}
}

func TestCurrent(t *testing.T) {
	tracker := New()
	
	if tracker.Current() != 0 {
		t.Errorf("Expected Current() 0 initially, got %d", tracker.Current())
	}
	
	tracker.Increment()
	tracker.Increment()
	tracker.Increment()
	
	if tracker.Current() != 3 {
		t.Errorf("Expected Current() 3, got %d", tracker.Current())
	}
}

func TestCurrentConcurrent(t *testing.T) {
	tracker := New()
	
	// Increment in background
	done := make(chan bool)
	go func() {
		for i := 0; i < 1000; i++ {
			tracker.Increment()
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()
	
	// Read count concurrently
	for i := 0; i < 100; i++ {
		count := tracker.Current()
		if count < 0 || count > 1000 {
			t.Errorf("Current() returned invalid value: %d", count)
		}
		time.Sleep(time.Microsecond * 10)
	}
	
	<-done
}

func TestRun(t *testing.T) {
	tracker := New()
	
	// Start the Run goroutine
	done := make(chan struct{})
	
	go tracker.Run(done)
	
	// Increment some values
	for i := 0; i < 100; i++ {
		tracker.Increment()
	}
	
	// Stop the tracker
	close(done)
	
	// Give it time to clean up
	time.Sleep(50 * time.Millisecond)
	
	// Verify count
	if tracker.Current() != 100 {
		t.Errorf("Expected count 100, got %d", tracker.Current())
	}
}

func TestRunStopsOnDone(t *testing.T) {
	tracker := New()
	
	done := make(chan struct{})
	
	finished := make(chan bool)
	go func() {
		tracker.Run(done)
		finished <- true
	}()
	
	// Let it run for a bit
	time.Sleep(10 * time.Millisecond)
	
	// Signal done
	close(done)
	
	// Wait for Run to finish
	select {
	case <-finished:
		// Success - Run() returned
	case <-time.After(time.Second):
		t.Error("Run() did not stop after done channel was closed")
	}
}

func TestMultipleTrackers(t *testing.T) {
	// Test that multiple trackers work independently
	tracker1 := New()
	tracker2 := New()
	
	tracker1.Increment()
	tracker1.Increment()
	
	tracker2.Increment()
	tracker2.Increment()
	tracker2.Increment()
	
	if tracker1.Current() != 2 {
		t.Errorf("Tracker1: expected count 2, got %d", tracker1.Current())
	}
	
	if tracker2.Current() != 3 {
		t.Errorf("Tracker2: expected count 3, got %d", tracker2.Current())
	}
}

func TestLargeNumbers(t *testing.T) {
	tracker := New()
	
	// Increment a large number of times
	for i := 0; i < 100000; i++ {
		tracker.Increment()
	}
	
	if tracker.Current() != 100000 {
		t.Errorf("Expected count 100000, got %d", tracker.Current())
	}
}

func BenchmarkIncrement(b *testing.B) {
	tracker := New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Increment()
	}
}

func BenchmarkIncrementParallel(b *testing.B) {
	tracker := New()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tracker.Increment()
		}
	})
}

func BenchmarkCurrent(b *testing.B) {
	tracker := New()
	tracker.Increment()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.Current()
	}
}
