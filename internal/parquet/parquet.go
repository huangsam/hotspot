// Package parquet provides data structures and functions for exporting hotspot
// analysis data to Parquet files using github.com/parquet-go/parquet-go.
package parquet

import (
	"fmt"
	"os"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/parquet-go/parquet-go"
)

// AnalysisRun represents a single hotspot analysis run with metadata.
// This struct maps to the hotspot_analysis_runs database table.
type AnalysisRun struct {
	// AnalysisID is the unique identifier for this analysis run
	AnalysisID int64 `parquet:"analysis_id,snappy"`

	// URN is the repository universal resource name (e.g., git:github.com/org/repo)
	URN *string `parquet:"urn,optional,snappy"`

	// StartTime is when the analysis began (stored as TIMESTAMP with nanosecond precision)
	StartTime time.Time `parquet:"start_time,snappy"`

	// EndTime is when the analysis completed (nullable, stored as TIMESTAMP with nanosecond precision)
	EndTime *time.Time `parquet:"end_time,optional,snappy"`

	// RunDurationMs is the duration of the analysis run in milliseconds (nullable)
	RunDurationMs *int32 `parquet:"run_duration_ms,optional,snappy"`

	// TotalFilesAnalyzed is the number of files analyzed in this run
	TotalFilesAnalyzed *int32 `parquet:"total_files_analyzed,optional,snappy"`

	// ConfigParams contains the JSON-encoded configuration parameters (nullable)
	ConfigParams *string `parquet:"config_params,optional,snappy"`
}

// FileScoresMetrics represents the metrics and scores for a single file in an analysis.
// This struct maps to the hotspot_file_scores_metrics database table.
type FileScoresMetrics struct {
	// AnalysisID references the parent analysis run
	AnalysisID int64 `parquet:"analysis_id,snappy"`

	// FilePath is the relative path to the file in the repository
	FilePath string `parquet:"file_path,snappy"`

	// AnalysisTime is when this file was analyzed (stored as TIMESTAMP with nanosecond precision)
	AnalysisTime time.Time `parquet:"analysis_time,snappy"`

	// TotalCommits is the number of commits affecting this file
	TotalCommits float64 `parquet:"total_commits,snappy"`

	// TotalChurn is the number of lines added/deleted in this file
	TotalChurn float64 `parquet:"total_churn,snappy"`

	// LinesOfCode is the current number of lines in the file
	LinesOfCode float64 `parquet:"lines_of_code,snappy"`

	// ContributorCount is the number of unique contributors to this file
	ContributorCount float64 `parquet:"contributor_count,snappy"`

	// AgeDays is the age of the file in days since first commit
	AgeDays float64 `parquet:"age_days,snappy"`

	// GiniCoefficient measures commit distribution (0-1, lower is more even)
	GiniCoefficient float64 `parquet:"gini_coefficient,snappy"`

	// LinesAdded is the total number of lines added to the file
	LinesAdded float64 `parquet:"lines_added,snappy"`

	// LinesDeleted is the total number of lines deleted from the file
	LinesDeleted float64 `parquet:"lines_deleted,snappy"`

	// DecayedCommits is the time-weighted commit count
	DecayedCommits float64 `parquet:"decayed_commits,snappy"`

	// DecayedChurn is the time-weighted churn count
	DecayedChurn float64 `parquet:"decayed_churn,snappy"`

	// RecentCommits is the number of commits in the recent window
	RecentCommits float64 `parquet:"recent_commits,snappy"`

	// RecentChurn is the total churn in the recent window
	RecentChurn float64 `parquet:"recent_churn,snappy"`

	// RecentContributorCount is the number of unique contributors in the recent window
	RecentContributorCount float64 `parquet:"recent_contributor_count,snappy"`

	// RecentLinesAdded is the number of lines added in the recent window
	RecentLinesAdded float64 `parquet:"recent_lines_added,snappy"`

	// RecentLinesDeleted is the number of lines deleted in the recent window
	RecentLinesDeleted float64 `parquet:"recent_lines_deleted,snappy"`

	// FileOwner is the primary owner of the file (nullable)
	FileOwner *string `parquet:"file_owner,optional,snappy"`

	// ScoreHot is the hotspot score in hot mode
	ScoreHot float64 `parquet:"score_hot,snappy"`

	// ScoreRisk is the hotspot score in risk mode
	ScoreRisk float64 `parquet:"score_risk,snappy"`

	// ScoreComplexity is the hotspot score in complexity mode
	ScoreComplexity float64 `parquet:"score_complexity,snappy"`

	// ScoreROI is the hotspot score in ROI mode
	ScoreROI float64 `parquet:"score_roi,snappy"`

	// ScoreLabel indicates which scoring mode was used
	ScoreLabel string `parquet:"score_label,snappy"`

	// Reasoning contains human-and-AI-readable justifications for the scores
	Reasoning []string `parquet:"reasoning,snappy"`

	// RecencySignal is the freshness score for the file
	RecencySignal float64 `parquet:"recency_signal,snappy"`

	// RecencyThresholdLow is the scale-aware baseline for recency
	RecencyThresholdLow float64 `parquet:"recency_threshold_low,snappy"`

	// RecencyThresholdHigh is the scale-aware ceiling for recency
	RecencyThresholdHigh float64 `parquet:"recency_threshold_high,snappy"`
}

