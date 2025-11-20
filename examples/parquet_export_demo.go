// Package main demonstrates how to use the internal/parquet package
// to export hotspot analysis data to Parquet files.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/huangsam/hotspot/internal/parquet"
)

func main() {
	// Example 1: Export analysis runs to Parquet
	fmt.Println("=== Example 1: Export Analysis Runs ===")
	analysisRuns := parquet.MockFetchAnalysisRuns()
	fmt.Printf("Fetched %d analysis runs\n", len(analysisRuns))

	outputPath1 := "/tmp/analysis_runs_demo.parquet"
	if err := parquet.WriteAnalysisRunsParquet(analysisRuns, outputPath1); err != nil {
		log.Fatalf("Failed to write analysis runs: %v", err)
	}

	info1, _ := os.Stat(outputPath1)
	fmt.Printf("Successfully wrote %d analysis runs to %s (size: %d bytes)\n\n",
		len(analysisRuns), outputPath1, info1.Size())

	// Example 2: Export file scores and metrics to Parquet
	fmt.Println("=== Example 2: Export File Scores and Metrics ===")
	fileScores := parquet.MockFetchFileScoresMetrics()
	fmt.Printf("Fetched %d file score records\n", len(fileScores))

	outputPath2 := "/tmp/file_scores_demo.parquet"
	if err := parquet.WriteFileScoresMetricsParquet(fileScores, outputPath2); err != nil {
		log.Fatalf("Failed to write file scores: %v", err)
	}

	info2, _ := os.Stat(outputPath2)
	fmt.Printf("Successfully wrote %d file score records to %s (size: %d bytes)\n\n",
		len(fileScores), outputPath2, info2.Size())

	fmt.Println("=== Demo Complete ===")
	fmt.Println("The Parquet files can now be used with:")
	fmt.Println("  - Apache Spark")
	fmt.Println("  - Apache Arrow")
	fmt.Println("  - Pandas (via pyarrow)")
	fmt.Println("  - DuckDB")
	fmt.Println("  - Any other Parquet-compatible tool")
}
