# Hotspot User Guide

This guide provides detailed documentation for using Hotspot's analysis features, configuration options, and common workflows.

## Analysis features

### Scoring modes

The core power of Hotspot lies in its `--mode` flag, which selects the ranking algorithm used to identify different types of risk.

**Example:** Identify owners of high-risk files.

`hotspot files --mode risk --owner`

| Mode | Focus | Description |
|------|-------|-------------|
| **hot** | Activity hotspots | Identify files and subsystems with the most activity. |
| **risk** | Knowledge risk | Find areas with unequal contribution and few owners. |
| **complexity** | Technical debt | Triage files with high churn, large size, and high complexity. |
| **stale** | Maintenance debt | Highlight critical files that are large, old, but rarely touched. |

### Scoring transparency & customization

The `metrics` command displays the formal mathematical formulas for all scoring modes, showing exactly how files are ranked. When using custom weights from a `.hotspot.yaml` config file, it shows your active configuration.

**Example:** View scoring formulas in action.

`hotspot metrics`

### Risk comparison & delta tracking

The `compare` subcommand allows you to measure the change in metrics between two different points in your repository's history. This is the most effective way to audit the impact of a new change set across multiple dimensions.

**Example:** Compare between releases, using the default 6-month lookback.

`hotspot compare files --mode complexity --base-ref v0.15.0 --target-ref v0.16.0`

| Flags | Description |
|-------|-------------|
| `--base-ref` | The BEFORE Git reference (e.g., `main`, `v1.0.0`, a commit hash). |
| `--target-ref` | The AFTER Git reference (defaults to `HEAD`). |
| `--lookback` | Time window (e.g. `6 months`) used for base and target. |

### Timeseries analysis

The `timeseries` subcommand tracks how hotspot scores change over time for a specific file or folder path. This helps you understand trends and identify when risk started increasing or decreasing.

**Example:** Track complexity score for a specific file over the past 3 months.

`hotspot timeseries --path main.go --mode complexity --interval "30 days" --points 3`

| Flags | Description |
|-------|-------------|
| `--path` | The file or folder path to analyze (required). |
| `--mode` | Scoring mode (hot, risk, complexity, stale). |
| `--interval` | Total time window (e.g., `6 months`, `1 year`). |
| `--points` | Number of data points to generate. |

### CI/CD Policy Enforcement

The `check` command allows you to enforce risk thresholds in CI/CD pipelines, failing builds when files exceed acceptable risk levels. If no thresholds are specified, it defaults to 50.0 for all scoring modes.

**Example:** Use CI configuration for policy enforcement.

`hotspot check`

**Example:** Use CLI overrides for policy enforcement.

`hotspot check --threshold-overrides "hot:75,risk:60,complexity:80,stale:70"`

| Flags | Description |
|-------|-------------|
| `--base-ref` | The BEFORE Git reference (e.g., `main`, `v1.0.0`, a commit hash). |
| `--target-ref` | The AFTER Git reference (defaults to `HEAD`). |
| `--lookback` | Time window (e.g. `6 months`) used for base and target. |
| `--threshold-overrides` | Custom risk thresholds per scoring mode (format: `hot:50,risk:50,complexity:50,stale:50`). |

The [example CI config](./examples/hotspot.ci.yml) shows how custom thresholds can be configured for each scoring mode and is useful for maintaining code quality standards specific to your team.

## Configuration

### Configuration file

For complex or repetitive commands, Hotspot can read all flags from a configuration file named **`.hotspot.yaml`** or **`.hotspot.yml`** placed in your repository root or home directory.

This allows you to manage settings without long command-line strings. Flags always override file settings. We provide four documented examples in the `examples/` directory to cover common use cases:

1.  [hotspot.basic.yml](./examples/hotspot.basic.yml): Quick setup for local development
2.  [hotspot.ci.yml](./examples/hotspot.ci.yml): Optimized settings for CI/CD policy enforcement
3.  [hotspot.docs.yml](./examples/hotspot.docs.yml): The canonical template listing every available setting
4.  [hotspot.weights.yml](./examples/hotspot.weights.yml): Advanced customization of scoring algorithm weights