// WriteAnalysisRunsParquet writes a slice of AnalysisRun structs to a Parquet file.
func WriteAnalysisRunsParquet(data []AnalysisRun, outputPath string) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Create a Parquet writer using struct schema inference
	// The schema is automatically derived from the AnalysisRun struct tags
	writer := parquet.NewGenericWriter[AnalysisRun](file)
	defer func() { _ = writer.Close() }()

	// Write all records to the file
	// The Write method accepts a variadic slice
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write data to parquet file: %w", err)
	}

	return nil
}

// WriteFileScoresMetricsParquet writes a slice of FileScoresMetrics structs to a Parquet file.
func WriteFileScoresMetricsParquet(data []FileScoresMetrics, outputPath string) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Create a Parquet writer using struct schema inference
	// The schema is automatically derived from the FileScoresMetrics struct tags
	writer := parquet.NewGenericWriter[FileScoresMetrics](file)
	defer func() { _ = writer.Close() }()

	// Write all records to the file
	// The Write method accepts a variadic slice
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write data to parquet file: %w", err)
	}

	return nil
}

// MockFetchAnalysisRuns generates sample AnalysisRun data for demonstration.
func MockFetchAnalysisRuns() []AnalysisRun {
	now := time.Now()
	startTime1 := now.Add(-2 * time.Hour)
	endTime1 := now.Add(-1*time.Hour - 30*time.Minute)
	durationMs1 := int32(endTime1.Sub(startTime1).Milliseconds())
	configParams1 := `{"mode":"hot","limit":100,"lookback":"30d"}`

	startTime2 := now.Add(-24 * time.Hour)
	endTime2 := now.Add(-23 * time.Hour)
	durationMs2 := int32(endTime2.Sub(startTime2).Milliseconds())
	configParams2 := `{"mode":"risk","limit":50,"lookback":"60d"}`

	startTime3 := now.Add(-10 * time.Minute)
	// Note: endTime3, durationMs3, configParams3 are nil to demonstrate nullable fields

	urn1 := "git:github.com/example/hotspot"
	urn2 := "git:github.com/example/analytics"

	fileCount1 := int32(150)
	fileCount2 := int32(75)

	return []AnalysisRun{
		{
			AnalysisID:         1,
			URN:                &urn1,
			StartTime:          startTime1,
			EndTime:            &endTime1,
			RunDurationMs:      &durationMs1,
			TotalFilesAnalyzed: &fileCount1,
			ConfigParams:       &configParams1,
		},
		{
			AnalysisID:         2,
			URN:                &urn2,
			StartTime:          startTime2,
			EndTime:            &endTime2,
			RunDurationMs:      &durationMs2,
			TotalFilesAnalyzed: &fileCount2,
			ConfigParams:       &configParams2,
		},
		{
			AnalysisID:         3,
			URN:                nil, // No URN for in-progress run
			StartTime:          startTime3,
			EndTime:            nil, // Still running - nullable field
			RunDurationMs:      nil, // Not yet calculated - nullable field
			TotalFilesAnalyzed: nil,
			ConfigParams:       nil, // No config stored - nullable field
		},
	}
}

