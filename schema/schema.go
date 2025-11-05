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
	Owner              string             `json:"owner"`               // Owner is the individual who has committed the most to this file
}

// FolderResults holds the final computed scores and aggregated metrics for a folder.
type FolderResults struct {
	Path    string  `json:"path"`    // Relative path to the folder in the repository
	Commits int     `json:"commits"` // Total number of commits across all contained files
	Churn   int     `json:"churn"`   // Total number of lines added/deleted across all contained files
	Score   float64 `json:"score"`   // Computed importance score for the folder
	Owner   string  `json:"owner"`   // Owner is the individual who has committed the most to the files in this folder

	// Fields for folder scoring calculation:
	TotalLOC         int     `json:"total_loc"`          // Sum of LOC of all contained files (used for weighted average)
	WeightedScoreSum float64 `json:"weighted_score_sum"` // Sum of (FileScore * FileLOC)
}

// ComparisonMetrics holds the base metrics, comparison metrics, and their deltas.
type ComparisonMetrics struct {
	Path      string
	BaseScore float64 // Score from the original/base analysis
	CompScore float64 // Score from the comparison/new analysis
	Delta     float64 // CompScore - BaseScore (Positive means worse/higher)

	DeltaCommits int // Change in total commits (Positive means more activity)
	DeltaChurn   int // Change in total churn (Positive means more volatility)
	DeltaLOC     int // Change in Lines of Code (Positive means file growth)
	DeltaContrib int // Change in Unique Contributors (Positive means more contrib growth)
}

// AggregateOutput is the aggregation of all things from the one-pass Git operation.
type AggregateOutput struct {
	CommitMap  map[string]int            // Maps file path to its commit count
	ChurnMap   map[string]int            // Maps file path to its churn (lines added/deleted) count
	ContribMap map[string]map[string]int // Maps file path to an inner map of AuthorName:CommitCount
}
