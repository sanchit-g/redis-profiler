package progress

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

type Tracker struct {
	count		int64
	startTime	time.Time
}

func New() *Tracker {
	return &Tracker{
		startTime: time.Now(),
	}
}

func (t *Tracker) Increment() {
	atomic.AddInt64(&t.count, 1)
}

func (t* Tracker) Current() int64 {
	return atomic.LoadInt64(&t.count)
}

func (t* Tracker) print() {
	elapsed := time.Since(t.startTime).Seconds()
	count := t.Current()

	rate := int64(0)
	if elapsed > 0 {
		rate = int64(float64(count) / elapsed)
	}

	fmt.Printf("\rScanning... %d keys processed (%d keys/sec)",
		count, rate)
}

func (t* Tracker) clear() {
	fmt.Printf("\r%s\r", strings.Repeat(" ", 60))
}

func (t* Tracker) Run(done <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.print()
		case <-done:
			t.clear()
			time.Sleep(10 * time.Millisecond)   // ensure clear is flushed to terminal
			return
		}
	}
}