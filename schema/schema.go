// Package schema has configs, models and global variables for all parts of hotspot.
package schema

import "time"

// FileResult represents the Git and file system metrics for a single file.
type FileResult struct {
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
	Owners             []string           `json:"owners"`              // Top 2 owners by commit count
}

// GetPath returns the file path.
func (f FileResult) GetPath() string {
	return f.Path
}

// GetScore returns the computed score.
func (f FileResult) GetScore() float64 {
	return f.Score
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
	return f.Owners
}

// FolderResult holds the final computed scores and aggregated metrics for a folder.
type FolderResult struct {
	Path    string   `json:"path"`    // Relative path to the folder in the repository
	Commits int      `json:"commits"` // Total number of commits across all contained files
	Churn   int      `json:"churn"`   // Total number of lines added/deleted across all contained files
	Score   float64  `json:"score"`   // Computed importance score for the folder
	Owners  []string `json:"owners"`  // Top 2 owners by commit count

	TotalLOC         int     `json:"total_loc"`          // Sum of LOC of all contained files (used for weighted average)
	WeightedScoreSum float64 `json:"weighted_score_sum"` // Sum of (FileScore * FileLOC)
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
	return f.Owners
}

// ComparisonDetails holds the base info, target info, and their associated deltas.
type ComparisonDetails struct {
	Path         string   `json:"path"`          // Relative path to the target in the repository
	BeforeScore  float64  `json:"before_score"`  // Score from the original/base analysis
	AfterScore   float64  `json:"after_score"`   // Score from the comparison/new analysis
	Delta        float64  `json:"delta"`         // CompScore - BaseScore (Positive means worse/higher)
	DeltaCommits int      `json:"delta_commits"` // Change in total commits (Positive means more activity)
	DeltaChurn   int      `json:"delta_churn"`   // Change in total churn (Positive means more volatility)
	Status       string   `json:"status"`        // Intrinsic status of the file as of now
	BeforeOwners []string `json:"before_owners"` // Owners from the base analysis
	AfterOwners  []string `json:"after_owners"`  // Owners from the target analysis

	*FileComparison   `json:"file_compare,omitempty"`
	*FolderComparison `json:"folder_compare,omitempty"`
}

// ComparisonSummary has high-level deltas and counts.
type ComparisonSummary struct {
	// 1. Net Score Delta
	NetScoreDelta float64 `json:"net_score_delta"`

	// 2. Net Churn Delta
	NetChurnDelta int `json:"net_churn_delta"`

	// 3. File Status Counts
	TotalNewFiles      int `json:"total_new_files"`
	TotalInactiveFiles int `json:"total_inactive_files"`
	TotalModifiedFiles int `json:"total_modified_files"`

	// 4. Ownership Changes
	TotalOwnershipChanges int `json:"total_ownership_changes"`
}

// ComparisonResult holds the comparison details and summary.
type ComparisonResult struct {
	Results []ComparisonDetails `json:"details"`
	Summary ComparisonSummary   `json:"summary"`
}

// FileComparison has file deltas.
type FileComparison struct {
	DeltaLOC     int `json:"delta_loc"`     // Change in LOC (Positive means file growth)
	DeltaContrib int `json:"delta_contrib"` // Change in contributors (Positive means contrib growth)
}

// FolderComparison has folder deltas.
type FolderComparison struct{}

// AggregateOutput is the aggregation of all things from the one-pass Git operation.
type AggregateOutput struct {
	CommitMap      map[string]int            // Maps file path to its commit count
	ChurnMap       map[string]int            // Maps file path to its churn (lines added/deleted) count
	ContribMap     map[string]map[string]int // Maps file path to an inner map of AuthorName:CommitCount
	FirstCommitMap map[string]time.Time      // Maps file path to its first commit time in the analysis window
}

// SingleAnalysisOutput is for one of the core algorithms.
type SingleAnalysisOutput struct {
	FileResults []FileResult
	*AggregateOutput
}

// CompareAnalysisOutput is for one of the core algorithms.
type CompareAnalysisOutput struct {
	FileResults   []FileResult
	FolderResults []FolderResult
}

// TimeseriesPoint represents a single data point in the timeseries.
type TimeseriesPoint struct {
	Period string  `json:"period"`
	Score  float64 `json:"score"`
	Mode   string  `json:"mode"`
	Path   string  `json:"path"`
}

// TimeseriesResult holds the timeseries data points.
type TimeseriesResult struct {
	Points []TimeseriesPoint `json:"points"`
}
