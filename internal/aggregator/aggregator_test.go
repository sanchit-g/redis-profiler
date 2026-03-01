package aggregator

import (
	"testing"

	"github.com/sanchit-g/redis-profiler/config"
	"github.com/sanchit-g/redis-profiler/internal/worker"
)

func TestNew(t *testing.T) {
	groups := []config.GroupConfig{
		{Name: "Sessions", Pattern: "session:*"},
		{Name: "Cache", Pattern: "cache:*"},
	}
	
	agg := New(groups)
	
	if agg == nil {
		t.Fatal("New() returned nil")
	}
	
	// Should have configured groups + unclassified group
	if len(agg.groups) != 3 {
		t.Errorf("Expected 3 groups (2 configured + 1 unclassified), got %d", len(agg.groups))
	}
	
	// Check unclassified group is last
	lastGroup := agg.groups[len(agg.groups)-1]
	if lastGroup.Name != "[unclassified]" {
		t.Errorf("Expected last group to be '[unclassified]', got '%s'", lastGroup.Name)
	}
	if lastGroup.Pattern != "*" {
		t.Errorf("Expected unclassified pattern to be '*', got '%s'", lastGroup.Pattern)
	}
	
	// Check seenKeys map is initialized
	if agg.seenKeys == nil {
		t.Error("seenKeys map not initialized")
	}
}

func TestFindGroup(t *testing.T) {
	groups := []config.GroupConfig{
		{Name: "Sessions", Pattern: "session:*"},
		{Name: "Cache", Pattern: "cache:*"},
		{Name: "Product", Pattern: "product:*"},
	}
	
	agg := New(groups)
	
	tests := []struct {
		key           string
		expectedGroup string
	}{
		{"session:user:123", "Sessions"},
		{"session:admin:456", "Sessions"},
		{"cache:item:789", "Cache"},
		{"product:details:100", "Product"},
		{"unknown:key:abc", "[unclassified]"},
		{"orphan", "[unclassified]"},
	}
	
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			group := agg.findGroup(tt.key)
			if group.Name != tt.expectedGroup {
				t.Errorf("findGroup(%s) = %s, want %s", tt.key, group.Name, tt.expectedGroup)
			}
		})
	}
}

func TestFindGroupFirstMatchWins(t *testing.T) {
	// Test that first matching pattern wins
	groups := []config.GroupConfig{
		{Name: "All Sessions", Pattern: "session:*"},
		{Name: "User Sessions", Pattern: "session:user:*"},
	}
	
	agg := New(groups)
	
	// Should match "All Sessions" since it comes first
	group := agg.findGroup("session:user:123")
	if group.Name != "All Sessions" {
		t.Errorf("Expected 'All Sessions' (first match), got '%s'", group.Name)
	}
}

func TestRunBasicAggregation(t *testing.T) {
	groups := []config.GroupConfig{
		{Name: "Sessions", Pattern: "session:*"},
	}
	
	agg := New(groups)
	
	// Create a channel and send test records
	resultsCh := make(chan worker.KeyRecord, 10)
	
	records := []worker.KeyRecord{
		{Name: "session:user:1", NameHash: 1, Bytes: 100, KeyType: "string", TTL: 3600},
		{Name: "session:user:2", NameHash: 2, Bytes: 150, KeyType: "string", TTL: 7200},
		{Name: "session:user:3", NameHash: 3, Bytes: 200, KeyType: "hash", TTL: -1},
	}
	
	for _, r := range records {
		resultsCh <- r
	}
	close(resultsCh)
	
	// Run aggregation
	agg.Run(resultsCh)
	
	// Verify results
	results := agg.Results()
	if len(results) == 0 {
		t.Fatal("No results returned")
	}
	
	// Find Sessions group
	var sessionsGroup *GroupStats
	for i := range results {
		if results[i].Name == "Sessions" {
			sessionsGroup = &results[i]
			break
		}
	}
	
	if sessionsGroup == nil {
		t.Fatal("Sessions group not found in results")
	}
	
	// Verify aggregated stats
	if sessionsGroup.KeyCount != 3 {
		t.Errorf("Expected KeyCount 3, got %d", sessionsGroup.KeyCount)
	}
	if sessionsGroup.TotalBytes != 450 {
		t.Errorf("Expected TotalBytes 450, got %d", sessionsGroup.TotalBytes)
	}
	if sessionsGroup.TypeCounts["string"] != 2 {
		t.Errorf("Expected 2 string keys, got %d", sessionsGroup.TypeCounts["string"])
	}
	if sessionsGroup.TypeCounts["hash"] != 1 {
		t.Errorf("Expected 1 hash key, got %d", sessionsGroup.TypeCounts["hash"])
	}
	if sessionsGroup.NoTTLCount != 1 {
		t.Errorf("Expected NoTTLCount 1, got %d", sessionsGroup.NoTTLCount)
	}
}

