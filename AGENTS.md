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
- **Scoring Modes**: Four distinct scoring algorithms (hot, risk, complexity, stale)

### Data Flow

```
CLI Args/Config → Validation → Git Analysis → Scoring → Ranking → Output
```

## Package Structure

```
hotspot/
├── main.go             # CLI entry point and command definitions
├── core/               # Core analysis logic and algorithms
│   ├── agg.go          # Git activity aggregation and filtering
│   ├── analysis.go     # Git analysis pipeline and file processing
│   ├── core.go         # Main execution functions
│   ├── score.go        # Scoring algorithms and metrics
│   ├── rank.go         # Ranking and sorting logic
│   ├── builder.go      # File metrics builder pattern
│   └── comparison.go   # Comparison analysis logic
├── schema/             # Data structures and constants
│   ├── schema.go       # Core data models
│   └── constants.go    # Scoring modes and output formats
└── internal/           # Internal utilities and helpers
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

The core package contains the main analysis algorithms, scoring logic, and execution functions.

### Main Execution Functions

#### ExecuteHotspotFiles

**Purpose**: Performs file-level hotspot analysis with optional rename tracking.

**Key Steps**:
1. Aggregate Git activity data
2. Filter and build file list
3. Analyze files concurrently
4. Rank results by score
5. Optional follow pass for renames
6. Format and output results

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
- **Computed Values**: Score, breakdown, owners

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

#### AggregateOutput

Raw aggregated data from Git analysis, used as input for file-level analysis.

### Constants

#### Scoring Modes

Four distinct scoring algorithms: hot (activity hotspots), risk (knowledge risk/bus factor), complexity (technical debt candidates), and stale (maintenance debt).

#### Output Formats

Three output formats supported: text (table), CSV, and JSON.

#### Breakdown Keys

Keys used in scoring breakdown to show contribution of each metric component.

## Key Design Patterns

### 1. Builder Pattern

Used for complex file analysis with method chaining to construct complex file metrics.

### 2. Worker Pool Pattern

Concurrent processing of files using goroutines and channels for parallel analysis.

### 3. Configuration Cloning

Time-window specific configurations are created by cloning base config for isolated analysis runs.

### 4. Executor Function Interface

Common interface for different analysis modes to enable consistent execution patterns.

## Command Flow Examples

### Files Analysis

```
CLI: hotspot files --mode hot --limit 10
↓
main.go: sharedSetup() → validation → config population
↓
core.ExecuteHotspotFiles() → runSingleAnalysisCore()
↓
analysis.go: aggregateActivity() → buildFilteredFileList() → analyzeRepo()
↓
score.go: computeScore() for each file
↓
rank.go: rankFiles() by score
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

This documentation provides a comprehensive overview of the hotspot CLI's core architecture, focusing on the main.go, core, and schema packages. The design emphasizes concurrent processing, flexible scoring modes, and clean separation of concerns between data aggregation, analysis, and presentation.
