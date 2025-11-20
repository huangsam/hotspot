# Parquet Export Package

This package provides data structures and functions for exporting hotspot analysis data to Apache Parquet files using the `github.com/parquet-go/parquet-go` library.

## Overview

The package defines Go structs that map to the hotspot analysis database tables and provides functions to write these structs to Parquet files with automatic schema inference.

## Quick Start with CLI

The easiest way to export analysis data is using the `hotspot analysis export` command:

```bash
# Run analysis with tracking enabled
hotspot files --analysis-backend sqlite

# Export to Parquet files
hotspot analysis export --analysis-backend sqlite --parquet-file output
```

This creates:
- `output.analysis_runs.parquet` - Analysis run metadata
- `output.file_scores_metrics.parquet` - Per-file metrics and scores

## Data Structures

### AnalysisRun

Maps to the `hotspot_analysis_runs` database table:

```go
type AnalysisRun struct {
    AnalysisID         int64      // Unique identifier for this analysis run
    StartTime          time.Time  // When the analysis began
    EndTime            *time.Time // When the analysis completed (nullable)
    RunDurationMs      *int32     // Duration in milliseconds (nullable)
    TotalFilesAnalyzed int32      // Number of files analyzed
    ConfigParams       *string    // JSON-encoded configuration (nullable)
}
```

### FileScoresMetrics

Maps to the `hotspot_file_scores_metrics` database table:

```go
type FileScoresMetrics struct {
    AnalysisID       int64     // References the parent analysis run
    FilePath         string    // Relative path to the file
    AnalysisTime     time.Time // When this file was analyzed
    TotalCommits     int32     // Number of commits affecting this file
    TotalChurn       int32     // Lines added/deleted
    ContributorCount int32     // Number of unique contributors
    AgeDays          float64   // Age in days since first commit
    GiniCoefficient  float64   // Commit distribution (0-1)
    FileOwner        *string   // Primary owner (nullable)
    ScoreHot         float64   // Hot mode score
    ScoreRisk        float64   // Risk mode score
    ScoreComplexity  float64   // Complexity mode score
    ScoreStale       float64   // Stale mode score
    ScoreLabel       string    // Which scoring mode was used
}
```

## Functions

### WriteAnalysisRunsParquet

Writes a slice of `AnalysisRun` structs to a Parquet file:

```go
func WriteAnalysisRunsParquet(data []AnalysisRun, outputPath string) error
```

**Parameters:**
- `data`: Slice of AnalysisRun structs to write
- `outputPath`: File path where the Parquet file will be written

**Returns:**
- `error`: Any error encountered during file creation or writing

### WriteFileScoresMetricsParquet

Writes a slice of `FileScoresMetrics` structs to a Parquet file:

```go
func WriteFileScoresMetricsParquet(data []FileScoresMetrics, outputPath string) error
```

**Parameters:**
- `data`: Slice of FileScoresMetrics structs to write
- `outputPath`: File path where the Parquet file will be written

**Returns:**
- `error`: Any error encountered during file creation or writing

### Mock Data Functions

For testing and demonstration purposes:

```go
func MockFetchAnalysisRuns() []AnalysisRun
func MockFetchFileScoresMetrics() []FileScoresMetrics
```

## Usage Example

```go
package main

import (
    "log"
    "github.com/huangsam/hotspot/internal/parquet"
)

func main() {
    // Get data (from database, calculation, etc.)
    analysisRuns := parquet.MockFetchAnalysisRuns()
    fileScores := parquet.MockFetchFileScoresMetrics()

    // Export to Parquet files
    if err := parquet.WriteAnalysisRunsParquet(analysisRuns, "analysis_runs.parquet"); err != nil {
        log.Fatal(err)
    }

    if err := parquet.WriteFileScoresMetricsParquet(fileScores, "file_scores.parquet"); err != nil {
        log.Fatal(err)
    }
}
```

See `examples/parquet_export_demo.go` for a complete working example.

## Key Features

- **Struct-based Schema Inference**: Schemas are automatically derived from Go struct tags
- **Nullable Field Support**: Nullable database columns are represented as pointer types in Go
- **Timestamp Precision**: Timestamps are stored with nanosecond precision
- **Compression**: Snappy compression is applied to all columns by default
- **Type Safety**: Compile-time type checking ensures data integrity

## Parquet Schema Details

### AnalysisRun Schema

```
message AnalysisRun {
  required int64 analysis_id (INT(64,true));
  required int64 start_time (TIMESTAMP(isAdjustedToUTC=true,unit=NANOS));
  optional int64 end_time (TIMESTAMP(isAdjustedToUTC=true,unit=NANOS));
  optional int32 run_duration_ms (INT(32,true));
  required int32 total_files_analyzed (INT(32,true));
  optional byte_array config_params (STRING);
}
```

### FileScoresMetrics Schema

```
message FileScoresMetrics {
  required int64 analysis_id (INT(64,true));
  required byte_array file_path (STRING);
  required int64 analysis_time (TIMESTAMP(isAdjustedToUTC=true,unit=NANOS));
  required int32 total_commits (INT(32,true));
  required int32 total_churn (INT(32,true));
  required int32 contributor_count (INT(32,true));
  required double age_days;
  required double gini_coefficient;
  optional byte_array file_owner (STRING);
  required double score_hot;
  required double score_risk;
  required double score_complexity;
  required double score_stale;
  required byte_array score_label (STRING);
}
```

## Reading Parquet Files

The generated Parquet files can be read by:

- **Apache Spark**: `spark.read.parquet("analysis_runs.parquet")`
- **Pandas**: `pd.read_parquet("analysis_runs.parquet")`
- **DuckDB**: `SELECT * FROM 'analysis_runs.parquet'`
- **Apache Arrow**: Using Arrow C++ or Python libraries
- **Go**: Using `parquet.NewGenericReader[AnalysisRun](file)`

## Testing

The package includes comprehensive tests covering:
- Schema validation
- Write/read round-trip verification
- Nullable field handling
- Empty data handling
- Error handling for invalid paths
- Timestamp precision validation

Run tests with:
```bash
go test ./internal/parquet/...
```

## Dependencies

- `github.com/parquet-go/parquet-go` v0.25.1 - Primary Parquet library for Go

This library provides excellent performance and native Go support with automatic schema inference from struct tags.