func TestRunDeduplication(t *testing.T) {
	groups := []config.GroupConfig{
		{Name: "Test", Pattern: "test:*"},
	}
	
	agg := New(groups)
	
	resultsCh := make(chan worker.KeyRecord, 10)
	
	// Send same key twice (same hash)
	records := []worker.KeyRecord{
		{Name: "test:key:1", NameHash: 12345, Bytes: 100, KeyType: "string", TTL: 3600},
		{Name: "test:key:1", NameHash: 12345, Bytes: 100, KeyType: "string", TTL: 3600},
		{Name: "test:key:2", NameHash: 67890, Bytes: 200, KeyType: "string", TTL: 3600},
	}
	
	for _, r := range records {
		resultsCh <- r
	}
	close(resultsCh)
	
	agg.Run(resultsCh)
	
	results := agg.Results()
	var testGroup *GroupStats
	for i := range results {
		if results[i].Name == "Test" {
			testGroup = &results[i]
			break
		}
	}
	
	if testGroup == nil {
		t.Fatal("Test group not found")
	}
	
	// Should only count 2 unique keys, not 3
	if testGroup.KeyCount != 2 {
		t.Errorf("Expected KeyCount 2 (deduplicated), got %d", testGroup.KeyCount)
	}
	if testGroup.TotalBytes != 300 {
		t.Errorf("Expected TotalBytes 300, got %d", testGroup.TotalBytes)
	}
}

func TestRunTTLBucketing(t *testing.T) {
	groups := []config.GroupConfig{
		{Name: "Test", Pattern: "test:*"},
	}
	
	agg := New(groups)
	
	resultsCh := make(chan worker.KeyRecord, 10)
	
	// Send keys with various TTLs
	records := []worker.KeyRecord{
		{Name: "test:1", NameHash: 1, Bytes: 100, KeyType: "string", TTL: 100},   // bucket 0
		{Name: "test:2", NameHash: 2, Bytes: 100, KeyType: "string", TTL: 250},   // bucket 0
		{Name: "test:3", NameHash: 3, Bytes: 100, KeyType: "string", TTL: 350},   // bucket 300
		{Name: "test:4", NameHash: 4, Bytes: 100, KeyType: "string", TTL: 650},   // bucket 600
		{Name: "test:5", NameHash: 5, Bytes: 100, KeyType: "string", TTL: -1},    // no TTL
	}
	
	for _, r := range records {
		resultsCh <- r
	}
	close(resultsCh)
	
	agg.Run(resultsCh)
	
	results := agg.Results()
	var testGroup *GroupStats
	for i := range results {
		if results[i].Name == "Test" {
			testGroup = &results[i]
			break
		}
	}
	
	if testGroup == nil {
		t.Fatal("Test group not found")
	}
	
	// Verify bucketing (5-minute buckets = 300 seconds)
	if testGroup.ExpiryBuckets[0] != 2 {
		t.Errorf("Expected 2 keys in bucket 0, got %d", testGroup.ExpiryBuckets[0])
	}
	if testGroup.ExpiryBuckets[300] != 1 {
		t.Errorf("Expected 1 key in bucket 300, got %d", testGroup.ExpiryBuckets[300])
	}
	if testGroup.ExpiryBuckets[600] != 1 {
		t.Errorf("Expected 1 key in bucket 600, got %d", testGroup.ExpiryBuckets[600])
	}
	if testGroup.NoTTLCount != 1 {
		t.Errorf("Expected 1 key with no TTL, got %d", testGroup.NoTTLCount)
	}
}

