package schema

import "time"

// FileAggregation consolidates all metrics for a single file during aggregation.
type FileAggregation struct {
	Commits        Metric
	Churn          Metric
	LinesAdded     Metric
	LinesDeleted   Metric
	DecayedCommits Metric
	DecayedChurn   Metric
	FirstCommit    time.Time
	Contributors   map[string]Metric // Author name -> commit count

	// Recent Activity (Fixed window, e.g. 30 days)
	RecentCommits      Metric
	RecentChurn        Metric
	RecentLinesAdded   Metric
	RecentLinesDeleted Metric
	RecentContributors map[string]Metric
}

// AggregateOutput is the aggregation of all things from the one-pass Git operation.
type AggregateOutput struct {
	FileStats map[string]*FileAggregation
	EndTime   time.Time // The end time of the analysis window (reference for decay)
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
