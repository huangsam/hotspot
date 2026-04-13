package schema

import "time"

// AggregateOutput is the aggregation of all things from the one-pass Git operation.
type AggregateOutput struct {
	CommitMap      map[string]int            // Maps file path to its commit count
	ChurnMap       map[string]int            // Maps file path to its churn (lines added/deleted) count
	ContribMap     map[string]map[string]int // Maps file path to an inner map of AuthorName:CommitCount
	FirstCommitMap map[string]time.Time      // Maps file path to its first commit time in the analysis window

	// Decomposed Churn
	LinesAddedMap   map[string]int
	LinesDeletedMap map[string]int

	// Recent Activity (Fixed window, e.g. 30 days)
	RecentCommitMap       map[string]int
	RecentChurnMap        map[string]int
	RecentLinesAddedMap   map[string]int
	RecentLinesDeletedMap map[string]int
	RecentContribMap      map[string]map[string]int
}

// FileMetrics represents raw git metrics for a single file.
type FileMetrics struct {
	AnalysisTime           time.Time
	TotalCommits           int
	TotalChurn             int
	LinesAdded             int
	LinesDeleted           int
	ContributorCount       int
	RecentCommits          int
	RecentChurn            int
	RecentLinesAdded       int
	RecentLinesDeleted     int
	RecentContributorCount int
	AgeDays                float64
	GiniCoefficient        float64
	FileOwner              string
}

// FileScores represents final computed scores for a single file.
type FileScores struct {
	AnalysisTime    time.Time
	HotScore        float64 // hot mode score
	RiskScore       float64 // risk mode score
	ComplexityScore float64 // complexity mode score
	StaleScore      float64 // stale mode score
	ScoreLabel      string  // current mode name
}

// AnalysisRunRecord represents a row from the hotspot_analysis_runs table.
type AnalysisRunRecord struct {
	AnalysisID         int64
	URN                string
	StartTime          time.Time
	EndTime            *time.Time
	RunDurationMs      *int32
	TotalFilesAnalyzed int32
	ConfigParams       *string
}

// AnalysisQueryFilter provides filtering and pagination for analysis queries.
type AnalysisQueryFilter struct {
	URN    string // Filter by repository URN (empty = all)
	Limit  int    // Maximum number of results (0 = no limit)
	Offset int    // Number of results to skip
}
