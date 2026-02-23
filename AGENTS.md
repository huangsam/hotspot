# Hotspot CLI Agent Documentation

This document provides high-level architectural context and domain concepts for the Hotspot CLI tool to assist AI agents. For detailed implementation, struct definitions, and execution flows, agents should directly read the source code in `cmd/`, `core/`, and `schema/`, as Hotspot uses standard Go CLI patterns (Cobra/Viper) that are easily discoverable.

## Architecture & Data Flow

Hotspot is a Git repository analysis CLI tool that identifies code hotspots through various scoring algorithms.

**Standard Flow:**

```
CLI Args/Config (Viper) → Validation → Git Analysis (Concurrent) → Scoring → Ranking → Output
```

**MCP (Model Context Protocol) Flow:**

Hotspot can run as an MCP server (`hotspot mcp`) to expose its analysis capabilities as JSON-RPC tools (`get_files_hotspots`, `compare_hotspots`, etc.) directly to LLMs.

```
MCP Request → Tool Handler → Git Analysis → Schema Enrichment → JSON Response
```

## Core Domain Concepts

### Scoring Modes

The `core` package implements four distinct scoring algorithms based on different risk assessment principles. This is the most critical domain knowledge:

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
   - **Focus**: Lack of recent activity on historically important files.
   - **Use Case**: Find files that may have accumulated technical debt due to neglect.

## Key Design Patterns

- **I/O Caching**: Results and analysis are cached using pluggable backends (SQLite, MySQL, PostgreSQL) to dramatically speed up repeated analyses. See `internal/iocache/`.
