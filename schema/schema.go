// Package schema has configs, models and global variables for all parts of hotspot.
package schema

import "time"

// FileResult represents the Git and file system metrics for a single file.
type FileResult struct {
	Path               string    `json:"path"`                 // Relative path to the file in the repository
	UniqueContributors Metric    `json:"unique_contributors"`  // Number of different authors who modified the file
	Commits            Metric    `json:"commits"`              // Total number of commits affecting this file
	RecentContributors Metric    `json:"recent_contributors"`  // Recent contributor count within a time window
	RecentCommits      Metric    `json:"recent_commits"`       // Recent commit count within a time window
	RecentChurn        Metric    `json:"recent_churn"`         // Recent churn within a time window
	RecentLinesAdded   Metric    `json:"recent_lines_added"`   // Recent lines added
	RecentLinesDeleted Metric    `json:"recent_lines_deleted"` // Recent lines deleted
	DecayedCommits     Metric    `json:"decayed_commits"`      // Time-weighted commit count
	DecayedChurn       Metric    `json:"decayed_churn"`        // Time-weighted churn count
	RecentWindowDays   int       `json:"recent_window_days"`   // Number of days defining the 'recent' window
	SizeBytes          int64     `json:"size_bytes"`           // Current size of the file in bytes (Stay int64 as it's a file property)
	LinesOfCode        Metric    `json:"lines_of_code"`        // Current lines of code
	AgeDays            Metric    `json:"age_days"`             // Age of the file in days since first commit
	Churn              Metric    `json:"churn"`                // Total number of lines added/deleted
	LinesAdded         Metric    `json:"lines_added"`          // Total lines added
	LinesDeleted       Metric    `json:"lines_deleted"`        // Total lines deleted
	Gini               float64   `json:"gini"`                 // Gini coefficient of commit distribution (0-1, lower is more even)
	FirstCommit        time.Time `json:"first_commit"`         // Timestamp of the file's first commit
	Owners             []string  `json:"owners"`               // Top 2 owners by commit count
	RecencySignal      float64   `json:"recency_signal"`       // 0-1 freshness score (recent activity vs lifetime volume)

	Mode          ScoringMode                              `json:"mode"`                 // Scoring mode used (hot, risk, complexity, roi)
	ModeScore     float64                                  `json:"score"`                // Computed score for the current mode (0-100)
	Reasoning     []string                                 `json:"reasoning,omitempty"`  // Human-and-AI-readable justifications for the score
	ModeBreakdown map[BreakdownKey]float64                 `json:"breakdown"`            // Normalized contribution of each metric to the score
	AllScores     map[ScoringMode]float64                  `json:"scores"`               // All computed scores by mode
	AllBreakdowns map[ScoringMode]map[BreakdownKey]float64 `json:"breakdowns,omitempty"` // Score breakdowns for all modes
}

// GetPath returns the file path.
func (f FileResult) GetPath() string {
	return f.Path
}

// GetScore returns the computed score.
func (f FileResult) GetScore() float64 {
	return f.ModeScore
}

// GetCommits returns the total commit count.
func (f FileResult) GetCommits() Metric {
	return f.Commits
}

// GetChurn returns the total churn.
func (f FileResult) GetChurn() Metric {
	return f.Churn
}

// GetOwners returns the top owners.
func (f FileResult) GetOwners() []string {
	if f.Owners == nil {
		return []string{}
	}
	return f.Owners
}

// FolderResult holds the final computed scores and aggregated metrics for a folder.
type FolderResult struct {
	Path           string   `json:"path"`            // Relative path to the folder in the repository
	Commits        Metric   `json:"commits"`         // Total number of commits across all contained files
	Churn          Metric   `json:"churn"`           // Total number of lines added/deleted across all contained files
	DecayedCommits Metric   `json:"decayed_commits"` // Time-weighted commits across all contained files
	DecayedChurn   Metric   `json:"decayed_churn"`   // Time-weighted churn across all contained files
	Score          float64  `json:"score"`           // Computed importance score for the folder
	Owners         []string `json:"owners"`          // Top 2 owners by commit count

	TotalLOC         Metric      `json:"total_loc"`          // Sum of LOC of all contained files (used for weighted average)
	WeightedScoreSum float64     `json:"weighted_score_sum"` // Sum of (FileScore * FileLOC)
	Mode             ScoringMode `json:"mode"`               // Scoring mode used (hot, risk, complexity, roi)
}

// GetPath returns the folder path.
func (f FolderResult) GetPath() string {
	return f.Path
}

// GetScore returns the computed score.
func (f FolderResult) GetScore() float64 {
	return f.Score
}

// GetCommits returns the total commit count.
func (f FolderResult) GetCommits() Metric {
	return f.Commits
}

// GetChurn returns the total churn.
func (f FolderResult) GetChurn() Metric {
	return f.Churn
}

// GetOwners returns the top owners.
func (f FolderResult) GetOwners() []string {
	if f.Owners == nil {
		return []string{}
	}
	return f.Owners
}
