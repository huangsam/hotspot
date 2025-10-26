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
hotspot .
```

## Scoring modes

The core power of Hotspot is the `--mode` flag, which defines the ranking algorithm:

| Mode | Focus |
|------|---------|
| **hot** (default) | Activity hotspots |
| **risk** | Knowledge/Bus factor risk |
| **complexity** | Technical debt candidates |
| **stale** | Maintenance debt |

## Common use cases

### Daily & Sprint Workflows

```bash
# Current Activity Hotspots
hotspot --mode hot --start 2024-10-01T00:00:00Z --limit 15

# Immediate Refactoring Targets
hotspot --mode complexity --limit 20
```

### Strategic Risk & Debt Management

```bash
# Bus Factor/Knowledge Risk
hotspot --mode risk --output csv --csv-file bus-factor.csv
hotspot --mode risk --output json --json-file bus-factor.json

# Maintenance Debt Audit
hotspot --mode stale --exclude "test/,vendor/"

# Complex Files with History
hotspot --mode complexity --limit 10 --follow
```

## Performance

|Size|Duration|
|---|---|
|**Typical repo (1k files)**|2-5 seconds|
|**Large repo (10k+ files)**|15-30 seconds|

## Tips

- Start with `hotspot .` for a quick snapshot
- Combine filters and excludes for focus
- Use CSV and JSON exports for tracking trends
- Try `--explain` to see a breakdown of what influenced file rank
- Try `--detail` to see metadata about each file
- Adjust `--workers` based on your CPU cores for optimal performance
