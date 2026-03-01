package report

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sanchit-g/redis-profiler/internal/aggregator"
)

// captureOutput intercepts stdout for testing
func captureOutput(f func()) string {
	oldOutput := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldOutput
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintHeader(t *testing.T) {
	groups := []aggregator.GroupStats{
		{Name: "Test1", KeyCount: 100, TotalBytes: 5000},
		{Name: "Test2", KeyCount: 200, TotalBytes: 10000},
	}
	
	output := captureOutput(func() {
		printHeader(groups, "localhost:6379")
	})
	
	if !strings.Contains(output, "Redis Memory Profiler") {
		t.Errorf("Expected output to contain 'Redis Memory Profiler', got: %s", output)
	}
	if !strings.Contains(output, "Connected: localhost:6379") {
		t.Error("Expected output to contain connected address")
	}
	if !strings.Contains(output, "Total keys: 300") {
		t.Error("Expected output to contain total keys")
	}
	if !strings.Contains(output, "Total memory: 14.65 KB") {
		t.Error("Expected output to contain total memory")
	}
}

func TestPrintHeaderEmpty(t *testing.T) {
	groups := []aggregator.GroupStats{}
	
	output := captureOutput(func() {
		printHeader(groups, "test:6379")
	})
	
	if !strings.Contains(output, "Total keys: 0") {
		t.Error("Expected output to contain 0 keys")
	}
}

func TestPrintTable(t *testing.T) {
	groups := []aggregator.GroupStats{
		{
			Name:         "Sessions",
			Pattern:      "session:*",
			KeyCount:     100,
			TotalBytes:   50000,
			TypeCounts:   map[string]int64{"string": 100},
			NoTTLCount:   10,
			ExpiryBuckets: map[int64]int64{300: 50, 600: 50},
		},
		{
			Name:         "Cache",
			Pattern:      "cache:*",
			KeyCount:     50,
			TotalBytes:   25000,
			TypeCounts:   map[string]int64{"hash": 50},
			NoTTLCount:   0,
			ExpiryBuckets: map[int64]int64{},
		},
	}
	
	output := captureOutput(func() {
		printTable(groups)
	})
	
	if !strings.Contains(output, "Sessions") || !strings.Contains(output, "Cache") {
		t.Error("Expected output to contain group names")
	}
	if !strings.Contains(output, "48.83 KB") || !strings.Contains(output, "24.41 KB") {
		t.Error("Expected output to contain memory sizes")
	}
	if !strings.Contains(output, "10%") || !strings.Contains(output, "0%") {
		t.Error("Expected output to contain No TTL percentages")
	}
}

func TestPrintTableEmptyGroups(t *testing.T) {
	groups := []aggregator.GroupStats{}
	
	output := captureOutput(func() {
		printTable(groups)
	})
	
	if !strings.Contains(output, "TOTAL") || !strings.Contains(output, "0 B") {
		t.Error("Expected output to contain empty summary")
	}
}

func TestPrintTableWithWarning(t *testing.T) {
	groups := []aggregator.GroupStats{
		{
			Name:        "Risky",
			KeyCount:    100,
			TotalBytes:  5000,
			NoTTLCount:  60, // 60%
		},
	}
	
	output := captureOutput(func() {
		printTable(groups)
	})
	
	if !strings.Contains(output, "⚠") {
		t.Error("Expected output to contain warning symbol for high No TTL percentage")
	}
}

func TestPrintCliffsWithCliff(t *testing.T) {
	groups := []aggregator.GroupStats{
		{
			Name:       "Test",
			KeyCount:   100,
			ExpiryBuckets: map[int64]int64{
				0:   10,
				300: 80, // 80% - threshold is 20
				600: 10,
			},
		},
	}
	
	output := captureOutput(func() {
		printCliffs(groups, 20, 5, nil)
	})
	
	if !strings.Contains(output, "BULK EXPIRY EVENTS DETECTED") {
		t.Error("Expected output to warn about bulk expiry")
	}
	if !strings.Contains(output, "300s \u2013 600s from now") {
		t.Error("Expected output to contain formatting for time window")
	}
}

func TestPrintCliffsNoCliff(t *testing.T) {
	groups := []aggregator.GroupStats{
		{
			Name:       "Test",
			KeyCount:   100,
			ExpiryBuckets: map[int64]int64{
				0:   33,
				300: 33,
				600: 34,
			},
		},
	}
	
	output := captureOutput(func() {
		printCliffs(groups, 20, 5, nil)
	})
	
	if !strings.Contains(output, "No bulk expiry events detected") {
		t.Error("Expected no bulk expiry events detected")
	}
}

func TestPrintCliffsIgnoredGroup(t *testing.T) {
	groups := []aggregator.GroupStats{
		{
			Name:       "Ignored",
			KeyCount:   100,
			ExpiryBuckets: map[int64]int64{
				300: 100,
			},
		},
	}
	
	output := captureOutput(func() {
		printCliffs(groups, 20, 5, []string{"Ignored"})
	})
	
	if !strings.Contains(output, "No bulk expiry events detected") {
		t.Error("Expected no bulk expiry events detected due to ignored group")
	}
}

func TestPrintCliffsZeroKeyCount(t *testing.T) {
	groups := []aggregator.GroupStats{
		{
			Name:          "Test",
			KeyCount:      0,
			ExpiryBuckets: map[int64]int64{300: 50},
		},
	}
	
	output := captureOutput(func() {
		printCliffs(groups, 20, 5, nil)
	})
	
	if !strings.Contains(output, "No bulk expiry events detected") {
		t.Error("Expected skipped cliff logic on zero key count")
	}
}

func TestPrint(t *testing.T) {
	groups := []aggregator.GroupStats{
		{
			Name:       "Test",
			KeyCount:   10,
			TotalBytes: 1000,
		},
	}
	
	output := captureOutput(func() {
		Print(groups, "localhost:6379", 20, 5, nil)
	})
	
	if !strings.Contains(output, "Redis Memory Profiler") || !strings.Contains(output, "Test") {
		t.Error("Expected full print string to contain header and group")
	}
}
