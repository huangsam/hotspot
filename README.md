# Hotspot

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/huangsam/hotspot/ci.yml)](https://github.com/huangsam/hotspot/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/huangsam/hotspot)](https://goreportcard.com/report/github.com/huangsam/hotspot)
[![License](https://img.shields.io/github/license/huangsam/hotspot)](https://github.com/huangsam/hotspot/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/huangsam/hotspot)](https://github.com/huangsam/hotspot/releases/latest)

Hotspot is a CLI tool that analyzes Git history to diagnose technical debt and bus factor risk based on developer activity, ownership, and churn patterns.

<img src="./images/logo.png" alt="Hotspot" width="250px" />

This tool operates as a **data-driven development intelligence.** While traditional [SCA] tools focus on code structure and style, and [DORA] metrics track team performance, Hotspot analyzes **actual development behavior** - commit patterns, ownership distribution, churn trends, and maintenance activity - to diagnose **technical debt** and **bus factor risk** within **your code** at the file and folder level.

[DORA]: https://en.wikipedia.org/wiki/DevOps_Research_and_Assessment
[SCA]: https://en.wikipedia.org/wiki/Static_program_analysis

## Motivation

For years, I've managed projects where everyone *knew* which files were the maintenance nightmares—the ones where a small change led to a two-day debugging session. As engineers, we invest heavily in **Code Correctness** - we run linters, use SCA tools, and write comprehensive unit tests.

However, these traditional QA methods often fail against **System Resilience**. Outages rarely stem from syntax errors; they're caused by code that's too complex, too fragile, or owned by too few people.

Hotspot was born to fix that: providing a transparent, auditable tool for teams to diagnose the **technical debt** and **knowledge risk** that truly drive production instability.

### Key features

- 🔍 **See what matters** - rank files and folders by activity, complexity, etc.
- ⚡ **Fast results** - analyze thousands of files in seconds
- 🧮 **Rich insights** - contributors, churn, size, age, and risk metrics
- 🎯 **Actionable filters** - narrow down by path, exclude noise, or track trends over time
- 📊 **Export results** - save to CSV/JSON to track trends and progress

### Target audience

- 🧑‍💻 **Developers** tracking sprint or release activity
- 🧹 **Tech leads** prioritizing refactors and risk across projects
- 🧾 **Managers** monitoring bus factor and maintenance debt

## Installation

### Requirements

- **Go 1.25+** for building from source
- **Git 2.2.0+** for `--date=iso-strict` support (used for precise timestamp parsing)

### Install from source

```bash
go install github.com/huangsam/hotspot@latest
```

### Download pre-built binary

Visit the [latest release](https://github.com/huangsam/hotspot/releases/latest) and download the `tar` archive for your system (supports **Windows**, **macOS**, and **Linux**), then extract the binary to your `$PATH`.

## Quick start

```bash
# Analyze files for immediate, tactical risk
hotspot files

# Analyze folders for strategic, subsystem risk
hotspot folders

# For an explicit path
hotspot files /path/to/repo/pkg
```

### Live demo

Here's a demo of Hotspot in action:

<img src="./images/demo.gif" alt="Hotspot Demo" width="800px" />

### Real world output

Here is what the tool shows for [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes):

<img src="./images/ranking.png" alt="Hotspot" width="800px" />

This ranking displays the **complexity score** and a colored label based on:

`hotspot files --mode complexity --start 2024-01-01T00:00:00Z --end 2025-01-01T00:00:00Z --workers 16 --follow --exclude 'vendor/,.pb.go'`

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

## Configuration file

For complex or repetitive commands, Hotspot can read all flags from a configuration file named **`.hotspot.yaml`** or **`.hotspot.yml`** placed in your repository root or home directory.

This allows you to manage settings without long command-line strings. Flags always override file settings. We provide four documented examples in the `examples/` directory to cover common use cases:

1.  [hotspot.basic.yml](./examples/hotspot.basic.yml): Quick setup for local development
2.  [hotspot.ci.yml](./examples/hotspot.ci.yml): Optimized settings for automated CI/CD runs (e.g., JSON output)
3.  [hotspot.docs.yml](./examples/hotspot.docs.yml): The canonical template listing every available setting
4.  [hotspot.weights.yml](./examples/hotspot.weights.yml): Advanced customization of scoring algorithm weights

## Common use cases

### Daily & Sprint Workflows

```bash
# Identify active subsystems for daily standup
hotspot folders --mode hot --start "2 weeks ago"

# Drill down to active files in a subsystem
hotspot files --mode hot ./path/from/folder/hot --start "2 weeks ago"
```

### Strategic Risk & Debt Management

```bash
# Bus Factor Audit (subsystems with few owners)
hotspot folders --mode risk --start "1 year ago"

# Maintenance Debt Audit (old, neglected modules)
hotspot folders --mode stale --start "5 years ago" --exclude "test/,vendor/"
```

### Change & Release Auditing

```bash
# Measure release risk changes
hotspot compare folders --mode complexity --base-ref v1.0.0 --target-ref HEAD

# Audit file-level risk changes
hotspot compare files --mode risk --base-ref main --target-ref feature/new-module
```

### Trend Analysis & Historical Tracking

```bash
# Track file complexity over time
hotspot timeseries --path src/main/java/App.java --mode complexity --interval "1 month" --points 6

# Identify when risk started increasing
hotspot timeseries --path lib/legacy.js --mode stale --interval "3 months" --points 8
```

## Performance

All measurements use default settings with 14 concurrent workers on a MacBook Pro with an M3 Max chip.

[csv-parser]: https://github.com/vincentlaucsb/csv-parser
[fd]: https://github.com/sharkdp/fd
[git]: https://github.com/git/git
[kubernetes]: https://github.com/kubernetes/kubernetes

### Test Repositories

The benchmarks use repositories of varying scales to demonstrate performance characteristics:

| Repository | Language | Scale | Description |
|------------|----------|-------|-------------|
| [csv-parser] | C++ | Small | Focused single-purpose CSV parsing library |
| [fd] | Rust | Medium | Actively maintained CLI file search utility |
| [git] | C | Large | Complex version control system |
| [kubernetes] | Go | Massive | Distributed container orchestration platform |

### Benchmark Results

Comprehensive performance benchmarks using [this script](./benchmark/main.go). This shows cold vs warm timings:

| Repository | Files (Cold/Warm) | Compare Files (Cold/Warm) | Timeseries (Cold/Warm) |
|------------|-------------------|---------------------------|------------------------|
| [csv-parser] | 0.037s / 0.012s | 0.127s / 0.035s | 0.118s / 0.042s |
| [fd] | 0.039s / 0.014s | 0.075s / 0.033s | 0.127s / 0.052s |
| [git] | 0.650s / 0.032s | 1.604s / 0.159s | 2.603s / 0.197s |
| [kubernetes] | 3.665s / 0.112s | 8.365s / 1.579s | 13.006s / 0.634s |

Hotspot caches Git analysis results to speed up repeat runs. Here are the benefits:

- Cold runs: Analyzes Git history without caching
- Warm runs: ~35x faster using cached data

If you need fresh analysis, clear the cache: `hotspot cache clear`
