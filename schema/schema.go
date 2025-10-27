// Package schema has configs, models and global variables for all parts of hotspot.
package schema

import "time"

// FileMetrics represents the Git and file system metrics for a single file.
// It includes contribution statistics, commit history, size, age, and derived metrics
// used to determine the file's overall importance score.
type FileMetrics struct {
	Path               string             `json:"path"`                // Relative path to the file in the repository
	UniqueContributors int                `json:"unique_contributors"` // Number of different authors who modified the file
	Commits            int                `json:"commits"`             // Total number of commits affecting this file
	RecentContributors int                `json:"recent_contributors"` // Recent contributor count within a time window
	RecentCommits      int                `json:"recent_commits"`      // Recent commit count within a time window
	RecentChurn        int                `json:"recent_churn"`        // Recent churn within a time window
	SizeBytes          int64              `json:"size_bytes"`          // Current size of the file in bytes
	LinesOfCode        int                `json:"lines_of_code"`       // Current lines of code
	AgeDays            int                `json:"age_days"`            // Age of the file in days since first commit
	Churn              int                `json:"churn"`               // Total number of lines added/deleted plus number of commits
	Gini               float64            `json:"gini"`                // Gini coefficient of commit distribution (0-1, lower is more even)
	FirstCommit        time.Time          `json:"first_commit"`        // Timestamp of the file's first commit
	Score              float64            `json:"score"`               // Computed importance score (0-100)
	Breakdown          map[string]float64 `json:"breakdown"`           // Normalized contribution of each metric for debugging/tuning
}

// Breakdown keys used in the scoring logic.
const (
	BreakdownContrib = "contrib" // nContrib
	BreakdownCommits = "commits" // nCommits
	BreakdownLOC     = "loc"     // nLOC
	BreakdownSize    = "size"    // nSize
	BreakdownAge     = "age"     // nAge
	BreakdownChurn   = "churn"   // nChurn

	BreakdownGini       = "gini"        // nGiniRaw
	BreakdownInvContrib = "inv_contrib" // nInvContrib
	BreakdownInvRecent  = "inv_recent"  // nInvRecentCommits (used in stale)
	BreakdownLowRecent  = "low_recent"  // nInvRecentCommits (used in complexity)
)
