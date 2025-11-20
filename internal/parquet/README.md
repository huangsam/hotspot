# Parquet Export Package

This package provides data structures and functions for exporting hotspot analysis data to Apache Parquet files.

## Quick Start

Export analysis data using the CLI:

```bash
# Run analysis with tracking enabled
hotspot files --analysis-backend sqlite

# Export to Parquet files
hotspot analysis export --analysis-backend sqlite --parquet-file output
```

Creates:
- `output.analysis_runs.parquet` - Analysis run metadata
- `output.file_scores_metrics.parquet` - Per-file metrics and scores

## Usage in Code

```go
import "github.com/huangsam/hotspot/internal/parquet"

// Convert and write data
runs := parquet.ConvertAnalysisRunRecords(dbRecords)
parquet.WriteAnalysisRunsParquet(runs, "output.parquet")
```

## Reading Exported Data

The Parquet files work with:
- **Python (Pandas)**: `pd.read_parquet('output.parquet')`
- **DuckDB**: `SELECT * FROM 'output.parquet'`
- **Apache Spark**: `spark.read.parquet("output.parquet")`

See the [User Guide](../../USERGUIDE.md#exporting-to-parquet) for detailed usage examples.
