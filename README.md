# Hotspot

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/huangsam/hotspot/ci.yml)](https://github.com/huangsam/hotspot/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/huangsam/hotspot)](https://goreportcard.com/report/github.com/huangsam/hotspot)
[![License](https://img.shields.io/github/license/huangsam/hotspot)](https://github.com/huangsam/hotspot/blob/main/LICENSE)

Hotspot is the Git analyzer that cuts through history to show you which files are your greatest risk.

- 🔍 **See what matters** - rank files by activity, ownership, or complexity
- ⚡ **Fast results** - analyze thousands of files in seconds
- 🧮 **Rich insights** - contributors, churn, size, age, and risk metrics
- 🎯 **Actionable filters** - narrow down by folder, exclude noise, or track trends over time

Perfect for:

- 🧑‍💻 **Developers** tracking sprint or release activity
- 🧹 **Tech leads** prioritizing refactors and risk
- 🧾 **Managers** monitoring bus factor and maintenance debt

## Quick start

```bash
go install github.com/huangsam/hotspot@latest

# For current path
hotspot files

# For explicit path
hotspot files /path/to/repo
```

## Scoring modes

The core power of Hotspot is the `--mode` flag, which defines the ranking algorithm:

| Mode | Focus | Description |
|------|---------|-------------|
| **hot** | Activity hotspots | Where activity is most concentrated. |
| **risk** | Knowledge risk | Files with contribution inequality and few owners. |
| **complexity** | Technical debt | Large files with high churn and low recent activity. |
| **stale** | Maintenance debt | Important files that haven't been modified recently. |

## Common use cases

### Daily & Sprint Workflows

```bash
# Current Activity Hotspots
hotspot files --mode hot

# Immediate Refactoring Targets
# After finding a problem path from the 'hot' mode, analyze its complexity
hotspot files --mode complexity --start 2025-01-01T00:00:00Z ./executors/kubernetes
```

### Strategic Risk & Debt Management

```bash
# Bus Factor/Knowledge Risk
hotspot files --mode risk --start 2025-01-01T00:00:00Z --output csv
hotspot files --mode risk --start 2025-01-01T00:00:00Z --output json

# Maintenance Debt Audit
hotspot files --mode stale --start 2020-01-01T00:00:00Z --exclude "test/,vendor/"

# Complex Files with History
hotspot files --mode complexity --start 2024-01-01T00:00:00Z --limit 50 --follow
```

## Performance

|Size|Duration|
|---|---|
|**Typical repo (1k files)**|2-5 seconds|
|**Large repo (10k+ files)**|15-30 seconds|

## Tips

- Start with `hotspot files` for a quick snapshot
- Exclude irrelevant files and folders to focus the analysis
- Export results as CSV/JSON to track trends and progress
- Choose a 6-month window and 25 results to identify tactical risks
- Choose a 12-month window and 50 results to identify strategic risks
- Choose a 24-month window and 100 results to identify audit risks
