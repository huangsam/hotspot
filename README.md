# Hotspot

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/huangsam/hotspot/ci.yml)](https://github.com/huangsam/hotspot/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/huangsam/hotspot)](https://goreportcard.com/report/github.com/huangsam/hotspot)
[![License](https://img.shields.io/github/license/huangsam/hotspot)](https://github.com/huangsam/hotspot/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/huangsam/hotspot)](https://github.com/huangsam/hotspot/releases/latest)

Hotspot is an agentic intelligence layer and CLI that analyzes Git history to diagnose technical debt and bus factor risk based on developer activity, ownership, and churn patterns.

<img src="./images/demo.gif" alt="Hotspot Demo" width="800px" />

Unlike traditional linters or team-velocity metrics, Hotspot analyzes **development behavior**. It turns Git metadata into high-fidelity signals for technical debt, knowledge silos, and refactoring ROI—empowering both humans and AI agents to make data-driven architecture decisions.

## Features

- 🤖 **Agentic Hub** - Native MCP server with shape-aware "Reasoning" labels for autonomous auditing.
- 🔍 **Tactical CLI** - Rapid file/folder ranking by activity, complexity, and ownership.
- 🧮 **Deep Metrics** - High-fidelity signals for churn, Ginni coefficients, and bus factor risk.
- 🕓 **Trend Tracking** - Time-anchored analysis and delta tracking across Git references.
- 📊 **Polyglot Exports** - Professional CSV/JSON/Parquet/Markdown reporting.

## Installation

### Requirements

- **Go 1.25+** for building from source
- **Git 2.2+** for repository analysis

### Install from source

```bash
go install github.com/huangsam/hotspot@latest
```

### Download pre-built binary

Visit the [latest release](https://github.com/huangsam/hotspot/releases/latest) and download the `tar` archive for your system (supports **Windows**, **macOS**, and **Linux**), then extract the binary to your `$PATH`.

## Quick start: Choose your path

Hotspot is designed for both human-driven tactical analysis and AI-driven strategic auditing.

### 🤖 Path A: AI Agent

Hotspot includes a **Self-Documenting Agentic Hub** via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). This allows AI agents to autonomously explore your repository.

```bash
# Start the MCP server (stdio)
hotspot mcp
```

### 🔍 Path B: Tactical CLI

For immediate terminal-based checks and CI/CD integration.

```bash
# Initialize hotspot with sensible defaults
hotspot init

# Analyze files for tactical risk
hotspot files

# Analyze folders for strategic subsystems
hotspot folders
```

### Documentation

- **[USERGUIDE.md](./USERGUIDE.md)**: Essential guide for analysis features, scoring modes, and core workflows.
- **[PLAYBOOK.md](./PLAYBOOK.md)**: Actionable guidance on using this data to foster a healthy engineering culture.

## Performance

Hotspot is optimized for speed, even on massive repositories, by caching Git analysis results and using concurrent workers.

[csv-parser]: https://github.com/vincentlaucsb/csv-parser
[fd]: https://github.com/sharkdp/fd
[git]: https://github.com/git/git
[kubernetes]: https://github.com/kubernetes/kubernetes

### Benchmark results

Comprehensive performance benchmarks using [this script](./benchmark/main.go). This shows cold vs warm timings:

| Repository | Files (Cold/Warm) | Compare Files (Cold/Warm) | Timeseries (Cold/Warm) |
|------------|-------------------|---------------------------|------------------------|
| [csv-parser] | 0.080s / 0.029s | 0.147s / 0.052s | 0.172s / 0.073s |
| [fd] | 0.042s / 0.025s | 0.086s / 0.052s | 0.136s / 0.074s |
| [git] | 0.524s / 0.047s | 1.408s / 0.177s | 2.179s / 0.314s |
| [kubernetes] | 3.027s / 0.142s | 7.057s / 1.241s | 12.297s / 1.212s |

The data shows that Hotspot caches Git analysis results to speed up repeated runs.
