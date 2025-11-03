package schema

import "sync"

var (
	// recentCommitsMapGlobal is used when running CLI with start param.
	recentCommitsMapGlobal map[string]int

	// recentChurnMapGlobal is used when running CLI with start param.
	recentChurnMapGlobal map[string]int

	// recentContribMapGlobal maps paths to authors to commit count.
	recentContribMapGlobal map[string]map[string]int

	// Add a sync.Once for each map to guarantee initialization only happens once
	commitsOnce sync.Once
	churnOnce   sync.Once
	contribOnce sync.Once
)

// GetRecentCommitsMapGlobal returns the recent commits map.
func GetRecentCommitsMapGlobal() map[string]int {
	commitsOnce.Do(func() {
		recentCommitsMapGlobal = make(map[string]int)
	})
	return recentCommitsMapGlobal
}

// GetRecentChurnMapGlobal returns the recent churn map.
func GetRecentChurnMapGlobal() map[string]int {
	churnOnce.Do(func() {
		recentChurnMapGlobal = make(map[string]int)
	})
	return recentChurnMapGlobal
}

// GetRecentContribMapGlobal returns the recent contrib map.
func GetRecentContribMapGlobal() map[string]map[string]int {
	contribOnce.Do(func() {
		recentContribMapGlobal = make(map[string]map[string]int)
	})
	return recentContribMapGlobal
}
