package schema

import "time"

// AggregateOutput is the aggregation of all things from the one-pass Git operation.
type AggregateOutput struct {
	CommitMap        map[string]Metric            // Maps file path to its commit count
	ChurnMap         map[string]Metric            // Maps file path to its churn (lines added/deleted) count
	ContribMap       map[string]map[string]Metric // Maps file path to an inner map of AuthorName:CommitCount
	FirstCommitMap   map[string]time.Time         // Maps file path to its first commit time in the analysis window
	DecayedCommitMap map[string]Metric            // Maps file path to time-weighted commit count
	DecayedChurnMap  map[string]Metric            // Maps file path to time-weighted churn count
	EndTime          time.Time                    // The end time of the analysis window (reference for decay)

	// Decomposed Churn
	LinesAddedMap   map[string]Metric
	LinesDeletedMap map[string]Metric

	// Recent Activity (Fixed window, e.g. 30 days)
	RecentCommitMap       map[string]Metric
	RecentChurnMap        map[string]Metric
	RecentLinesAddedMap   map[string]Metric
	RecentLinesDeletedMap map[string]Metric
	RecentContribMap      map[string]map[string]Metric
}

// FileMetrics represents raw git metrics for a single file.
type FileMetrics struct {
	AnalysisTime           time.Time
	TotalCommits           Metric
	TotalChurn             Metric
	LinesAdded             Metric
	LinesDeleted           Metric
	DecayedCommits         Metric
	DecayedChurn           Metric
	LinesOfCode            Metric
	ContributorCount       Metric
	RecentCommits          Metric
	RecentChurn            Metric
	RecentLinesAdded       Metric
	RecentLinesDeleted     Metric
	RecentContributorCount Metric
	AgeDays                Metric
	GiniCoefficient        float64
	FileOwner              string
	RecencySignal          float64
	RecencyThresholdLow    float64
	RecencyThresholdHigh   float64
}

// FileScores represents final computed scores for a single file.
type FileScores struct {
	AnalysisTime    time.Time
	HotScore        float64  // hot mode score
	RiskScore       float64  // risk mode score
	ComplexityScore float64  // complexity mode score
	ROIScore        float64  // roi mode score
	ScoreLabel      string   // current mode name
	Reasoning       []string // justifications for the current score
}

// BatchFileResult groups all data for a single file to be stored in the analysis store.
type BatchFileResult struct {
	Path    string
	Metrics FileMetrics
	Scores  FileScores
}

// AnalysisRunRecord represents a row from the hotspot_analysis_runs table.
type AnalysisRunRecord struct {
	AnalysisID         int64
	URN                string
	StartTime          time.Time
	EndTime            *time.Time
	RunDurationMs      *int32
	TotalFilesAnalyzed *int32
	ConfigParams       *string
}

// AnalysisQueryFilter provides filtering and pagination for analysis queries.
type AnalysisQueryFilter struct {
	URN    string // Filter by repository URN (empty = all)
	Limit  int    // Maximum number of results (0 = no limit)
	Offset int    // Number of results to skip
}
