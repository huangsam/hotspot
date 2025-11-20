// Package parquet provides data structures and functions for exporting hotspot
// analysis data to Parquet files using github.com/parquet-go/parquet-go.
package parquet

import (
	"fmt"
	"os"
	"time"

	"github.com/parquet-go/parquet-go"
)

// AnalysisRun represents a single hotspot analysis run with metadata.
// This struct maps to the hotspot_analysis_runs database table.
type AnalysisRun struct {
	// AnalysisID is the unique identifier for this analysis run
	AnalysisID int64 `parquet:"analysis_id,snappy"`

	// StartTime is when the analysis began (stored as TIMESTAMP with nanosecond precision)
	StartTime time.Time `parquet:"start_time,snappy"`

	// EndTime is when the analysis completed (nullable, stored as TIMESTAMP with nanosecond precision)
	EndTime *time.Time `parquet:"end_time,optional,snappy"`

	// RunDurationMs is the duration of the analysis run in milliseconds (nullable)
	RunDurationMs *int32 `parquet:"run_duration_ms,optional,snappy"`

	// TotalFilesAnalyzed is the number of files analyzed in this run
	TotalFilesAnalyzed int32 `parquet:"total_files_analyzed,snappy"`

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
	TotalCommits int32 `parquet:"total_commits,snappy"`

	// TotalChurn is the number of lines added/deleted in this file
	TotalChurn int32 `parquet:"total_churn,snappy"`

	// ContributorCount is the number of unique contributors to this file
	ContributorCount int32 `parquet:"contributor_count,snappy"`

	// AgeDays is the age of the file in days since first commit
	AgeDays float64 `parquet:"age_days,snappy"`

	// GiniCoefficient measures commit distribution (0-1, lower is more even)
	GiniCoefficient float64 `parquet:"gini_coefficient,snappy"`

	// FileOwner is the primary owner of the file (nullable)
	FileOwner *string `parquet:"file_owner,optional,snappy"`

	// ScoreHot is the hotspot score in hot mode
	ScoreHot float64 `parquet:"score_hot,snappy"`

	// ScoreRisk is the hotspot score in risk mode
	ScoreRisk float64 `parquet:"score_risk,snappy"`

	// ScoreComplexity is the hotspot score in complexity mode
	ScoreComplexity float64 `parquet:"score_complexity,snappy"`

	// ScoreStale is the hotspot score in stale mode
	ScoreStale float64 `parquet:"score_stale,snappy"`

	// ScoreLabel indicates which scoring mode was used
	ScoreLabel string `parquet:"score_label,snappy"`
}

// WriteAnalysisRunsParquet writes a slice of AnalysisRun structs to a Parquet file.
// This function demonstrates how to use parquet-go for struct-based schema inference.
//
// Parameters:
//   - data: Slice of AnalysisRun structs to write
//   - outputPath: File path where the Parquet file will be written
//
// Returns:
//   - error: Any error encountered during file creation or writing
func WriteAnalysisRunsParquet(data []AnalysisRun, outputPath string) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Create a Parquet writer using struct schema inference
	// The schema is automatically derived from the AnalysisRun struct tags
	writer := parquet.NewGenericWriter[AnalysisRun](file)
	defer writer.Close()

	// Write all records to the file
	// The Write method accepts a variadic slice
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write data to parquet file: %w", err)
	}

	return nil
}

// WriteFileScoresMetricsParquet writes a slice of FileScoresMetrics structs to a Parquet file.
// This function demonstrates how to use parquet-go for struct-based schema inference.
//
// Parameters:
//   - data: Slice of FileScoresMetrics structs to write
//   - outputPath: File path where the Parquet file will be written
//
// Returns:
//   - error: Any error encountered during file creation or writing
func WriteFileScoresMetricsParquet(data []FileScoresMetrics, outputPath string) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Create a Parquet writer using struct schema inference
	// The schema is automatically derived from the FileScoresMetrics struct tags
	writer := parquet.NewGenericWriter[FileScoresMetrics](file)
	defer writer.Close()

	// Write all records to the file
	// The Write method accepts a variadic slice
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write data to parquet file: %w", err)
	}

	return nil
}

// MockFetchAnalysisRuns generates sample AnalysisRun data for demonstration.
// This function shows how nullable fields are mapped to nil pointers.
//
// Returns:
//   - []AnalysisRun: A slice of sample analysis run records
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

	return []AnalysisRun{
		{
			AnalysisID:         1,
			StartTime:          startTime1,
			EndTime:            &endTime1,
			RunDurationMs:      &durationMs1,
			TotalFilesAnalyzed: 150,
			ConfigParams:       &configParams1,
		},
		{
			AnalysisID:         2,
			StartTime:          startTime2,
			EndTime:            &endTime2,
			RunDurationMs:      &durationMs2,
			TotalFilesAnalyzed: 75,
			ConfigParams:       &configParams2,
		},
		{
			AnalysisID:         3,
			StartTime:          startTime3,
			EndTime:            nil, // Still running - nullable field
			RunDurationMs:      nil, // Not yet calculated - nullable field
			TotalFilesAnalyzed: 0,
			ConfigParams:       nil, // No config stored - nullable field
		},
	}
}

// MockFetchFileScoresMetrics generates sample FileScoresMetrics data for demonstration.
// This function shows how nullable fields are mapped to nil pointers.
//
// Returns:
//   - []FileScoresMetrics: A slice of sample file metrics records
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
			ScoreStale:       15.2,
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
			ScoreStale:       25.6,
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
			ScoreStale:       5.7,
			ScoreLabel:       "risk",
		},
	}
}
