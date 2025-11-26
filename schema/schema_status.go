package schema

import "time"

// CacheStatus represents the status of the cache store.
type CacheStatus struct {
	Backend         string    `json:"backend"`
	Connected       bool      `json:"connected"`
	TotalEntries    int       `json:"total_entries"`
	LastEntryTime   time.Time `json:"last_entry_time"`
	OldestEntryTime time.Time `json:"oldest_entry_time"`
	TableSizeBytes  int64     `json:"table_size_bytes"`
}

// AnalysisStatus represents the status of the analysis store.
type AnalysisStatus struct {
	Backend            string           `json:"backend"`
	Connected          bool             `json:"connected"`
	TotalRuns          int              `json:"total_runs"`
	LastRunID          int64            `json:"last_run_id"`
	LastRunTime        time.Time        `json:"last_run_time"`
	OldestRunTime      time.Time        `json:"oldest_run_time"`
	TotalFilesAnalyzed int              `json:"total_files_analyzed"`
	TableSizes         map[string]int64 `json:"table_sizes"`
}

// FileScoresMetricsRecord represents a row from the hotspot_file_scores_metrics table.
type FileScoresMetricsRecord struct {
	AnalysisID       int64
	FilePath         string
	AnalysisTime     time.Time
	TotalCommits     int32
	TotalChurn       int32
	ContributorCount int32
	AgeDays          float64
	GiniCoefficient  float64
	FileOwner        *string
	ScoreHot         float64
	ScoreRisk        float64
	ScoreComplexity  float64
	ScoreStale       float64
	ScoreLabel       string
}
