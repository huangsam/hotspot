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

## Key features

- 🔍 **See what matters** - rank files and folders by activity, complexity, etc.
- ⚡ **Fast results** - analyze thousands of files in seconds
- 🧮 **Rich insights** - contributors, churn, size, age, and risk metrics
- 🎯 **Actionable filters** - narrow down by path or track trends over time
- 🕓 **Robust time windows** - support for human-readable time
- 📊 **Export results** - save to CSV/JSON/Parquet to track trends and progress
- 🔄 **CI/CD integration** - enforce risk thresholds in pipelines

## Target audience

- 🧑‍💻 **Developers** tracking sprint or release activity
- 🧹 **Tech leads** prioritizing refactors and risk across projects
- 🧾 **Managers** monitoring bus factor and maintenance debt

## Installation

### Requirements

- **Go 1.25+** for building from source
- **Git 2.2.0+** for repository analysis

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

### AI Agent Integration (MCP)

Hotspot includes a built-in [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server. This allows AI agents (like Claude Desktop or Cursor) to analyze your repositories directly with full support for filtering and time-anchored trends!

```bash
# Start the MCP server (stdio)
hotspot mcp
```

The MCP tools (`get_files_hotspots`, `compare_hotspots`, etc.) support the same parameters as the CLI, including `repo_path`, `mode`, `limit`, `start`, and `end`.

### Full Documentation

- **[USERGUIDE.md](./USERGUIDE.md)**: Detailed commands, configuration options, backend setup, data exports, and common workflows.
- **[PLAYBOOK.md](./PLAYBOOK.md)**: Actionable guidance on using this data to foster a healthy engineering culture.

### Live demo

Here's a demo of Hotspot in action:

<img src="./images/demo.gif" alt="Hotspot Demo" width="800px" />



## Performance

All measurements are done using 14 concurrent workers on a Macbook Pro with the M3 Max chip.

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
| [csv-parser] | 0.090s / 0.031s | 0.198s / 0.078s | 0.257s / 0.095s |
| [fd] | 0.061s / 0.031s | 0.137s / 0.083s | 0.210s / 0.099s |
| [git] | 0.687s / 0.048s | 1.687s / 0.193s | 2.692s / 0.322s |
| [kubernetes] | 4.071s / 0.113s | 9.053s / 1.431s | 15.429s / 1.113s |

The data shows that Hotspot caches Git analysis results to speed up repeated runs.
