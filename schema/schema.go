// Package schema has configs, models and global variables for all parts of hotspot.
package schema

import "time"

// FileResult represents the Git and file system metrics for a single file.
type FileResult struct {
	Path               string    `json:"path"`                // Relative path to the file in the repository
	UniqueContributors int       `json:"unique_contributors"` // Number of different authors who modified the file
	Commits            int       `json:"commits"`             // Total number of commits affecting this file
	RecentContributors int       `json:"recent_contributors"` // Recent contributor count within a time window
	RecentCommits      int       `json:"recent_commits"`      // Recent commit count within a time window
	RecentChurn        int       `json:"recent_churn"`        // Recent churn within a time window
	SizeBytes          int64     `json:"size_bytes"`          // Current size of the file in bytes
	LinesOfCode        int       `json:"lines_of_code"`       // Current lines of code
	AgeDays            int       `json:"age_days"`            // Age of the file in days since first commit
	Churn              int       `json:"churn"`               // Total number of lines added/deleted plus number of commits
	Gini               float64   `json:"gini"`                // Gini coefficient of commit distribution (0-1, lower is more even)
	FirstCommit        time.Time `json:"first_commit"`        // Timestamp of the file's first commit
	Owners             []string  `json:"owners"`              // Top 2 owners by commit count

	Mode          ScoringMode                              `json:"mode"`                 // Scoring mode used (hot, risk, complexity, stale)
	ModeScore     float64                                  `json:"score"`                // Computed score for the current mode (0-100)
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
func (f FileResult) GetCommits() int {
	return f.Commits
}

// GetChurn returns the total churn.
func (f FileResult) GetChurn() int {
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
	Path    string   `json:"path"`    // Relative path to the folder in the repository
	Commits int      `json:"commits"` // Total number of commits across all contained files
	Churn   int      `json:"churn"`   // Total number of lines added/deleted across all contained files
	Score   float64  `json:"score"`   // Computed importance score for the folder
	Owners  []string `json:"owners"`  // Top 2 owners by commit count

	TotalLOC         int         `json:"total_loc"`          // Sum of LOC of all contained files (used for weighted average)
	WeightedScoreSum float64     `json:"weighted_score_sum"` // Sum of (FileScore * FileLOC)
	Mode             ScoringMode `json:"mode"`               // Scoring mode used (hot, risk, complexity, stale)
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
func (f FolderResult) GetCommits() int {
	return f.Commits
}

// GetChurn returns the total churn.
func (f FolderResult) GetChurn() int {
	return f.Churn
}

// GetOwners returns the top owners.
func (f FolderResult) GetOwners() []string {
	if f.Owners == nil {
		return []string{}
	}
	return f.Owners
}
