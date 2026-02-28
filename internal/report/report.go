package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sanchit-g/redis-profiler/internal/aggregator"
)

const (
	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
)

func humanBytes(bytes int64) string {
	if bytes < KB {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < MB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	}
	if bytes < GB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
}

func printHeader(groups []aggregator.GroupStats, redisAddr string) {
	totalKeys := int64(0)
	totalBytes := int64(0)

	for _, group := range groups {
		totalKeys += group.KeyCount
		totalBytes += group.TotalBytes
	}

	fmt.Println()
	fmt.Println("Redis Memory Profiler")
	fmt.Printf("Connected: %s  |  Total keys: %d  |  Total memory: %s\n",
		redisAddr, totalKeys, humanBytes(totalBytes))
	fmt.Println()
}

func printTable(groups []aggregator.GroupStats) {
	// sort by memory descending - largest first
	sorted := make([]aggregator.GroupStats, len(groups))
	copy(sorted, groups)
	sort.Slice(sorted, func (i, j int) bool {
		return sorted[i].TotalBytes > sorted[j].TotalBytes
	})

	// header
	fmt.Printf("%-22s %-12s %-14s %-12s %-8s\n",
		"GROUP", "KEYS", "MEMORY", "AVG SIZE", "NO TTL")
	fmt.Println(strings.Repeat("─", 72))

	// rows
	totalKeys := int64(0)
	totalBytes := int64(0)

	for _, group := range sorted {
		if group.KeyCount == 0 {
			continue
		}

		avgSize := int64(0)
		if group.KeyCount > 0 {
			avgSize = group.TotalBytes / group.KeyCount
		}

		noTTLPct := float64(0)
		if group.KeyCount > 0 {
			noTTLPct = float64(group.NoTTLCount) / float64(group.KeyCount) * 100
		}

		warning := ""
		if noTTLPct > 50 {
			warning = "  ⚠"
		}

		fmt.Printf("%-22s %-12d %-14s %-12s %.0f%%%s\n",
			group.Name,
			group.KeyCount,
			humanBytes(group.TotalBytes),
			humanBytes(avgSize),
			noTTLPct,
			warning,
		)

		totalKeys += group.KeyCount
		totalBytes += group.TotalBytes
	}

	// totals
	fmt.Println(strings.Repeat("─", 72))
	fmt.Printf("%-22s %-12d %-14s\n", "TOTAL", totalKeys, humanBytes(totalBytes))
	fmt.Println()
}

func printCliffs(groups []aggregator.GroupStats, threshold int, windowMins int, ignoreGroups []string) {
	// build a set of ignored groups for quick lookup
	ignored := make(map[string]bool)
	for _, name := range ignoreGroups {
		ignored[name] = true
	}
	
	windowSecs := int64(windowMins * 60)
	found := false

	for _, group := range groups {
		if group.KeyCount == 0 || ignored[group.Name] {
			continue
		}

		for bucketStart, count := range group.ExpiryBuckets {
			pct := float64(count) / float64(group.KeyCount) * 100
			if pct >= float64(threshold) {
				if !found {
					fmt.Println("⚠  BULK EXPIRY EVENTS DETECTED")
					fmt.Println()
					found = true
				}

				bucketEnd := bucketStart + windowSecs
				fmt.Printf("   Group   : %s\n", group.Name)
				fmt.Printf("   Window  : %ds – %ds from now  |  %d keys (%.2f%% of group)\n",
					bucketStart, bucketEnd, count, pct)
				fmt.Println("   Note    : If these keys receive read traffic at expiry,")
				fmt.Println("             cache miss rates may spike.")
				fmt.Println()
			}
		}
	}

	if !found {
		fmt.Println("✓  No bulk expiry events detected")
	}
}

func Print(
	groups []aggregator.GroupStats,
	redisAddr string,
	cliffThreshold int,
	cliffWindowMins int,
	cliffIgnoreGroups []string,
) {
	printHeader(groups, redisAddr)
	printTable(groups)
	printCliffs(groups, cliffThreshold, cliffWindowMins, cliffIgnoreGroups)
}