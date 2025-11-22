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
- 📊 **Export results** - save to CSV/JSON/Parquet to track trends and progress
- 🔄 **CI/CD integration** - enforce risk thresholds in pipelines

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

For detailed usage, configuration options, and common workflows, see [USERGUIDE.md](./USERGUIDE.md).

### Live demo

Here's a demo of Hotspot in action:

<img src="./images/demo.gif" alt="Hotspot Demo" width="800px" />

### Real world output

Here is what the tool shows for [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes):

<img src="./images/ranking.png" alt="Hotspot" width="800px" />

This ranking displays the **complexity score** and a colored label based on:

`hotspot files --mode complexity --start 2024-01-01T00:00:00Z --end 2025-01-01T00:00:00Z --workers 16 --follow --exclude 'vendor/,.pb.go'`

## Performance

All measurements use default settings with 14 concurrent workers on a MacBook Pro with an M3 Max chip.

[csv-parser]: https://github.com/vincentlaucsb/csv-parser
[fd]: https://github.com/sharkdp/fd
[git]: https://github.com/git/git
[kubernetes]: https://github.com/kubernetes/kubernetes

### Test repositories

The benchmarks use repositories of varying scales to demonstrate performance characteristics:

| Repository | Language | Scale | Description |
|------------|----------|-------|-------------|
| [csv-parser] | C++ | Small | Focused single-purpose CSV parsing library |
| [fd] | Rust | Medium | Actively maintained CLI file search utility |
| [git] | C | Large | Complex version control system |
| [kubernetes] | Go | Massive | Distributed container orchestration platform |

### Benchmark results

Comprehensive performance benchmarks using [this script](./benchmark/main.go). This shows cold vs warm timings:

| Repository | Files (Cold/Warm) | Compare Files (Cold/Warm) | Timeseries (Cold/Warm) |
|------------|-------------------|---------------------------|------------------------|
| [csv-parser] | 0.037s / 0.013s | 0.127s / 0.035s | 0.114s / 0.045s |
| [fd] | 0.044s / 0.014s | 0.076s / 0.036s | 0.129s / 0.059s |
| [git] | 0.564s / 0.033s | 1.361s / 0.153s | 2.328s / 0.200s |
| [kubernetes] | 3.129s / 0.111s | 6.980s / 1.543s | 11.010s / 0.615s |

The data shows that Hotspot caches Git analysis results to speed up repeated runs. You can configure different backends (SQLite, MySQL, PostgreSQL) for caching and analysis storage. If you need fresh analysis, clear the cache: `hotspot cache clear`
