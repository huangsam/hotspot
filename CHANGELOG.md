# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.10.4] - 2025-11-26

### Added
- Database integration tests and test setup optimization

### Changed
- Schema organization: split `schema.go` into multiple focused files
- Reorganize schema package and remove external DB rules
- Reduce metrics indentation
- Update Go modules and add testcontainers-go

## [1.10.3] - 2025-11-26

### Added
- Insights to success/failure messages for check output

### Changed
- Optimize memory usage for score calculations
- Improve check failure output
- Reorder schema fields, add line breaks, and refine comments for better structure
- Refactor schema terms across multiple files
- Reorganize all core tests for readability
- Migrate filter logic to check builder and split print functions
- Update benchmark results, performance details, requirements language, and agentic docs

### Fixed
- Check section references in USERGUIDE and README formatting

## [1.10.2] - 2025-11-23

### Added
- Tests for builder patterns and check enhancements

### Changed
- Move algorithms to core/algo and refactor check logic
- Rename core/builder to core/builder_file
- Enhance success/failure output with max scores and emojis
- Update AGENTS.md with structural changes and dev flow

## [1.10.1] - 2025-11-22

### Changed
- Fix schema terms in configuration and migration code
- Change PostgreSQL driver from pq to pgx
- Update dependencies for pgx driver
- Update README and USERGUIDE documentation

## [1.10.0] - 2025-11-22

