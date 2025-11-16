# 🚀 Changelog

## v1.5.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.4.0...v1.5.0)

### Features

- **Database Cache Backends**: Add MySQL and PostgreSQL support for result caching

### Improvements

- **Analysis Footer**: Make analysis footer more compact
- **Benchmark Script**: Simplify benchmark report generation
- **Documentation**: Update README with new findings

### Development

- **Package Refactor**: Restructure core and internal packages for better organization
- **Test Restoration**: Restore missing tests from recent refactor

## v1.4.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.3.3...v1.4.0)

### Features

- **SQLite Result Caching**: Add persistent caching with ~35x faster analysis for repeated runs and seamless mode switching

### Development

- **Cache Schema**: Bump cache version for compatibility

## v1.3.3

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.3.2...v1.3.3)

### Improvements

- **Documentation**: Add docs for config weights and update timeseries section

### Development

- **Tests**: Add more schema tests

### Fixes

- **Changelog**: Fix accidental header cleanup

## v1.3.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.3.1...v1.3.2)

### Development

- **Test Coverage**: Improve test coverage across core analysis, config validation, and Git client
- **Config Testing**: Add edge case testing for config validation
- **Time Assertions**: Replace brittle time equality with `WithinDuration` checks
- **Documentation**: Add demo GIF and minor README improvements

### Improvements

- **Test Robustness**: Enhance test reliability with paths and time tolerances

## v1.3.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.3.0...v1.3.1)

### Improvements

- **Timeseries Accuracy**: Make timeseries logic more accurate with comprehensive analysis
- **Timeseries UX**: Improve lookback and timeseries user experience with better defaults
- **Timeseries Header**: Fix timeseries header semantics for clearer output
- **Code Organization**: Migrate timeseries constants and refactor analysis logic
- **Documentation**: Add comments to time approximations and update README benchmarks

### Fixes

- **Integration Tests**: Fix integration test issues and update benchmarks

## v1.3.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.2.0...v1.3.0)

### Features

- **Benchmark Script**: Add benchmark script and performance details for analysis

### Fixes

- **Name Abbreviation**: Handle edge cases in name abbreviation logic

### Improvements

- **Documentation**: Update README with motivation section and improved formatting
- **Benchmark Results**: Update and link benchmark tables with latest performance data

## v1.2.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.1.5...v1.2.0)

### Features

- **Metrics Output Formats**: Add CSV and JSON output support to `metrics` command for programmatic access

### Development

- **Integration Tests**: Add integration tests for folders and compare commands
- **Test Performance**: Optimize integration test performance

## v1.1.5

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.1.4...v1.1.5)

### Improvements

- **Timeseries Timestamps**: Add exact timestamps to timeseries output for better precision
- **Terminal Width Detection**: Detect terminal width for improved tabular view formatting

### Fixes

- **JSON Output**: Fix nil owner arrays in JSON output to prevent serialization errors

## v1.1.4

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.1.3...v1.1.4)

### Improvements

- **CSV Data Integrity**: Use pipe delimiters for owner names in all CSV outputs to prevent parsing issues
- **Comparison CSV Enhancement**: Separate before/after owner columns for clearer comparison data

## v1.1.3

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.1.2...v1.1.3)

### Improvements

- **Code Consolidation**: Extract duplicated weight processing logic into shared `ProcessWeightsRawInput` helper, eliminating ~50 lines of duplicated code
- **Function Merging**: Combine `loadActiveWeights` and `ExecuteHotspotMetrics` functions to reduce unnecessary abstraction

## v1.1.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.1.1...v1.1.2)

### Performance

- **Analysis Optimization**: Remove O(N) git calls during file analysis by using pre-aggregated commit data instead of individual `git log` calls per file, significantly improving performance for large repositories

### Fixes

- **Age Calculation**: Fix age verification tests by using relative time windows from aggregated data instead of broken individual git queries
- **Test Consolidation**: Remove duplicate `parseHotspotOutput` function and unify all integration tests to use single `parseHotspotDetailOutput` function with `HotspotFileDetail` struct

### Improvements

- **Integration Test Performance**: Optimize integration tests to build hotspot binary only once instead of 5-6 times, reducing test execution time by 37% (from ~7.4s to ~4.6s)
- **Code Cleanup**: Remove unused `GetFileFirstCommitTime` method and related dead code
- **Git Safety**: Add `hotspot-integration-*/` pattern to `.gitignore` to prevent accidental commits of temporary test directories

## v1.1.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.1.0...v1.1.1)

### Improvements

- **Context Integration**: Add context.Context throughout the codebase for better cancellability and request-scoped data flow
- **Compact Headers**: Make command headers more compact and consistent across all subcommands

## v1.1.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.0.1...v1.1.0)

