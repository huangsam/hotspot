package core

var (
	// RecentCommitsMapGlobal is used when running CLI with start param
	RecentCommitsMapGlobal map[string]int

	// RecentChurnMapGlobal is used when running CLI with start param
	RecentChurnMapGlobal map[string]int
)

// RecentContribMapGlobal maps paths to authors to commit count.
var RecentContribMapGlobal map[string]map[string]int