### Added
- Schema versioning and migration system for analysis tracking (#24)
- Database migration framework for analysis schema updates

### Changed
- Migrate command logic from main.go to iocache package for better organization
- Apply periods to the end of all command descriptions

## [1.9.0] - 2025-11-22

### Added
- `hotspot check` command for CI with configurable thresholds (#22)

### Changed
- Optimize check runtime and bindings
- Split phony targets into multiple lines in Makefile
- Optimize integration test runtime and config overrides
- Update benchmark results in README.md
- Increase test coverage in core and iocache

## [1.8.0] - 2025-11-21

### Added
- Parquet exports from analysis DB (#20)

### Changed
- Update README.md and USERGUIDE.md with Parquet export details

## [1.7.0] - 2025-11-20

### Added
- Unified `hotspot_file_scores_metrics` table merging prior tables for better data organization

### Changed
- Remove deprecated AnalysisStore methods for cleaner interface and reduced complexity

### Deprecated
- Retract v1.6.0 due to breaking API changes that would cause incompatibility

## [1.6.0] - 2025-11-20

### Added
- Analysis tracking system with DB storage for SQLite, MySQL, PostgreSQL backends
- Status commands for analysis and cache backends
- Comprehensive tests for analysis tracking and backend status

### Changed
- Migrate user instructions to USERGUIDE.md for better UX
- Bump golang.org/x/crypto from 0.37.0 to 0.45.0
- Refactor iocache package and integrate tracking into core analysis pipeline

## [1.5.1] - 2025-11-16

### Added
- Coverage command to Makefile
- Comprehensive cache tests
- Folder aggregation tests

### Changed
- Update make help formatting for better readability
- Add emoji guard for file output success
- Split output tests into multiple files
- Refactor outwriter for testability

## [1.5.0] - 2025-11-16

### Added
- MySQL and PostgreSQL support for result caching
- `--color` and `--emoji` flags for output formatting

### Changed
- Make analysis footer more compact
- Simplify benchmark report generation
- Fix timing calculations in benchmark logic for accuracy
- Update README with new findings and make use cases section more focused
- Simplify metrics command output for better readability
- Restructure core and internal packages for better organization
- Significantly improve test coverage for iocache and outwriter
- Refine code style across core, internal, and main.go
- Refactor output handling in outwriter package
- Rename iocache to cache for better naming consistency
- Reduce duplication in outwriter package for better maintainability

## [1.4.0] - 2025-11-14

### Added
- Persistent SQLite result caching with ~35x faster analysis for repeated runs and seamless mode switching

### Changed
- Bump cache version for compatibility

## [1.3.3] - 2025-11-11

### Added
- Documentation for config weights and timeseries section
- More schema tests

### Fixed
- Accidental header cleanup in changelog

## [1.3.2] - 2025-11-11

### Changed
- Improve test coverage across core analysis, config validation, and Git client
- Add edge case testing for config validation
- Replace brittle time equality with `WithinDuration` checks
- Enhance test reliability with paths and time tolerances
- Add demo GIF and minor README improvements

## [1.3.1] - 2025-11-11

### Changed
- Make timeseries logic more accurate with comprehensive analysis
- Improve lookback and timeseries user experience with better defaults
- Fix timeseries header semantics for clearer output
- Migrate timeseries constants and refactor analysis logic
- Add comments to time approximations and update README benchmarks

### Fixed
- Integration test issues
- Update benchmarks

## [1.3.0] - 2025-11-10

### Added
- Benchmark script and performance details for analysis

### Changed
- Update README with motivation section and improved formatting
- Update and link benchmark tables with latest performance data

### Fixed
- Handle edge cases in name abbreviation logic

## [1.2.0] - 2025-11-09

### Added
- CSV and JSON output support to `metrics` command for programmatic access
- Integration tests for folders and compare commands

### Changed
- Optimize integration test performance

## [1.1.5] - 2025-11-09

### Changed
- Add exact timestamps to timeseries output for better precision
- Detect terminal width for improved tabular view formatting

### Fixed
- Nil owner arrays in JSON output to prevent serialization errors

## [1.1.4] - 2025-11-09

### Changed
- Use pipe delimiters for owner names in all CSV outputs to prevent parsing issues
- Separate before/after owner columns for clearer comparison data

## [1.1.3] - 2025-11-09

### Changed
- Extract duplicated weight processing logic into shared `ProcessWeightsRawInput` helper, eliminating ~50 lines of duplicated code
- Combine `loadActiveWeights` and `ExecuteHotspotMetrics` functions to reduce unnecessary abstraction

## [1.1.2] - 2025-11-09

### Changed
- Remove O(N) git calls during file analysis by using pre-aggregated commit data instead of individual `git log` calls per file, significantly improving performance for large repositories
- Optimize integration tests to build hotspot binary only once instead of 5-6 times, reducing test execution time by 37% (from ~7.4s to ~4.6s)
- Remove unused `GetFileFirstCommitTime` method and related dead code
- Add `hotspot-integration-*/` pattern to `.gitignore` to prevent accidental commits of temporary test directories
- Unify all integration tests to use single `parseHotspotDetailOutput` function with `HotspotFileDetail` struct

### Fixed
- Age verification tests by using relative time windows from aggregated data instead of broken individual git queries

## [1.1.1] - 2025-11-08

### Changed
- Add context.Context throughout the codebase for better cancellability and request-scoped data flow
- Make command headers more compact and consistent across all subcommands

## [1.1.0] - 2025-11-08

### Added
- Owner information to comparison and timeseries outputs

### Changed
- Add Mode field to all result structs for uniform API
- Add rank and label fields to folder JSON output
- Remove unused config parameters from writer functions

## [1.0.1] - 2025-11-08

### Changed
- Use * for multiply sign in output

### Fixed
- Gitignore: add .back and .backup to gitignore

## [1.0.0] - 2025-11-08

### Added
- Custom scoring weights via YAML configuration
- `metrics` subcommand for viewing scoring formulas
- `timeseries` subcommand for historical score tracking
- `compare` subcommand for delta analysis between Git refs
- CPU/memory profiling capabilities

### Changed
- Enable live reporting of score weights
- Shorten metrics command output
- Add references to weights in documentation
- Update README with comprehensive examples and performance data

### Fixed
- Custom weight config logic
- Lint issues with integration tests
- Add tests for custom weight functionality

## [0.9.1] - 2025-11-08

### Changed
- Refactor core aggregate analysis flow
- Organize features into overarching section
- Make small tweaks to README examples

## [0.9.0] - 2025-11-08

### Changed
- Shorten the config file section
- Update scoring section in README
- Add timeseries section to README
- Consolidate gitignore sections
- Apply lint and format to integration tests
- Update Go dependencies
- Remove fuzz testdata
- Add tests for core aggregation and analysis
- Add AGENTS.md for improved documentation

## [0.8.0] - 2025-11-08

### Added
- Timeseries subcommand for single path analysis
- Integration test for timeseries subcommand

### Changed
- Rename comparison structs to match conventions
- Modernize syntax in integration tests

## [0.7.2] - 2025-11-08

### Added
- Profiling capability to runtime
- Fuzz tests to codebase

### Changed
- Enhance integration test setup and teardown

## [0.7.1] - 2025-11-08

### Added
- Show multiple owners for each file

### Changed
- Add diagnostics info for table view

## [0.7.0] - 2025-11-08

### Changed
- Update ranking image with new UX
- Add suite of repos for testing
- Adjust integration tests to use JSON output
- Add preliminary forms of integration tests

### Fixed
- Edge cases in commit detection
- Git age calculations
- Help text inconsistency

## [0.6.5] - 2025-11-07

### Changed
- Enhance compare command validation
- Split Git logic into multiple modules
- Funnel all errors into single module
- Enhance readability of help text
- Refine error-based messages across app

## [0.6.4] - 2025-11-07

### Changed
- Apply formatting to performance table
- Revise use case sections in README
- Update ranking image for better UX
- Add dots to more comments
- Refactor all flag logic for readability
- Consolidate global flags for readability

## [0.6.3] - 2025-11-07

### Changed
- Add command for lefthook pre-commit
- Refine pre-commit setup
- Rename pre-commit command
- Add context propagation and tweak folder table
- Fix sequence ID of comment
- Shorten analysis header for brevity

## [0.6.2] - 2025-11-06

### Changed
- Add ldflags to build

## [0.6.1] - 2025-11-06

### Changed
- Update comments of compare subcommands
- Remove redundant comment in logic

## [0.6.0] - 2025-11-06

### Added
- Viper for streamlined customization
- Sample configs and reference them in README
- Concrete example to compare section

### Changed
- Remove extraneous comment from main.go
- Add minimal error handling for Viper issues
- Lower-case bus-factor term in README
- Shorten wording on hot mode in README

## [0.5.0] - 2025-11-06

### Changed
- Adjust lefthook semantics
- Use constants for modes
- Add differentiator clause in README
- Refine some tables
- Add coloring to table deltas
- Refactor all Git client touch points
- Simplify lefthook to one command
- Add lefthook to project

## [0.4.4] - 2025-11-05

### Changed
- Update Go dependencies
- Improve UX of compare table

## [0.4.3] - 2025-11-05

### Added
- File deltas to compare JSON

## [0.4.2] - 2025-11-05

### Changed
- Refine content in risk section
- Revise example to be more realistic

### Fixed
- Implicit rank bug due to refactor
- Compare setup logic
- Edge case with file comparisons
- Hotspot delta section

## [0.3.0] - 2025-11-04

### Changed
- Add qualifier about performance measurements
- Fix wording for output section
- Adjust output and make example repeatable
- Change low color from blue to cyan
- Consolidate Makefile logic for cleanup
- Tweak from cyan to blue
- Adjust colors to work for all terminals
- Ensure Makefile applies fix and format

## [0.2.2] - 2025-11-04

### Changed
- Remove bin gitkeep assets
- Fix linting issues
- Refactor Makefile for readability
- Add version command for diagnostics
- Verify and fix aggregation for recent assets
- Replace require with assert usage in test
- Refactor tests into separate layers

## [0.2.1] - 2025-11-03

### Changed
- Add option C to README
- Enable time-ago for --start and --end

## [0.2.0] - 2025-11-03

### Added
- Time-ago semantics support
- Contribution guidelines
- Issue templates for GitHub

### Changed
- Change changelog format
- Simplify tips language
- Add doc references to ago semantics

## [0.1.0] - 2025-11-03

### Added
- Goreleaser build settings
- Tests for computing folder score
- Unit test for selectOutputFile

### Changed
- Rewrite punch line to be succinct
- Use text-less image in README
- Adjust first README tip
- Fill README image space better
- Adjust README image slightly
- Add sample output to README
- Shuffle README content around image
- Add image to hotspot README
- Adjust label colors to match theme
- Add comments to all schema fields
- Enhance UX for folder JSON output
- Add owner info to folder results

## Earlier versions

Initial development covered core functionality including:

### Added
- CSV and JSON output modes
- Folder analysis support
- Benchmarking capabilities
- Version command for diagnostics
- Time-ago semantics
- Viper configuration
- Compare subcommand
- Output formats (CSV/JSON)
- Git client abstraction
- Cobra CLI framework
- Comprehensive testing
- CI/CD pipeline
- Linting and formatting
- Makefile build system

### Changed
- Performance optimizations and logic simplification
- Error handling and logging enhancements
- UX and output formatting improvements
- Documentation comprehensive additions
- Code quality and maintainability refactoring
- Proper package structure architecture

[1.10.4]: https://github.com/huangsam/hotspot/compare/v1.10.3...v1.10.4
[1.10.3]: https://github.com/huangsam/hotspot/compare/v1.10.2...v1.10.3
[1.10.2]: https://github.com/huangsam/hotspot/compare/v1.10.1...v1.10.2
[1.10.1]: https://github.com/huangsam/hotspot/compare/v1.10.0...v1.10.1
[1.10.0]: https://github.com/huangsam/hotspot/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/huangsam/hotspot/compare/v1.8.0...v1.9.0
[1.8.0]: https://github.com/huangsam/hotspot/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/huangsam/hotspot/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/huangsam/hotspot/compare/v1.5.1...v1.6.0
[1.5.1]: https://github.com/huangsam/hotspot/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/huangsam/hotspot/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/huangsam/hotspot/compare/v1.3.3...v1.4.0
[1.3.3]: https://github.com/huangsam/hotspot/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/huangsam/hotspot/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/huangsam/hotspot/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/huangsam/hotspot/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/huangsam/hotspot/compare/v1.1.5...v1.2.0
[1.1.5]: https://github.com/huangsam/hotspot/compare/v1.1.4...v1.1.5
[1.1.4]: https://github.com/huangsam/hotspot/compare/v1.1.3...v1.1.4
[1.1.3]: https://github.com/huangsam/hotspot/compare/v1.1.2...v1.1.3
[1.1.2]: https://github.com/huangsam/hotspot/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/huangsam/hotspot/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/huangsam/hotspot/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/huangsam/hotspot/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/huangsam/hotspot/compare/v0.9.1...v1.0.0
[0.9.1]: https://github.com/huangsam/hotspot/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/huangsam/hotspot/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/huangsam/hotspot/compare/v0.7.2...v0.8.0
[0.7.2]: https://github.com/huangsam/hotspot/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/huangsam/hotspot/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/huangsam/hotspot/compare/v0.6.5...v0.7.0
[0.6.5]: https://github.com/huangsam/hotspot/compare/v0.6.4...v0.6.5
[0.6.4]: https://github.com/huangsam/hotspot/compare/v0.6.3...v0.6.4
[0.6.3]: https://github.com/huangsam/hotspot/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/huangsam/hotspot/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/huangsam/hotspot/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/huangsam/hotspot/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/huangsam/hotspot/compare/v0.4.4...v0.5.0
[0.4.4]: https://github.com/huangsam/hotspot/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/huangsam/hotspot/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/huangsam/hotspot/compare/v0.3.0...v0.4.2
[0.3.0]: https://github.com/huangsam/hotspot/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/huangsam/hotspot/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/huangsam/hotspot/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/huangsam/hotspot/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/huangsam/hotspot/compare/v0.0.0...v0.1.0
