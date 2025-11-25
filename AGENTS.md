# Hotspot CLI Agent Documentation

This document provides comprehensive documentation for the Hotspot CLI tool, specifically focusing on the `main.go`, `core`, and `schema` packages. This documentation is designed to be easily consumable by other LLMs for understanding the codebase structure, functionality, and implementation patterns.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Package Structure](#package-structure)
3. [Main Package (main.go)](#main-package-maingo)
4. [Core Package](#core-package)
5. [Schema Package](#schema-package)
6. [Key Design Patterns](#key-design-patterns)
7. [Command Flow Examples](#command-flow-examples)

## Architecture Overview

Hotspot is a Git repository analysis CLI tool that identifies code hotspots through various scoring modes. The tool analyzes Git history, file metrics, and contributor patterns to rank files and folders by maintenance burden and risk.

### Core Components

- **CLI Framework**: Cobra-based command structure with Viper configuration management
- **Analysis Engine**: Multi-threaded Git analysis with configurable scoring algorithms
- **Output System**: Multi-format output (text, JSON, CSV) with customizable formatting
- **Backend Support**: Configurable backends for caching and analysis storage (SQLite, MySQL, PostgreSQL, None)
- **Scoring Modes**: Four distinct scoring algorithms (hot, risk, complexity, stale)

### Data Flow

```
CLI Args/Config → Validation → Git Analysis → Scoring → Ranking → Output
```

## Package Structure

```
hotspot/
├── benchmark/          # Benchmarking tools and performance testing
├── core/               # Core analysis logic and algorithms
│   ├── agg/            # Git activity aggregation and caching
│   └── algo/           # Numerical algorithms for scoring and ranking
├── integration/        # Integration tests and test repositories
├── schema/             # Data structures and constants
└── internal/           # Internal utilities and helpers
    ├── contract/       # Configuration, Git client interfaces, and utilities
    ├── iocache/        # I/O caching and analysis storage with backend support
    ├── outwriter/      # Output formatting and writing
    └── parquet/        # Parquet file handling utilities
```

## Main Package (main.go)

The main package serves as the CLI entry point, defining commands, flags, and configuration management.

### Key Components

#### Root Command Structure

The root command defines the base CLI structure with help text and default behavior.

#### Command Hierarchy

- `rootCmd`: Base command with global flags
  - `filesCmd`: File-level analysis (`hotspot files [repo-path]`)
  - `foldersCmd`: Folder-level analysis (`hotspot folders [repo-path]`)
  - `compareCmd`: Comparison analysis
    - `compareFilesCmd`: Compare file metrics between Git refs
    - `compareFoldersCmd`: Compare folder metrics between Git refs
  - `timeseriesCmd`: Timeseries analysis for specific paths
  - `checkCmd`: CI/CD policy enforcement (`hotspot check [repo-path]`)
  - `metricsCmd`: Display formal definitions of all scoring modes (`hotspot metrics`)
  - `cacheCmd`: Cache management
    - `cacheStatusCmd`: Show cache status (`hotspot cache status`)
    - `cacheClearCmd`: Clear cache data (`hotspot cache clear`)
  - `analysisCmd`: Analysis management
    - `analysisStatusCmd`: Show analysis status (`hotspot analysis status`)
    - `analysisClearCmd`: Clear analysis data (`hotspot analysis clear`)
    - `analysisExportCmd`: Export analysis data to Parquet files (`hotspot analysis export`)
    - `analysisMigrateCmd`: Run database schema migrations (`hotspot analysis migrate`)
  - `versionCmd`: Version information

#### Configuration Management

The main package uses Viper for configuration management with multiple sources:

1. **Command-line flags** (highest priority)
2. **Environment variables** (prefixed with `HOTSPOT_`)
3. **Config file** (`.hotspot.yaml` in current/home directory)
4. **Defaults** (lowest priority)

#### Flag Definitions

Global flags available on all commands include scoring mode, output format, and result limit. Timeseries-specific flags include path, interval, and number of points.

#### Execution Flow

1. **PreRunE**: `sharedSetupWrapper` validates config and initializes Git client
2. **Run**: Command-specific execution logic calls core functions
3. **Post-processing**: Results are formatted and output via internal package

## Core Package

The core package contains the main analysis algorithms, scoring logic, and execution functions. Aggregation logic has been separated into the `core/agg` subpackage for better organization. Numerical algorithms for scoring and ranking have been moved to the `core/algo` subpackage.

### Main Execution Functions

#### ExecuteHotspotCheck

**Purpose**: Runs the check command for CI/CD gating with configurable thresholds.

**Key Features**:

- Validates hotspot scores against user-defined thresholds
- Supports all scoring modes (hot, risk, complexity, stale)
- Provides informative output for success/failure
- Designed for integration into CI/CD pipelines

#### ExecuteHotspotCompare

**Purpose**: Runs two file-level analyses (Base and Target) based on Git references and computes the delta results.

**Key Steps**:

1. Analyze base reference files
2. Analyze target reference files
3. Compare results between references
4. Calculate deltas and ranking changes
5. Format and output comparison results

#### ExecuteHotspotCompareFolders

**Purpose**: Runs two folder-level analyses (Base and Target) based on Git references and computes the delta results.

**Key Steps**:

1. Analyze base reference files and aggregate to folders
2. Analyze target reference files and aggregate to folders
3. Compare folder results between references
4. Calculate deltas and ranking changes
5. Format and output comparison results

#### ExecuteHotspotFiles

**Purpose**: Performs file-level hotspot analysis with optional rename tracking.

**Key Steps**:

1. Aggregate Git activity data
2. Filter and build file list
3. Analyze files concurrently
4. Rank results by score
5. Optional follow pass for renames
6. Format and output results

#### ExecuteHotspotFolders

**Purpose**: Performs folder-level hotspot analysis and prints results to stdout.

**Key Steps**:

1. Aggregate Git activity data
2. Filter and build file list
3. Analyze files concurrently
4. Aggregate results to folder level
5. Rank folders by score
6. Format and output results

#### ExecuteHotspotMetrics

**Purpose**: Displays the formal definitions of all scoring modes.

**Key Features**:
- Static display that does not require Git analysis
- Shows purpose, factors, and mathematical formulas for all scoring modes
- Includes custom weights when configured

#### ExecuteHotspotTimeseries

**Purpose**: Analyzes hotspot scores over time for a specific file or folder path.

**Key Features**:

- Disjoint time windows spanning total interval
- Path-specific score extraction
- Supports both files and folders
- Multiple scoring modes

### Analysis Pipeline

#### runSingleAnalysisCore

Performs the common Aggregation, Filtering, and Analysis steps for single analysis modes.

#### Concurrent Analysis (analyzeRepo)

Processes all files in parallel using a worker pool pattern with goroutines and channels.

### Scoring System

#### Scoring Modes

The core package implements four scoring algorithms based on different risk assessment principles:

1. **Hot Mode** (default): Activity hotspots
   - **Principle**: Identifies files with high recent activity and volatility
   - **Focus**: Recent commits, churn, and active development
   - **Use Case**: Find files currently undergoing active development or refactoring

2. **Risk Mode**: Knowledge risk/bus factor
   - **Principle**: Identifies files with concentrated ownership and high bus factor risk
   - **Focus**: Few contributors, uneven ownership distribution, knowledge silos
   - **Use Case**: Find files that would be problematic if key contributors leave

3. **Complexity Mode**: Technical debt candidates
   - **Principle**: Identifies large, old files with high maintenance burden
   - **Focus**: File size, age, complexity, and change difficulty
   - **Use Case**: Find files that are expensive to modify or maintain

4. **Stale Mode**: Maintenance debt
   - **Principle**: Identifies important files that haven't been touched recently
   - **Focus**: Lack of recent activity on historically important files
   - **Use Case**: Find files that may have accumulated technical debt due to neglect

#### Score Calculation

Calculates a file's importance score (0-100) based on normalized metrics and mode-specific weighting. Applies debuffs for test/config files.

### Builder Pattern

The core package uses a builder pattern for file analysis with method chaining to construct complex file metrics.

## Schema Package

The schema package defines all data structures and constants used throughout the application.

### Core Data Structures

#### FileResult

Contains all metrics and computed scores for a single file.

**Key Fields**:

- **Git Metrics**: Commits, contributors, churn, Gini coefficient
- **File Metrics**: Size, lines of code, age
- **Computed Values**: Score, breakdown, owners, mode

#### FolderResult

Aggregated metrics for folders, computed as weighted average of contained files.

#### TimeseriesResult

Time-series data for tracking hotspot scores over time.

**Key Fields**:

- **Points**: Array of TimeseriesPoint objects containing the time-series data

#### TimeseriesPoint

Time-series data point representing a single measurement in the timeseries analysis.

**Key Fields**:

- **Period**: Time period label (e.g., "Current (30d)", "30d to 60d Ago")
- **Score**: Computed hotspot score for this time period
- **Path**: File or folder path being analyzed
- **Owners**: Top owners for this time period (may be empty for periods with no activity)
- **Mode**: Scoring mode used (hot, risk, complexity, stale)

#### AggregateOutput

Raw aggregated data from Git analysis, used as input for file-level analysis.

### Constants

#### Scoring Modes

Four distinct scoring algorithms: hot (activity hotspots), risk (knowledge risk/bus factor), complexity (technical debt candidates), and stale (maintenance debt).

#### Output Formats

Three output formats supported: text (table), CSV, and JSON.

#### Breakdown Keys

Keys used in scoring breakdown to show contribution of each metric component.

## Internal Package

The internal package has been restructured into focused subpackages for better organization:

### contract/

Contains configuration management, Git client interfaces, time utilities, and general-purpose helpers.

### iocache/

Implements I/O caching and analysis tracking functionality with support for multiple database backends (SQLite, MySQL, PostgreSQL).

### outwriter/

Handles output formatting and writing for different formats (text tables, JSON, CSV) and analysis types (files, folders, comparisons, timeseries).

### parquet/

Parquet file handling utilities.

## Key Design Patterns

### 1. Builder Pattern

Used for complex file analysis with method chaining to construct complex file metrics.

### 2. Worker Pool Pattern

Concurrent processing of files using goroutines and channels for parallel analysis.

### 3. Configuration Cloning

Time-window specific configurations are created by cloning base config for isolated analysis runs.

### 4. Executor Function Interface

Common interface for different analysis modes to enable consistent execution patterns.

## Development Workflow

The project uses a comprehensive Makefile to ensure reproducible builds, consistent testing, and standardized development workflows. Always use the Makefile targets instead of running `go` commands directly to maintain consistency across the team.

### Building

```bash
# Build and install globally (useful for system-wide testing)
make reinstall

# Clean and rebuild binary only (faster for local development)
make clean build
```

### Testing

```bash
# Run unit tests (fast, cached)
make test

# Run all tests including integration (comprehensive)
make test-all

# Force fresh test run (bypass cache)
make test FORCE=1
```

### Code Quality

```bash
# Format and lint code
make format

# Run all checks (format + lint + test)
make check

# Most thorough check (includes integration tests, bypasses cache)
make check FORCE=1 INTEGRATION=1
```

### Development Tips

1. **Use `make reinstall`** for development builds (cleans, builds, and installs globally)
2. **Use `make test`** for quick feedback, `make test-all` before commits
3. **Run `make check`** before pushing changes, `make check FORCE=1 INTEGRATION=1` for thorough validation
4. **Set `FORCE=1`** when tests seem to pass unexpectedly (cache issues)

### Binary Execution

For consistent benchmarking and profiling, use the built binary rather than `go run`:

```bash
# Build and run
make build
./bin/hotspot --help
```

## Command Flow Examples

### Files Analysis

```
CLI: hotspot files --mode hot --limit 10
↓
main.go: sharedSetup() → validation → config population
↓
core.ExecuteHotspotFiles() → runSingleAnalysisCore()
↓
core/agg: CachedAggregateActivity() → BuildFilteredFileList() → analyzeRepo()
↓
core/algo: ComputeScore() for each file
↓
core/algo: RankFiles() by score
↓
internal: PrintFileResults() → table/json/csv output
```

### Timeseries Analysis

```
CLI: hotspot timeseries --path src/main.go --interval 180d --points 4
↓
main.go: sharedSetup() → parameter validation
↓
core.ExecuteHotspotTimeseries() → time window calculation
↓
Loop over N time windows:
  cfgWindow := cfg.CloneWithTimeWindow(start, end)
  runSingleAnalysisCore() → find score for specific path
↓
internal: PrintTimeseriesResults() → period/score/mode/path output
```

### Comparison Analysis

```
CLI: hotspot compare files --base-ref main --target-ref feature --lookback "6 months"
↓
main.go: sharedSetup() → compare mode validation
↓
core.ExecuteHotspotCompare() → runCompareAnalysisForRef() for base and target
↓
comparison.go: compareFileResults() → calculate deltas
↓
internal: PrintComparisonResults() → before/after/delta output
```

### Check Analysis

```
CLI: hotspot check --thresholds-override "hot:50,risk:50"
↓
main.go: sharedSetup() → threshold validation
↓
core.ExecuteHotspotCheck() → runSingleAnalysisCore() → validate against thresholds
↓
Output success/failure with details for CI/CD gating
```

This documentation provides a comprehensive overview of the hotspot CLI's core architecture, focusing on the main.go, core, and schema packages. The design emphasizes concurrent processing, flexible scoring modes, and clean separation of concerns between data aggregation, analysis, and presentation.
