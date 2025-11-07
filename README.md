# Hotspot

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/huangsam/hotspot/ci.yml)](https://github.com/huangsam/hotspot/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/huangsam/hotspot)](https://goreportcard.com/report/github.com/huangsam/hotspot)
[![License](https://img.shields.io/github/license/huangsam/hotspot)](https://github.com/huangsam/hotspot/blob/main/LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/huangsam/hotspot)](https://github.com/huangsam/hotspot/releases/latest)

Hotspot cuts through Git history to show you which files and folders are your greatest risk.

<img src="./images/logo.png" alt="Hotspot" width="250px" />

This tool operates as a **tactical, code-level risk finder**. While [DORA] metrics track team performance and [SCA] flags external security issues, Hotspot focuses entirely on diagnosing **technical debt** and **bus factor risk** within **your code** at the file and folder level.

[DORA]: https://en.wikipedia.org/wiki/DevOps_Research_and_Assessment
[SCA]: https://en.wikipedia.org/wiki/Static_program_analysis

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
| **hot** | Activity hotspots | Identify files and subsystems with the most activity. |
| **risk** | Knowledge risk | Find areas with unequal contribution and few owners. |
| **complexity** | Technical debt | Triage files with high churn, large size, and high complexity. |
| **stale** | Maintenance debt | Highlight critical files that are large, old, but rarely touched. |

## Risk comparison & delta tracking

The `compare` subcommand allows you to measure the change in metrics between two different points in your repository's history. This is the most effective way to audit the impact of a new change set across multiple dimensions.

**Example:** Compare between releases, using the default 6-month lookback:

`hotspot compare files --mode complexity --base-ref v0.15.0 --target-ref v0.16.0`

| Flags | Description |
|-------|-------------|
| `--base-ref` | The BEFORE Git reference (e.g., `main`, `v1.0.0`, a commit hash). |
| `--target-ref` | The AFTER Git reference (defaults to `HEAD`). |
| `--lookback` | Time window (e.g. `6 months`) used for base and target. |

## Configuration file

For complex or repetitive commands, Hotspot can read all flags from a configuration file named **`.hotspot.yaml`** or **`.hotspot.yml`** placed in your repository root or home directory.

This allows you to manage settings without long command-line strings. Flags always override file settings.

### Examples

We provide three documented examples in the `examples/` directory to cover common use cases:

1.  [hotspot.basic.yml](./examples/hotspot.basic.yml): Quick setup for local development
2.  [hotspot.ci.yml](./examples/hotspot.ci.yml): Optimized settings for automated CI/CD runs (e.g., JSON output)
3.  [hotspot.docs.yml](./examples/hotspot.docs.yml): The canonical template listing every available setting

## Common use cases

### Daily & Sprint Workflows

```bash
# 1. Identify active subsystems for daily standup/priority setting
hotspot folders --mode hot --start "2 weeks ago"

# 2. Drill down to the specific active files within a subsystem
hotspot files --mode hot ./path/from/folder/hot --start "2 weeks ago"

# 3. Immediate Refactoring Targets (Files with high recent complexity)
hotspot files --mode complexity --start "3 months ago" ./path/from/file/hot
```

### Strategic Risk & Debt Management

```bash
# 1. Bus Factor/Knowledge Risk Audit (Which subsystems lack owners?)
hotspot folders --mode risk --start "1 year ago"

# 2. Maintenance Debt Audit (Which modules are old, large, and neglected?)
hotspot folders --mode stale --start "5 years ago" --exclude "test/,vendor/"

# 3. Structural Bottleneck Audit (Identify the largest, most-churned, core subsystems)
hotspot folders --mode complexity --start "18 months ago"
```

### Change & Release Auditing

```bash
# 1. Measure Release Risk (Did complexity increase in core folders between releases?)
hotspot compare folders --mode complexity --base-ref v1.0.0 --target-ref HEAD

# 2. Audit File-Level Risk Change (Identify individual files where risk score worsened)
hotspot compare files --mode risk --base-ref main --target-ref feature/new-module

# 3. Track Activity Shift (Which subsystems became 'hot' or 'stale' after the merge?)
hotspot compare folders --mode hot --base-ref v0.15.0 --target-ref v0.16.0
```

## Performance

|Size|Duration|
|---|---|
|**Typical repo (1k files)**|2-5 seconds|
|**Large repo (10k+ files)**|15-30 seconds|

These details were measured from running Hotspot over a 6-month window.
