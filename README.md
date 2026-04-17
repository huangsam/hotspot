# Hotspot

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/huangsam/hotspot/ci.yml)](https://github.com/huangsam/hotspot/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/huangsam/hotspot)](https://goreportcard.com/report/github.com/huangsam/hotspot)
[![License](https://img.shields.io/github/license/huangsam/hotspot)](https://github.com/huangsam/hotspot/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/huangsam/hotspot)](https://github.com/huangsam/hotspot/releases/latest)

Hotspot is an agentic intelligence layer and CLI that analyzes Git history to diagnose technical debt and bus factor risk based on developer activity, ownership, and churn patterns.

<img src="./images/demo.gif" alt="Hotspot Demo" width="800px" />

Unlike traditional linters or team-velocity metrics, Hotspot analyzes **development behavior**. It turns Git metadata into high-fidelity signals for technical debt, knowledge silos, and refactoring ROI—empowering both humans and AI agents to make data-driven architecture decisions.

## Features

- 🤖 **Agent-First Intelligence** - Native MCP server for autonomous AI auditing and refactoring.
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
| [csv-parser] | 0.079s / 0.031s | 0.172s / 0.069s | 0.204s / 0.092s |
| [fd] | 0.055s / 0.029s | 0.113s / 0.068s | 0.176s / 0.096s |
| [git] | 0.610s / 0.050s | 1.491s / 0.187s | 2.269s / 0.329s |
| [kubernetes] | 3.436s / 0.127s | 8.083s / 1.357s | 12.650s / 1.208s |

The data shows that Hotspot caches Git analysis results to speed up repeated runs.