### Backend configuration

Hotspot supports multiple backends for caching Git analysis results and storing analysis data: **SQLite** (default, local), **MySQL**, **PostgreSQL**, or **None** (in-memory only).

#### Configuration

Set backends via environment variables:

```bash
export HOTSPOT_CACHE_BACKEND=mysql
export HOTSPOT_CACHE_DB_CONNECT="user:pass@tcp(localhost:3306)/hotspot"
export HOTSPOT_ANALYSIS_BACKEND=postgresql
export HOTSPOT_ANALYSIS_DB_CONNECT="host=localhost port=5432 user=postgres dbname=hotspot"
```

Or in `.hotspot.yaml`:

```yaml
cache:
  backend: mysql
  db_connect: "user:pass@tcp(localhost:3306)/hotspot"
analysis:
  backend: postgresql
  db_connect: "host=localhost port=5432 user=postgres dbname=hotspot"
```

#### Management commands

```bash
hotspot cache status    # Check cache backend status
hotspot cache clear     # Clear cached data
hotspot analysis status # Check analysis backend status
hotspot analysis clear  # Clear stored analysis runs
```

### Exporting to Parquet

Export analysis data to Parquet files for use with analytics tools like Spark, Pandas, and DuckDB.

#### Basic export

```bash
# Run analysis with tracking enabled
hotspot files --analysis-backend sqlite

# Export to Parquet files
hotspot analysis export --analysis-backend sqlite --output-file mydata
```

This creates two files:

- `mydata.analysis_runs.parquet` - Analysis run metadata
- `mydata.file_scores_metrics.parquet` - Per-file metrics and scores

#### Using with different backends

SQLite (default):

```bash
hotspot analysis export --analysis-backend sqlite --output-file export/data
```

MySQL:

```bash
hotspot analysis export \
  --analysis-backend mysql \
  --analysis-db-connect "user:pass@tcp(localhost:3306)/hotspot" \
  --output-file export/data
```

PostgreSQL:

```bash
hotspot analysis export \
  --analysis-backend postgresql \
  --analysis-db-connect "host=localhost port=5432 dbname=hotspot" \
  --output-file export/data
```

#### Reading exported data

Python (Pandas):

```python
import pandas as pd

runs = pd.read_parquet('mydata.analysis_runs.parquet')
files = pd.read_parquet('mydata.file_scores_metrics.parquet')

# Analyze trends
print(files.groupby('analysis_id')['score_hot'].mean())
```

DuckDB:

```sql
-- Query Parquet files directly
SELECT * FROM 'mydata.analysis_runs.parquet';
SELECT file_path, score_hot, score_risk
FROM 'mydata.file_scores_metrics.parquet'
ORDER BY score_hot DESC
LIMIT 10;
```

Apache Spark:

```scala
val runs = spark.read.parquet("mydata.analysis_runs.parquet")
val files = spark.read.parquet("mydata.file_scores_metrics.parquet")

files.groupBy("analysis_id").count().show()
```

## Common use cases

### Daily & sprint workflows

```bash
# Identify active subsystems for daily standup
hotspot folders --mode hot --start "2 weeks ago"

# Drill down to active files in a subsystem
hotspot files --mode hot ./path/from/folder/hot --start "2 weeks ago"
```

### Strategic risk & debt management

```bash
# Bus Factor Audit (subsystems with few owners)
hotspot folders --mode risk --start "1 year ago"

# Maintenance Debt Audit (old, neglected modules)
hotspot folders --mode stale --start "5 years ago" --exclude "test/,vendor/"
```

### Change & release auditing

```bash
# Measure release risk changes
hotspot compare folders --mode complexity --base-ref v1.0.0 --target-ref HEAD

# Audit file-level risk changes
hotspot compare files --mode risk --base-ref main --target-ref feature/new-module
```

### Trend analysis & historical tracking

```bash
# Track file complexity over time
hotspot timeseries --path src/main/java/App.java --mode complexity --interval "1 month" --points 6

# Identify when risk started increasing
hotspot timeseries --path lib/legacy.js --mode stale --interval "3 months" --points 8
```
