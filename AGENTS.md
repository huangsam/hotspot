# Hotspot CLI Agent Documentation

This document provides high-level architectural context and domain concepts for the Hotspot CLI tool to assist AI agents. For detailed implementation, struct definitions, and execution flows, agents should directly read the source code in `cmd/`, `core/`, and `schema/`, as Hotspot uses standard Go CLI patterns (Cobra/Viper) that are easily discoverable.

## Architecture & Data Flow

Hotspot is a Git repository analysis tool that identifies code hotspots through various scoring algorithms. Whether invoked via the traditional CLI or the MCP server, it follows a unified analysis pipeline:

**Analysis Pipeline:**

```
CLI Args (Viper) \                                                     / CLI (Table/CSV/etc.)
                  → Validation → Git Analysis → Scoring → Ranking → Output
MCP Request (URN) /                                                     \ MCP (JSON Response)
```

Hotspot can run as an MCP server (`hotspot mcp`) to expose its analysis capabilities as JSON-RPC tools with full parameter parity. **Critically, all MCP tools now support an optional `urn` parameter** to enable portable repository identity across machines (see Repository URN pattern below).

Agents should provide a `urn` to ensure analysis runs for the same repository are unified in the database, regardless of the local clone path. Note that `repo_path` (defaulting to `.`) is still required to perform fresh Git analysis.

## Self-Discovery & Guided Playbooks

Hotspot is designed to be a "Zero-Config" agentic hub. Agents can autonomously discover documentation and workflows using native MCP capabilities:

**1. Documentation Resources**:
Agents should first read the following resources to understand the repository's context and tool capabilities:
- `hotspot://docs/agents`: This architectural guide (embedded in binary).
- `hotspot://docs/user-guide`: Comprehensive CLI usage and examples (embedded in binary).
- `hotspot://config`: The current repository's `.hotspot.yml` configuration (if present).

**2. Guided Playbooks (Prompts)**:
Agents can use the following standard prompts to execute pre-defined analysis workflows:
- `repository-audit`: A comprehensive health check identifying activity hotspots and knowledge risks.
- `refactor-prioritization`: A specialized guide for using ROI mode to identify high-return refactoring targets.

## Core Domain Concepts

### Scoring Modes

The `core` package implements five distinct scoring algorithms based on different risk assessment principles. This is the most critical domain knowledge:

1. **Hot Mode** (Activity hotspots)
   - **Principle**: Identifies files with high recent activity and volatility.
   - **Focus**: Recent commits, churn, and active development.
   - **Use Case**: Find files currently undergoing active development or significant refactoring.

2. **Risk Mode** (Knowledge risk / bus factor)
   - **Principle**: Identifies files with concentrated ownership and high bus factor risk.
   - **Focus**: Few contributors, uneven ownership distribution, knowledge silos.
   - **Use Case**: Find files that would be problematic to maintain if key contributors leave.

3. **Complexity Mode** (Technical debt candidates)
   - **Principle**: Identifies large, old files with high maintenance burden.
   - **Focus**: File size, age, complexity, and historical churn.
   - **Use Case**: Find files that are expensive to modify or maintain.

4. **Stale Mode** (Maintenance debt)
   - **Principle**: Identifies important files that haven't been touched recently.
   - **Use Case**: Find files that may have accumulated technical debt due to neglect.

5. **ROI Mode** (Refactoring priority)
   - **Principle**: Identifies files where refactoring effort provides the highest technical return.
   - **Focus**: High churn on complex/large legacy files (Technical impact vs. Effort).
   - **Use Case**: Prioritize refactoring targets in a large codebase with limited resources.

## Repository Shape & Preset System

Hotspot includes **shape analysis** (lightweight single-pass aggregation) to characterize repositories and recommend presets.

**Three fixed presets:**
| Preset | Mode | Use Case |
|--------|------|----------|
| **small** | hot | CLI tools, microservices, libraries |
| **large** | roi | Large monorepos with deep histories |
| **infra** | risk | Infrastructure-as-code repositories |

**Workflow:** `hotspot shape` → get recommendation → apply via `--preset <name>` to other commands.