func TestRunMultipleGroups(t *testing.T) {
	groups := []config.GroupConfig{
		{Name: "Sessions", Pattern: "session:*"},
		{Name: "Cache", Pattern: "cache:*"},
	}
	
	agg := New(groups)
	
	resultsCh := make(chan worker.KeyRecord, 10)
	
	records := []worker.KeyRecord{
		{Name: "session:1", NameHash: 1, Bytes: 100, KeyType: "string", TTL: 3600},
		{Name: "session:2", NameHash: 2, Bytes: 150, KeyType: "string", TTL: 3600},
		{Name: "cache:1", NameHash: 3, Bytes: 200, KeyType: "string", TTL: 1800},
		{Name: "cache:2", NameHash: 4, Bytes: 250, KeyType: "hash", TTL: 1800},
		{Name: "orphan:1", NameHash: 5, Bytes: 50, KeyType: "string", TTL: -1},
	}
	
	for _, r := range records {
		resultsCh <- r
	}
	close(resultsCh)
	
	agg.Run(resultsCh)
	
	results := agg.Results()
	
	// Verify each group
	groupMap := make(map[string]*GroupStats)
	for i := range results {
		groupMap[results[i].Name] = &results[i]
	}
	
	// Sessions group
	if sessions, ok := groupMap["Sessions"]; ok {
		if sessions.KeyCount != 2 {
			t.Errorf("Sessions: expected 2 keys, got %d", sessions.KeyCount)
		}
		if sessions.TotalBytes != 250 {
			t.Errorf("Sessions: expected 250 bytes, got %d", sessions.TotalBytes)
		}
	} else {
		t.Error("Sessions group not found")
	}
	
	// Cache group
	if cache, ok := groupMap["Cache"]; ok {
		if cache.KeyCount != 2 {
			t.Errorf("Cache: expected 2 keys, got %d", cache.KeyCount)
		}
		if cache.TotalBytes != 450 {
			t.Errorf("Cache: expected 450 bytes, got %d", cache.TotalBytes)
		}
	} else {
		t.Error("Cache group not found")
	}
	
	// Unclassified group
	if unclassified, ok := groupMap["[unclassified]"]; ok {
		if unclassified.KeyCount != 1 {
			t.Errorf("Unclassified: expected 1 key, got %d", unclassified.KeyCount)
		}
		if unclassified.TotalBytes != 50 {
			t.Errorf("Unclassified: expected 50 bytes, got %d", unclassified.TotalBytes)
		}
	} else {
		t.Error("Unclassified group not found")
	}
}

func TestRunEmptyChannel(t *testing.T) {
	groups := []config.GroupConfig{
		{Name: "Test", Pattern: "test:*"},
	}
	
	agg := New(groups)
	
	resultsCh := make(chan worker.KeyRecord)
	close(resultsCh)
	
	// Should not panic
	agg.Run(resultsCh)
	
	results := agg.Results()
	if len(results) == 0 {
		t.Error("Expected results even with empty channel")
	}
	
	// All groups should have zero counts
	for _, group := range results {
		if group.KeyCount != 0 {
			t.Errorf("Group %s: expected 0 keys, got %d", group.Name, group.KeyCount)
		}
	}
}
