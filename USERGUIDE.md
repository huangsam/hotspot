# Hotspot User Guide

This guide provides the essential documentation for using Hotspot's analysis features and common workflows.

## Core Philosophy

Hotspot is built on the premise that **System Resilience** and **Team Sustainability** are just as critical as code correctness. While traditional QA tools (linters, unit tests, SCA) catch syntax errors and logic bugs, production outages and development bottlenecks often stem from:
- **High Complexity**: Fragile code that is expensive and risky to modify.
- **Knowledge Silos**: Critical subsystems owned by too few people (low bus factor).
- **Knowledge Decay**: Historically important files that have been abandoned.

Hotspot provides the data-driven signal needed to identify these risks and start the conversations required to fix them.

## Analysis Features

### 1. Repository Shape & Preset Recommendation
The `init` command analyzes your repository and creates a recommended configuration:

```bash
hotspot init            # Run shape analysis and write .hotspot.yml
hotspot shape           # Print shape metrics as JSON
```

### 2. Scoring Modes
The core power of Hotspot lies in its `--mode` flag, which selects the ranking algorithm:

`hotspot files --mode risk --owner`

| Mode | Focus | Description |
|------|-------|-------------|
| **hot** | Activity hotspots | Identify files and subsystems with the most activity. |
| **risk** | Knowledge risk | Find areas with unequal ownership and knowledge decay. |
| **complexity** | Technical debt | Triage files with high churn, large size, and high complexity. |
| **roi** | Refactoring ROI | Prioritize refactoring targets that offer the highest technical return. |

### 3. Risk Comparison & Delta Tracking
Measure the change in metrics between two different points in history.

`hotspot compare files --mode complexity --base-ref v1.17.0 --target-ref v1.18.0`

### 4. Timeseries Analysis
Track how hotspot scores change over time for a specific file or folder path.

`hotspot timeseries --path main.go --mode complexity --interval "30 days" --points 3`

---

## Interpreting Results

Hotspot provides two primary ways to consume analysis data, depending on whether you are looking for strategic insights or tactical details.

### 📊 Visual Heatmaps (Strategic)
The `heatmap` output format transforms raw metrics into a high-fidelity SVG treemap. This is the **Strategic View**, designed to help you instantly identify which subsystems are dominating your technical debt landscape.

```bash
hotspot files --output heatmap --output-file images/heatmap.svg
```

Files are color-coded by risk level and sized by complexity, making "God Objects" and abandoned modules immediately obvious.

### 🔍 The Terminal Experience (Tactical)
The default tabular output is the **Tactical View**. It is optimized for power users who need high-precision data, rapid sorting, and easy integration with other terminal tools.

The terminal output includes:
- **High-Precision Scores**: Continuous magnitudes that eliminate "clipping" artifacts.
- **Reasoning Labels**: Metric-anchored justifications (e.g., "Historical Hotspot") that prevent misinterpreting stale data.
- **Ownership Metrics**: When using `--owner`, identifies knowledge silos directly in the table.

---

## Configuration

### Configuration File
Manage settings without long command-line strings by using a `.hotspot.yaml` file. The easiest way to get started is by using the `init` command, which generates a configuration tailored to your repository:

- `hotspot init --preset small`
- `hotspot init --preset large`
- `hotspot init --preset infra`

For full details on the built-in presets and available configuration options, refer to the [canonical preset definitions](schema/data/presets.yaml) and the [reference configuration template](examples/reference/hotspot.docs.yml).

### Exporting Results
Hotspot supports multiple export formats to assist in reporting:

```bash
hotspot files --output markdown --explain > report.md
hotspot files --output csv --output-file findings.csv
```

---

## Next Steps & Deep Dives

For specialized use cases, please refer to the following guides:

- **[Operations Guide](docs/OPERATIONS.md)**: Database backends, migration, and analysis history tracking.
- **[CI/CD Enforcement](docs/CI.md)**: Using Hotspot to gate builds in your pipeline.
- **[AI & MCP Server](docs/MCP.md)**: Connecting Hotspot to AI agents like Claude or Cursor.
- **[Strategic Playbook](PLAYBOOK.md)**: In-depth recipes for risk auditing and refactoring prioritization.
