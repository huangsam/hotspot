package iocache

import (
	"errors"
	"fmt"

	"github.com/huangsam/hotspot/internal/parquet"
)

// ExecuteAnalysisExport performs the actual export of analysis data to Parquet files.
func ExecuteAnalysisExport(outputFile string) error {
	// Validate that output file is specified
	if outputFile == "" {
		return errors.New("--output-file is required for export command")
	}

	// Get the analysis store
	store := Manager.GetAnalysisStore()

	// Check if there's any data to export
	status, err := store.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get analysis status: %w", err)
	}

	if status.TotalRuns == 0 {
		return errors.New("no analysis data found to export")
	}

	fmt.Printf("Exporting data from %s backend...\n", status.Backend)
	fmt.Printf("Total analysis runs: %d\n", status.TotalRuns)
	fmt.Printf("Total file records: %d\n", status.TableSizes["hotspot_file_scores_metrics"])

	// Retrieve all analysis runs
	analysisRuns, err := store.GetAllAnalysisRuns()
	if err != nil {
		return fmt.Errorf("failed to retrieve analysis runs: %w", err)
	}

	// Retrieve all file scores metrics
	fileMetrics, err := store.GetAllFileScoresMetrics()
	if err != nil {
		return fmt.Errorf("failed to retrieve file scores metrics: %w", err)
	}

	// Convert to Parquet format
	parquetAnalysisRuns := parquet.ConvertAnalysisRunRecords(analysisRuns)
	parquetFileMetrics := parquet.ConvertFileScoresMetricsRecords(fileMetrics)

	// Write analysis runs to Parquet
	analysisRunsFile := outputFile + ".analysis_runs.parquet"
	if err := parquet.WriteAnalysisRunsParquet(parquetAnalysisRuns, analysisRunsFile); err != nil {
		return fmt.Errorf("failed to write analysis runs: %w", err)
	}
	fmt.Printf("Exported %d analysis runs to: %s\n", len(parquetAnalysisRuns), analysisRunsFile)

	// Write file scores metrics to Parquet
	fileMetricsFile := outputFile + ".file_scores_metrics.parquet"
	if err := parquet.WriteFileScoresMetricsParquet(parquetFileMetrics, fileMetricsFile); err != nil {
		return fmt.Errorf("failed to write file scores metrics: %w", err)
	}
	fmt.Printf("Exported %d file score records to: %s\n", len(parquetFileMetrics), fileMetricsFile)

	fmt.Println("\nExport complete! The Parquet files can be used with:")
	fmt.Println("  - Apache Spark")
	fmt.Println("  - Apache Arrow")
	fmt.Println("  - Pandas (via pyarrow)")
	fmt.Println("  - DuckDB")
	fmt.Println("  - Any other Parquet-compatible tool")

	return nil
}
