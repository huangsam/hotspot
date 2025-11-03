# Hotspot

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/huangsam/hotspot/ci.yml)](https://github.com/huangsam/hotspot/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/huangsam/hotspot)](https://goreportcard.com/report/github.com/huangsam/hotspot)
[![License](https://img.shields.io/github/license/huangsam/hotspot)](https://github.com/huangsam/hotspot/blob/main/LICENSE)

Hotspot is the Git analyzer that cuts through history to show you which files and folders are your greatest risk.

<img src="./images/logo.png" alt="Hotspot" width="250px" />

Offerred capabilities:

- 🔍 **See what matters** - rank files and folders by activity, complexity, etc.
- ⚡ **Fast results** - analyze thousands of files in seconds
- 🧮 **Rich insights** - contributors, churn, size, age, and risk metrics
- 🎯 **Actionable filters** - narrow down by path, exclude noise, or track trends over time

Target audience:

- 🧑‍💻 **Developers** tracking sprint or release activity
- 🧹 **Tech leads** prioritizing refactors and risk
- 🧾 **Managers** monitoring bus factor and maintenance debt

## Quick start

```bash
go install github.com/huangsam/hotspot@latest

# Analyze files for immediate, tactical risk
hotspot files

# Analyze folders for strategic, subsystem risk
hotspot folders

# For an explicit path
hotspot files /path/to/repo/pkg
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
# Identify active subsystems for daily standup/priority setting
hotspot folders --mode hot

# Check the high-level risk of the target subsystem
hotspot folders --mode risk

# Drill down to the specific active files within the target path
hotspot files --mode hot ./path/from/folder/hot

# Immediate Refactoring Targets (after finding a problem path)
hotspot files --mode complexity --start 2025-01-01T00:00:00Z ./path/from/file/hot
```

### Strategic Risk & Debt Management

```bash
# Bus Factor/Knowledge Risk (Strategic Ownership Audit)
# Identify the subsystems with the highest knowledge concentration
hotspot folders --mode risk --start 2025-01-01T00:00:00Z

# Maintenance Debt Audit (Legacy Subsystem Triage)
# Identify entire modules that have been neglected (old, large, little recent change)
hotspot folders --mode stale --start 2020-01-01T00:00:00Z --exclude "test/,vendor/"

# Structural Bottleneck Audit (Core Complexity)
# Identify the largest, most-churned, core subsystems
hotspot folders --mode complexity --start 2024-01-01T00:00:00Z

# Drill down: Find the high-risk files within the high-risk folders
hotspot files --mode risk --start 2025-01-01T00:00:00Z --path ./path/from/folder/risk
```

## Performance

|Size|Duration|
|---|---|
|**Typical repo (1k files)**|2-5 seconds|
|**Large repo (10k+ files)**|15-30 seconds|

## Tips

- Start with hotspot folders for a high-level strategic overview
- Exclude irrelevant files and folders (`test/`, `vendor/`) to focus the analysis
- Export results as CSV/JSON to track trends and progress
- **Tactical Risk:** Use a 6-month window to identify immediate project risks (hot, risk)
- **Strategic Debt:** Use a 12-24 month window for long-term audits (complexity, stale)
