package aggregator

import (
	"path/filepath"

	"github.com/sanchit-g/redis-profiler/config"
	"github.com/sanchit-g/redis-profiler/internal/worker"
)


type GroupStats struct {
	Name 			string
	Pattern			string
	KeyCount		int64
	TotalBytes		int64
	TypeCounts		map[string]int64
	NoTTLCount		int64
	ExpiryBuckets	map[int64]int64
}

type Aggregator struct {
	groups []GroupStats
	seenKeys map[uint64]struct{}
}

func New(groupConfigs []config.GroupConfig) *Aggregator {
	groups := make([]GroupStats, 0, len(groupConfigs)+1)

	for _, g := range groupConfigs {
		groups = append(groups, GroupStats{
			Name: g.Name,
			Pattern: g.Pattern,
			KeyCount: 0,
			TotalBytes: 0,
			TypeCounts: make(map[string]int64),
			NoTTLCount: 0,
			ExpiryBuckets: make(map[int64]int64),
		})
	}

	// always add an un classified group at the end
	groups = append(groups, GroupStats{
		Name: "[unclassified]",
		Pattern: "*",
		KeyCount: 0,
		TotalBytes: 0,
		TypeCounts: make(map[string]int64),
		NoTTLCount: 0,
		ExpiryBuckets: make(map[int64]int64),
	})

	return &Aggregator{
		groups: groups,
		seenKeys: make(map[uint64]struct{}),
	}
}

func (a *Aggregator) findGroup(key string) *GroupStats {
	for i := range a.groups {
		matched, err := filepath.Match(a.groups[i].Pattern, key)
		if err == nil && matched {
			return &a.groups[i]
		}
	}
	// return unclassified group which is always the last one
	return &a.groups[len(a.groups)-1]
}

func (a *Aggregator) Run(resultsCh <-chan worker.KeyRecord) {
	for record := range resultsCh {
		// deduplication check
		if _, seen := a.seenKeys[record.NameHash]; seen {
			continue
		}
		a.seenKeys[record.NameHash] = struct{}{}

		// find which group this key belongs to
		group := a.findGroup(record.Name)
		
		// accumulate stats
		group.KeyCount++
		group.TotalBytes += record.Bytes
		group.TypeCounts[record.KeyType]++
		
		if record.TTL == -1 {
			group.NoTTLCount++
		} else {
			// bucket by 5-min windows
			bucket := (record.TTL / 300) * 300
			group.ExpiryBuckets[bucket]++
		}
	}
}

func (a *Aggregator) Results() []GroupStats {
	return a.groups
}