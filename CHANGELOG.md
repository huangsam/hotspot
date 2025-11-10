# 🚀 Changelog

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

- Add .back and .backup to gitignore
- Use * for multiply sign in output

## v1.0.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.9.1...v1.0.0)

### Features

- **Custom Scoring Weights**: Add support for custom weights via YAML configuration
- **Metrics Transparency**: Add `metrics` subcommand for viewing scoring formulas
- **Trend Analysis**: Add `timeseries` subcommand for historical score tracking
- **Release Auditing**: Add `compare` subcommand for delta analysis between Git refs
- **Profiling Support**: Add CPU/memory profiling capabilities

### Improvements

- Enable live reporting of score weights
- Shorten metrics command output
- Add references to weights in documentation
- Update README with comprehensive examples and performance data

### Fixes

- Fix custom weight config logic
- Fix lint issues with integration tests
- Add tests for custom weight functionality

## v0.9.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.9.0...v0.9.1)

### Improvements

- Refactor core aggregate analysis flow
- Organize features into overarching section
- Make small tweaks to README examples

## v0.9.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.8.0...v0.9.0)

### Improvements

- Shorten the config file section
- Update scoring section in README
- Add timeseries section to README
- Consolidate gitignore sections

### Development

- Apply lint and format to integration tests
- Update Go dependencies
- Remove fuzz testdata
- Add tests for core aggregation and analysis
- Add AGENTS.md for improved documentation

## v0.8.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.7.2...v0.8.0)

### Features

- Add timeseries subcommand for single path analysis
- Add integration test for timeseries subcommand

### Improvements

- Rename comparison structs to match conventions
- Modernize syntax in integration tests

## v0.7.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.7.1...v0.7.2)

### Features

- Add profiling capability to runtime

### Development

- Add fuzz tests to codebase
- Enhance integration test setup and teardown

## v0.7.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.7.0...v0.7.1)

### Features

- Show multiple owners for each file

### Improvements

- Add diagnostics info for table view

## v0.7.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.5...v0.7.0)

### Improvements

- Update ranking image with new UX
- Add suite of repos for testing
- Adjust integration tests to use JSON output
- Add preliminary forms of integration tests

### Fixes

- Fix edge cases in commit detection
- Fix Git age calculations
- Fix help text inconsistency

## v0.6.5

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.4...v0.6.5)

### Improvements

- Enhance compare command validation
- Split Git logic into multiple modules
- Funnel all errors into single module
- Enhance readability of help text
- Refine error-based messages across app

## v0.6.4

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.3...v0.6.4)

### Improvements

- Apply formatting to performance table
- Revise use case sections in README
- Update ranking image for better UX
- Add dots to more comments
- Refactor all flag logic for readability
- Consolidate global flags for readability

## v0.6.3

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.2...v0.6.3)

### Development

- Add command for lefthook pre-commit
- Refine pre-commit setup
- Rename pre-commit command

### Improvements

- Add context propagation and tweak folder table
- Fix sequence ID of comment
- Shorten analysis header for brevity

## v0.6.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.1...v0.6.2)

### Build

- Add ldflags to build

## v0.6.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.6.0...v0.6.1)

### Improvements

- Update comments of compare subcommands
- Remove redundant comment in logic

## v0.6.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.5.0...v0.6.0)

### Improvements

- Remove extraneous comment from main.go
- Add minimal error handling for Viper issues
- Add sample configs and reference them in README
- Add Viper for streamlined customization

### Features

- Provide concrete example to compare section
- Lower-case bus-factor term in README
- Shorten wording on hot mode in README

## v0.5.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.4.4...v0.5.0)

### Development

- Adjust lefthook semantics
- Use constants for modes
- Add differentiator clause in README
- Refine some tables
- Add coloring to table deltas

### Improvements

- Refactor all Git client touch points
- Simplify lefthook to one command
- Add lefthook to project

## v0.4.4

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.4.3...v0.4.4)

### Development

- Update Go dependencies

### Improvements

- Improve UX of compare table

## v0.4.3

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.4.2...v0.4.3)

### Features

- Add file deltas to compare JSON

## v0.4.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.3.0...v0.4.2)

### Improvements

- Refine content in risk section
- Revise example to be more realistic

### Fixes

- Fix implicit rank bug due to refactor
- Fix compare setup logic
- Fix edge case with file comparisons
- Fix hotspot delta section

## v0.3.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.2.2...v0.3.0)

### Improvements

- Add qualifier about performance measurements
- Fix wording for output section
- Adjust output and make example repeatable
- Change low color from blue to cyan
- Consolidate Makefile logic for cleanup

### Development

- Tweak from cyan to blue
- Adjust colors to work for all terminals
- Ensure Makefile applies fix and format

## v0.2.2

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.2.1...v0.2.2)

### Improvements

- Remove bin gitkeep assets
- Fix linting issues
- Refactor Makefile for readability
- Add version command for diagnostics

### Development

- Verify and fix aggregation for recent assets
- Replace require with assert usage in test
- Refactor tests into separate layers

## v0.2.1

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.2.0...v0.2.1)

### Improvements

- Add option C to README
- Enable time-ago for --start and --end

## v0.2.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.1.0...v0.2.0)

### Features

- Change changelog format
- Simplify tips language
- Add doc references to ago semantics
- Add support for time-ago semantics

### Development

- Add contribution guidelines
- Add issue templates for GitHub

## v0.1.0

[Full Changelog](https://github.com/huangsam/hotspot/compare/v0.0.0...v0.1.0)

### Build

- Add goreleaser build settings

### Improvements

- Rewrite punch line to be succinct
- Use text-less image in README
- Adjust first README tip
- Fill README image space better
- Adjust README image slightly
- Add sample output to README
- Shuffle README content around image
- Add image to hotspot README

### Development

- Add tests for computing folder score
- Add unit test for selectOutputFile
- Adjust label colors to match theme
- Add comments to all schema fields
- Enhance UX for folder JSON output
- Add owner info to folder results

## Earlier versions

[Full Changelog](https://github.com/huangsam/hotspot/compare/3e250a2...v0.1.0)

### Features

- Add CSV and JSON output modes
- Add support for analyzing folders
- Add ability to run benchmarks
- Add version command for diagnostics
- Add time-ago semantics
- Add Viper configuration
- Add compare subcommand
- Add folder analysis support
- Add CSV/JSON output
- Add Git client abstraction
- Add Cobra CLI framework
- Add comprehensive testing
- Add CI/CD pipeline
- Add linting and formatting
- Add Makefile build system

### Improvements

- Optimize performance and simplify logic
- Enhance error handling and logging
- Improve UX and output formatting
- Add comprehensive documentation
- Refactor code for maintainability
- Add proper package structure
