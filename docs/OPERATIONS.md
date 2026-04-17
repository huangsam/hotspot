# Backend & Operations Guide

This guide covers advanced configuration for persistence backends, analysis tracking, and large-scale repository operations.

## Backend Configuration

Hotspot supports multiple backends for caching Git analysis results and storing analysis data: **SQLite** (default, local), **MySQL**, **PostgreSQL**, or **None** (in-memory only).

### Environment Variables

```bash
export HOTSPOT_CACHE_BACKEND=mysql
export HOTSPOT_CACHE_DB_CONNECT="user:pass@tcp(localhost:3306)/hotspot"
export HOTSPOT_ANALYSIS_BACKEND=postgresql
export HOTSPOT_ANALYSIS_DB_CONNECT="host=localhost port=5432 user=postgres dbname=hotspot"
```

### YAML Configuration

```yaml
cache:
  backend: mysql
  db_connect: "user:pass@tcp(localhost:3306)/hotspot"
analysis:
  backend: postgresql
  db_connect: "host=localhost port=5432 user=postgres dbname=hotspot"
```

## Management Commands

| Command | Description |
|---------|-------------|
| `hotspot cache status` | Check cache backend status |
| `hotspot cache clear` | Clear cached data |
| `hotspot analysis status` | Check analysis backend status |
| `hotspot analysis history` | List historical analysis runs |
| `hotspot analysis clear` | Clear stored analysis runs |
| `hotspot analysis migrate` | Migrate analysis database schema |

## Analysis History & Tracking

The `analysis history` command provides a chronological audit trail of your repository analysis runs.

```bash
hotspot analysis history --analysis-backend sqlite
```

### Output Formats
- **Table**: Clean, high-level summary.
- **Markdown**: Documentation-ready tables.
- **CSV**: Stable schema for automation.
- **JSON**: Full metadata for developers.

## Database Migration

The `analysis migrate` command manages schema updates for analysis backends. Run this after upgrading Hotspot.

```bash
# SQLite (Default)
hotspot analysis migrate --analysis-backend sqlite

# MySQL
hotspot analysis migrate --analysis-backend mysql --analysis-db-connect "user:pass@tcp(localhost:3306)/hotspot"
```

## Data Export (Parquet)

Export analysis data for use with analytics tools like Pandas, DuckDB, or Spark.

```bash
hotspot analysis export --analysis-backend sqlite --output-file mydata
```

Creates:
- `mydata.analysis_runs.parquet`
- `mydata.file_scores_metrics.parquet`