### Features

- **Ownership Tracking**: Add owner information to comparison and timeseries outputs

### Improvements

- **Schema Consistency**: Add Mode field to all result structs for uniform API
- **JSON Output Uniformity**: Add rank and label fields to folder JSON output
- **Code Cleanup**: Remove unused config parameters from writer functions

## v1.0.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v1.0.0...v1.0.1)

### Fixes

- **Gitignore**: Add .back and .backup to gitignore
- **Output**: Use * for multiply sign in output

## v1.0.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.9.1...v1.0.0)

### Features

- **Custom Scoring Weights**: Add support for custom weights via YAML configuration
- **Metrics Transparency**: Add `metrics` subcommand for viewing scoring formulas
- **Trend Analysis**: Add `timeseries` subcommand for historical score tracking
- **Release Auditing**: Add `compare` subcommand for delta analysis between Git refs
- **Profiling Support**: Add CPU/memory profiling capabilities

### Improvements

- **Score Reporting**: Enable live reporting of score weights
- **Command Output**: Shorten metrics command output
- **Documentation**: Add references to weights in documentation
- **README**: Update README with comprehensive examples and performance data

### Fixes

- **Weight Config**: Fix custom weight config logic
- **Lint Issues**: Fix lint issues with integration tests
- **Tests**: Add tests for custom weight functionality

## v0.9.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.9.0...v0.9.1)

### Improvements

- **Analysis Flow**: Refactor core aggregate analysis flow
- **Features**: Organize features into overarching section
- **README**: Make small tweaks to README examples

## v0.9.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.8.0...v0.9.0)

### Improvements

- **Config File**: Shorten the config file section
- **README**: Update scoring section in README
- **Documentation**: Add timeseries section to README
- **Gitignore**: Consolidate gitignore sections

### Development

- **Integration Tests**: Apply lint and format to integration tests
- **Dependencies**: Update Go dependencies
- **Test Data**: Remove fuzz testdata
- **Core Tests**: Add tests for core aggregation and analysis
- **Documentation**: Add AGENTS.md for improved documentation

## v0.8.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.7.2...v0.8.0)

### Features

- **Timeseries**: Add timeseries subcommand for single path analysis
- **Integration Tests**: Add integration test for timeseries subcommand

### Improvements

- **Structs**: Rename comparison structs to match conventions
- **Syntax**: Modernize syntax in integration tests

## v0.7.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.7.1...v0.7.2)

### Features

- **Profiling**: Add profiling capability to runtime

### Development

- **Fuzz Tests**: Add fuzz tests to codebase
- **Test Setup**: Enhance integration test setup and teardown

## v0.7.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.7.0...v0.7.1)

### Features

- **Multiple Owners**: Show multiple owners for each file

### Improvements

- **Diagnostics**: Add diagnostics info for table view

## v0.7.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.5...v0.7.0)

### Improvements

- **UX**: Update ranking image with new UX
- **Testing**: Add suite of repos for testing
- **Integration Tests**: Adjust integration tests to use JSON output
- **Test Forms**: Add preliminary forms of integration tests

### Fixes

- **Commit Detection**: Fix edge cases in commit detection
- **Age Calculations**: Fix Git age calculations
- **Help Text**: Fix help text inconsistency

## v0.6.5

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.4...v0.6.5)

### Improvements

- **Validation**: Enhance compare command validation
- **Git Logic**: Split Git logic into multiple modules
- **Error Handling**: Funnel all errors into single module
- **Help Text**: Enhance readability of help text
- **Messages**: Refine error-based messages across app

## v0.6.4

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.3...v0.6.4)

### Improvements

- **Performance Table**: Apply formatting to performance table
- **README**: Revise use case sections in README
- **UX**: Update ranking image for better UX
- **Comments**: Add dots to more comments
- **Flag Logic**: Refactor all flag logic for readability
- **Global Flags**: Consolidate global flags for readability

## v0.6.3

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.2...v0.6.3)

### Development

- **Pre-commit**: Add command for lefthook pre-commit
- **Setup**: Refine pre-commit setup
- **Command**: Rename pre-commit command

### Improvements

- **Context**: Add context propagation and tweak folder table
- **Comments**: Fix sequence ID of comment
- **Headers**: Shorten analysis header for brevity

## v0.6.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.1...v0.6.2)

### Build

- **Build Flags**: Add ldflags to build

## v0.6.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.0...v0.6.1)

### Improvements

- **Comments**: Update comments of compare subcommands
- **Code Cleanup**: Remove redundant comment in logic

## v0.6.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.5.0...v0.6.0)

### Improvements

- **Code Cleanup**: Remove extraneous comment from main.go
- **Error Handling**: Add minimal error handling for Viper issues
- **Configuration**: Add sample configs and reference them in README
- **Customization**: Add Viper for streamlined customization