// MockFetchFileScoresMetrics generates sample FileScoresMetrics data for demonstration.
func MockFetchFileScoresMetrics() []FileScoresMetrics {
	now := time.Now()
	owner1 := "alice@example.com"
	owner2 := "bob@example.com"

	return []FileScoresMetrics{
		{
			AnalysisID:       1,
			FilePath:         "src/main.go",
			AnalysisTime:     now.Add(-1 * time.Hour),
			TotalCommits:     42,
			TotalChurn:       850,
			ContributorCount: 5,
			AgeDays:          365.5,
			GiniCoefficient:  0.35,
			FileOwner:        &owner1,
			ScoreHot:         85.3,
			ScoreRisk:        62.1,
			ScoreComplexity:  71.8,
			ScoreLabel:       "hot",
		},
		{
			AnalysisID:       1,
			FilePath:         "src/utils/helper.go",
			AnalysisTime:     now.Add(-1 * time.Hour),
			TotalCommits:     18,
			TotalChurn:       320,
			ContributorCount: 3,
			AgeDays:          180.0,
			GiniCoefficient:  0.42,
			FileOwner:        &owner2,
			ScoreHot:         45.7,
			ScoreRisk:        38.9,
			ScoreComplexity:  52.3,
			ScoreLabel:       "hot",
		},
		{
			AnalysisID:       2,
			FilePath:         "test/fixture.go",
			AnalysisTime:     now.Add(-23 * time.Hour),
			TotalCommits:     5,
			TotalChurn:       125,
			ContributorCount: 2,
			AgeDays:          90.0,
			GiniCoefficient:  0.60,
			FileOwner:        nil, // No clear owner - nullable field
			ScoreHot:         12.3,
			ScoreRisk:        8.5,
			ScoreComplexity:  10.2,
			ScoreLabel:       "risk",
		},
	}
}

// ConvertAnalysisRunRecords converts schema.AnalysisRunRecord to AnalysisRun for Parquet export.
func ConvertAnalysisRunRecords(records []schema.AnalysisRunRecord) []AnalysisRun {
	result := make([]AnalysisRun, len(records))
	for i, record := range records {
		var urn *string
		if record.URN != "" {
			urn = &record.URN
		}
		result[i] = AnalysisRun{
			AnalysisID:         record.AnalysisID,
			URN:                urn,
			StartTime:          record.StartTime,
			EndTime:            record.EndTime,
			RunDurationMs:      record.RunDurationMs,
			TotalFilesAnalyzed: record.TotalFilesAnalyzed,
			ConfigParams:       record.ConfigParams,
		}
	}
	return result
}

// ConvertFileScoresMetricsRecords converts schema.FileScoresMetricsRecord to FileScoresMetrics for Parquet export.
func ConvertFileScoresMetricsRecords(records []schema.FileScoresMetricsRecord) []FileScoresMetrics {
	result := make([]FileScoresMetrics, len(records))
	for i, record := range records {
		result[i] = FileScoresMetrics{
			AnalysisID:             record.AnalysisID,
			FilePath:               record.FilePath,
			AnalysisTime:           record.AnalysisTime,
			TotalCommits:           record.TotalCommits.Float64(),
			TotalChurn:             record.TotalChurn.Float64(),
			LinesOfCode:            record.LinesOfCode.Float64(),
			ContributorCount:       record.ContributorCount.Float64(),
			AgeDays:                record.AgeDays.Float64(),
			RecentLinesAdded:       record.RecentLinesAdded.Float64(),
			RecentLinesDeleted:     record.RecentLinesDeleted.Float64(),
			LinesAdded:             record.LinesAdded.Float64(),
			LinesDeleted:           record.LinesDeleted.Float64(),
			DecayedCommits:         record.DecayedCommits.Float64(),
			DecayedChurn:           record.DecayedChurn.Float64(),
			RecentCommits:          record.RecentCommits.Float64(),
			RecentChurn:            record.RecentChurn.Float64(),
			RecentContributorCount: record.RecentContributorCount.Float64(),
			GiniCoefficient:        record.GiniCoefficient,
			FileOwner:              record.FileOwner,
			ScoreHot:               record.ScoreHot,
			ScoreRisk:              record.ScoreRisk,
			ScoreComplexity:        record.ScoreComplexity,
			ScoreROI:               record.ScoreROI,
			ScoreLabel:             record.ScoreLabel,
			Reasoning:              record.Reasoning,
			RecencySignal:          record.RecencySignal,
			RecencyThresholdLow:    record.RecencyThresholdLow,
			RecencyThresholdHigh:   record.RecencyThresholdHigh,
		}
	}
	return result
}