## Key Design Patterns

- **I/O Caching**: Results and analysis are cached using pluggable backends (SQLite, MySQL, PostgreSQL) to dramatically speed up repeated analyses. See `internal/iocache/`.

- **Repository URN (Portable Identity)**: Every analysis run is tagged with a canonical repository identifier (`RepoURN`) of the form `git:host/owner/repo` (resolved from remote origin URL), `local:rootHash` (for local-only repos), or `local:absPath` (fallback). This ensures cache keys and DB records are path-independent and stable across checkout locations, solving multi-machine fragmentation. All MCP tools (`get_files_hotspots`, `get_folders_hotspots`, `compare_hotspots`, `get_timeseries`) now accept an optional `urn` parameter, enabling agents to query by URN alone for fleet-wide querying and enterprise RAG without local path dependencies.

- **Per-Dialect Migrations**: `internal/iocache/migrations/` contains three subdirectories (`sqlite/`, `mysql/`, `postgres/`) with backend-specific SQL files. `MigrateAnalysis` selects the correct subdirectory via `buildSource()`. DDL differs meaningfully across backends (e.g. `AUTOINCREMENT` vs `AUTO_INCREMENT` vs `BIGSERIAL`, `TEXT` vs `DATETIME(6)` vs `TIMESTAMPTZ`). Do not write dialect-agnostic SQL for schema changes — add a file per dialect.

- **Analysis Store Filtering**: `AnalysisStore` supports pagination and URN-based filtering via `schema.AnalysisQueryFilter`. Persistence dialects handle backend-specific variations (e.g., PostgreSQL placeholders vs SQLite/MySQL) internally.

- **Quiet by Default & Telemetry Separation**: To support both human users and machine-readable pipelines (e.g., MCP, CI systems), the tool strictly separates output channels. Payload data (JSON, Parquet, or text tables) MUST go to `stdout`. Human UX contextual headers (`Repo: ..., Range: ...`) MUST go to `stderr`. Diagnostic and progress telemetry (e.g., "Running --follow...") MUST be routed through `internal/logger` (`logger.Info(...)`), remaining silent by default unless the user sets verbose/debug flags. **Never use `fmt.Println` or `fmt.Printf` for progress or status events**, to prevent corrupting structured data on standard out.

- **Single-Pass Performance & AI Signal**: All git analysis (Total vs. Recent, Adds vs. Deletes) MUST happen in a single log pass in `core/agg` to avoid I/O regressions. Maintain raw magnitude metrics; modern AI handles absolute signals and ratios better than pre-normalized values.

- **Enriched AI Signal (Reasoning)**: Analysis results (`FileResult`) include a `Reasoning` slice containing human-and-AI-readable justifications (e.g., "High Churn: Recent volatility...") to assist LLMs and human reviewers in interpreting complex score vectors without manual metric re-calculation.

- **High-Precision Architecture (Metric Type)**: Hotspot has evolved from a discrete integer engine to a continuous signal architecture via the `schema.Metric` type (aliased to `float64`), providing high-precision magnitudes that eliminate "clipping" artifacts during decay or weighting. This enables time-weighted activity (exponential decay with a 180-day half-life) for `hot` and `roi` modes to prioritize current development bottlenecks. By presenting raw, continuous magnitudes rather than coarse-grained integers, the engine provides a more robust signal for AI reasoning and facilitates multi-source ingestion—the "Sponge" architecture—blending "fuzzy" signals from external sources (e.g., JIRA, Slack, sentiment) into a unified scoring vector.

- **Modular Output Provider Pattern**: Output formatting is decoupled from core analysis via the `outwriter.FormatProvider` interface, with specific formats like JSON, CSV, Text, Markdown, Parquet, and Describe implemented as specialized files within the `internal/outwriter/provider/` package. Cross-provider logic such as coloring, table rendering, and metric models is consolidated within the same package to facilitate code reuse and eliminate package circularity. The `internal/outwriter/outwriter.go` registry dispatches calls based on the configured `OutputMode`, and the `FormatProvider` interface should always be used when passing writers through the core orchestration layer.