### Features

- **Examples**: Provide concrete example to compare section
- **Terminology**: Lower-case bus-factor term in README
- **Documentation**: Shorten wording on hot mode in README

## v0.5.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.4.4...v0.5.0)

### Development

- **Lefthook**: Adjust lefthook semantics
- **Constants**: Use constants for modes
- **README**: Add differentiator clause in README
- **Tables**: Refine some tables
- **Colors**: Add coloring to table deltas

### Improvements

- **Git Client**: Refactor all Git client touch points
- **Lefthook**: Simplify lefthook to one command
- **Project Setup**: Add lefthook to project

## v0.4.4

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.4.3...v0.4.4)

### Development

- **Dependencies**: Update Go dependencies

### Improvements

- **Compare UX**: Improve UX of compare table

## v0.4.3

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.4.2...v0.4.3)

### Features

- **Compare JSON**: Add file deltas to compare JSON

## v0.4.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.3.0...v0.4.2)

### Improvements

- **Risk Section**: Refine content in risk section
- **Examples**: Revise example to be more realistic

### Fixes

- **Rank Bug**: Fix implicit rank bug due to refactor
- **Compare Logic**: Fix compare setup logic
- **File Comparisons**: Fix edge case with file comparisons
- **Delta Section**: Fix hotspot delta section

## v0.3.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.2.2...v0.3.0)

### Improvements

- **Performance Notes**: Add qualifier about performance measurements
- **Output Section**: Fix wording for output section
- **Examples**: Adjust output and make example repeatable
- **Colors**: Change low color from blue to cyan
- **Makefile**: Consolidate Makefile logic for cleanup

### Development

- **Colors**: Tweak from cyan to blue
- **Terminals**: Adjust colors to work for all terminals
- **Build**: Ensure Makefile applies fix and format

## v0.2.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.2.1...v0.2.2)

### Improvements

- **Assets**: Remove bin gitkeep assets
- **Linting**: Fix linting issues
- **Makefile**: Refactor Makefile for readability
- **Version Command**: Add version command for diagnostics

### Development

- **Aggregation**: Verify and fix aggregation for recent assets
- **Tests**: Replace require with assert usage in test
- **Layers**: Refactor tests into separate layers

## v0.2.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.2.0...v0.2.1)

### Improvements

- **README Option**: Add option C to README
- **Time-ago**: Enable time-ago for --start and --end

## v0.2.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.1.0...v0.2.0)

### Features

- **Format**: Change changelog format
- **Language**: Simplify tips language
- **Documentation**: Add doc references to ago semantics
- **Time-ago**: Add support for time-ago semantics

### Development

- **Guidelines**: Add contribution guidelines
- **Templates**: Add issue templates for GitHub

## v0.1.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.0.0...v0.1.0)

### Build

- **Goreleaser**: Add goreleaser build settings

### Improvements

- **README**: Rewrite punch line to be succinct
- **README Image**: Use text-less image in README
- **README Tips**: Adjust first README tip
- **README Layout**: Fill README image space better
- **README Image**: Adjust README image slightly
- **README Output**: Add sample output to README
- **README Content**: Shuffle README content around image
- **README**: Add image to hotspot README

### Development

- **Tests**: Add tests for computing folder score
- **Unit Tests**: Add unit test for selectOutputFile
- **Colors**: Adjust label colors to match theme
- **Schema**: Add comments to all schema fields
- **UX**: Enhance UX for folder JSON output
- **Owners**: Add owner info to folder results

## Earlier versions

[Full Changelog](https://github.com/huangsam/hotspot/compare/3e250a2...v0.1.0)

### Features

- **Output Modes**: Add CSV and JSON output modes
- **Folder Analysis**: Add support for analyzing folders
- **Benchmarks**: Add ability to run benchmarks
- **Version Command**: Add version command for diagnostics
- **Time-ago**: Add time-ago semantics
- **Configuration**: Add Viper configuration
- **Compare Command**: Add compare subcommand
- **Folder Support**: Add folder analysis support
- **Output Formats**: Add CSV/JSON output
- **Git Client**: Add Git client abstraction
- **CLI Framework**: Add Cobra CLI framework
- **Testing**: Add comprehensive testing
- **CI/CD**: Add CI/CD pipeline
- **Code Quality**: Add linting and formatting
- **Build System**: Add Makefile build system

### Improvements

- **Performance**: Optimize performance and simplify logic
- **Error Handling**: Enhance error handling and logging
- **UX**: Improve UX and output formatting
- **Documentation**: Add comprehensive documentation
- **Code Quality**: Refactor code for maintainability
- **Architecture**: Add proper package structure
