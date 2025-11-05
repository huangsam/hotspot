# Hotspot

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/huangsam/hotspot/ci.yml)](https://github.com/huangsam/hotspot/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/huangsam/hotspot)](https://goreportcard.com/report/github.com/huangsam/hotspot)
[![License](https://img.shields.io/github/license/huangsam/hotspot)](https://github.com/huangsam/hotspot/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/huangsam/hotspot)](https://github.com/huangsam/hotspot/releases/latest)

Hotspot cuts through Git history to show you which files and folders are your greatest risk.

<img src="./images/logo.png" alt="Hotspot" width="250px" />

Offerred capabilities:

- 🔍 **See what matters** - rank files and folders by activity, complexity, etc.
- ⚡ **Fast results** - analyze thousands of files in seconds
- 🧮 **Rich insights** - contributors, churn, size, age, and risk metrics
- 🎯 **Actionable filters** - narrow down by path, exclude noise, or track trends over time
- 📊 **Export results** - save to CSV/JSON to track trends and progress

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

### Early testing

1. **Download the Binary:** Visit the [latest release](https://github.com/huangsam/hotspot/releases/latest) and download the `tar` archive for your system
2. **Extract & Install:**
    * **Linux/macOS:** Extract `hotspot` binary to your `$PATH`
    * **Windows:** Extract `hotspot.exe` to a known location
3. **Verify Installation:** Run `hotspot --help` in your terminal

### Example output

Here is what the tool shows for [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes):

<img src="./images/ranking.png" alt="Hotspot" width="768px" />

This ranking displays the **complexity score** and a colored label based on:

`hotspot files --mode complexity --start 2024-01-01T00:00:00Z --end 2025-01-01T00:00:00Z --workers 16`

## Scoring modes

The core power of Hotspot is the `--mode` flag, which defines the ranking algorithm:

| Mode | Focus | Description |
|------|-------|-------------|
| **hot** | Activity hotspots | Where activity is most concentrated. |
| **risk** | Knowledge risk | Targets with contribution inequality and few owners. |
| **complexity** | Technical debt | Large targets with high churn and low recent activity. |
| **stale** | Maintenance debt | Important targets that haven't been modified recently. |

## Risk comparison & delta tracking

The `compare` subcommand allows you to measure the change in metrics between two different points in your repository's history (e.g., a feature branch vs. main). This is the most effective way to audit the impact of a new change set.

**Example:** Compare your current branch against `main`, focusing only on files in the `pkg/auth` module, using a 3-month activity window:

`hotspot compare files --base-ref main --target-ref HEAD --lookback "3 months" ./pkg/auth`

| Flags | Description |
|-------|-------------|
| `--base-ref` | The BEFORE Git reference (e.g., `main`, `v1.0.0`, a commit hash). |
| `--target-ref` | The AFTER Git reference (defaults to `HEAD`). |
| `--lookback` | Time window of activity (e.g. `6 months`) used for calculation. |

## Common use cases

### Daily & Sprint Workflows

```bash
# Identify active subsystems for daily standup/priority setting
hotspot folders --mode hot --start "2 weeks ago"

# Check the high-level risk of the target subsystem
hotspot folders --mode risk --start "1 month ago"

# Drill down to the specific active files within the target path
hotspot files --mode hot ./path/from/folder/hot --start "2 weeks ago"

# Immediate Refactoring Targets (after finding a problem path)
hotspot files --mode complexity --start "3 months ago" ./path/from/file/hot
```

### Strategic Risk & Debt Management

```bash
# Bus Factor/Knowledge Risk (Strategic Ownership Audit)
# Identify the subsystems with the highest knowledge concentration

# Option A (Precise Audit): Use ISO 8601 for a fixed, auditable period
hotspot folders --mode risk --start 2024-01-01T00:00:00Z

# Option B (Rolling Audit): Use natural language for a rolling window
hotspot folders --mode risk --start "1 year ago"

# Option C (Historical Window Audit): Define a precise range using start and end
hotspot folders --mode risk --start 2024-01-01T00:00:00Z --end "1 month ago"

# Maintenance Debt Audit (Legacy Subsystem Triage)
# Identify entire modules that have been neglected (old, large, little recent change)
hotspot folders --mode stale --start "5 years ago" --exclude "test/,vendor/"

# Structural Bottleneck Audit (Core Complexity)
# Identify the largest, most-churned, core subsystems
hotspot folders --mode complexity --start "18 months ago"

# Drill down: Find the high-risk files within the high-risk folders
hotspot files --mode risk --start "1 year ago" --path ./path/from/folder/risk
```

## Performance

|Size|Duration|
|---|---|
|**Typical repo (1k files)**|2-5 seconds|
|**Large repo (10k+ files)**|15-30 seconds|

These details were measured from running Hotspot over a 6-month window.
