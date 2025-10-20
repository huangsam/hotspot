# Hotspot

Hotspot is the Git analyzer that cuts through history to instantly show you which files are your greatest risk.

## Features

- 🔥 **Multiple Scoring Modes** - 8 different perspectives on file importance
- ⚡ **Fast Parallel Analysis** - Concurrent processing with configurable workers
- 📊 **Rich Metrics** - Tracks contributors, commits, churn, size, age, and distribution
- 🎯 **Flexible Filtering** - Path filters and exclude patterns with glob support
- 📅 **Time Range Support** - Analyze specific periods or recent activity windows
- 📝 **Multiple Output Formats** - Human-readable tables or CSV export
- 🔍 **Rename Tracking** - Optional `--follow` mode to track file history through renames
- 🧮 **Detailed Breakdowns** - Explain mode shows per-metric contributions

## Installation

```bash
go install github.com/huangsam/hotspot@latest
```

## Usage

### Basic usage
```bash
# Analyze current directory with default settings
hotspot .

# Analyze specific repository
hotspot /path/to/repo

# Show top 20 files
hotspot -limit 20 .

# Use a specific scoring mode
hotspot -mode risk .
```

### Scoring Modes

The `-mode` flag controls how files are scored and ranked:

| Mode | Purpose | Best For |
|------|---------|----------|
| **hot** (default) | Activity hotspots | Finding where most work happens |
| **risk** | Knowledge/bus factor risk | Identifying single points of failure |
| **complexity** | Technical debt | Finding refactoring candidates |
| **stale** | Maintenance debt | Important but neglected files |

### Filtering & Exclusions

```bash
# Filter by path prefix
hotspot -filter src/api .

# Exclude patterns (comma-separated)
hotspot -exclude "vendor/,node_modules/,*.min.js" .

# Default exclusions: vendor/, node_modules/, third_party/, .min.js, .min.css
```

Exclude patterns support:

- **Globs**: `*.min.js`, `**/*.test.go`
- **Prefixes**: `vendor/`, `node_modules/`
- **Extensions**: `.min.js`, `.map`
- **Substrings**: `generated`

### Explain Mode

With `-explain`, see how each metric contributes to the score:
```
   1  src/core/engine.go                        78.3  High      ...

      Breakdown: contrib=12.4% commits=23.1% size=14.2% age=6.8% churn=19.7% gini=2.1%
```

## Metrics Explained

| Metric | Description |
|--------|-------------|
| **Contributors** | Number of unique authors who modified the file |
| **Commits** | Total number of commits affecting this file |
| **Size** | Current file size in kilobytes |
| **Age** | Days since the file's first commit |
| **Churn** | Total lines added/deleted plus commit count (volatility measure) |
| **Gini** | Gini coefficient (0-1) measuring contributor inequality. Lower = more even distribution |
| **First Commit** | Date of the file's first appearance in Git history |
| **Score** | Computed importance (0-100) based on selected mode |
| **Label** | Criticality level: Critical (≥80), High (≥60), Moderate (≥40), Low (<40) |

## Use Cases

### Daily & Sprint Workflows

```bash
# Current Activity Hotspots
hotspot -mode hot -start 2024-10-01T00:00:00Z -limit 15 .

# Immediate Refactoring Targets
hotspot -mode complexity -limit 20 .
```

### Strategic Risk & Debt Management

```bash
# Bus Factor/Knowledge Risk
hotspot -mode risk -output csv -csv-file bus-factor.csv .

# Maintenance Debt Audit
hotspot -mode stale -exclude "test/,vendor/" .

# Complex Files with History
hotspot -mode complexity -limit 10 -follow .
```

## Performance

- **Typical repo (1k files)**: 2-5 seconds
- **Large repo (10k+ files)**: 15-30 seconds
- **With `-follow` flag**: 2-3x slower (only runs on top N results)
- **With `-start` flag**: Fast single-pass aggregation

Adjust `-workers` based on your CPU cores for optimal performance.

## Tips

1. **Start with defaults**: Run `hotspot .` first to get a baseline
2. **Try different modes**: Each reveals different insights about your codebase
3. **Use time ranges**: `-start` is great for sprint or release analysis
4. **Combine filters**: Use `-filter` and `-exclude` to focus on specific areas
5. **Export for tracking**: Use CSV output to track trends over time
6. **Explain for tuning**: Use `-explain` to understand scoring decisions

## Limitations

- Requires Git repository with commit history
- File renames require `-follow` flag for accurate tracking (slower)
- Very large files (>500KB) or very active files (>500 commits) may saturate metrics
- Recent metrics require `-start` flag and only work within specified time window
