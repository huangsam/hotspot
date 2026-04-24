# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org).

## [1.20.0] - 2026-04-24

### Added
- High-fidelity interactive SVG heatmap visualization for hotspots.
- Squarified treemap algorithm with directory-based spatial clustering.
- Dynamic font scaling and smart label hiding for dense visualizations.
- Integrated interactive SVG hotspots directly into README documentation.
- Native MCP tool `get_heatmap` for agent-driven visual repository audits.

### Changed
- Refactored heatmap provider into a modular, component-based architecture.
- Optimized SVG coordinate math and visual hierarchy for clarity.
- Standardized SVG output to use a clean, "Intelligence Layer" aesthetic.

### Fixed
- Resolved cyclomatic complexity and errcheck lints in output providers.
- Updated MCP server tests to validate registration of 10 tools.

## [1.19.0] - 2026-04-21

### Added
- Centralized data-driven configuration system in `schema/data/`.
- Folder Risk Intelligence: added `Gini` and `UniqueContributors` metrics.
- Extended IaC Detection: support for Kustomize, Skaffold, Tilt, and CI/CD.

### Changed
- Consolidated all repository presets into `presets.yaml`.
- Refactored `init` command to use internal data-driven presets.
- Hardened Path Filtering: ensured strict directory boundaries in monorepos.
- Preset Refinement: expanded exclusions for modern toolchain noise.
- Updated `USERGUIDE.md` and `README.md` for new configuration flow.

### Removed
- Deleted redundant `examples/cli/` directory and `clipresets` package.

### Fixed
- Monorepo Accuracy: fixed edge cases in partial directory matching.

## [1.18.1] - 2026-04-20

### Added
- IaC detection for Chef, Puppet, CloudFormation, Bicep, SAM, and Serverless.
- Zero-allocation classification engine (0 B/op) for monorepo performance.
- Memory reporting in `make bench` for continuous performance monitoring.
- High-throughput benchmarks for IaC detection and path matching.

### Changed
- Refactored `isIaCFile` into a declarative structure for maintainability.
- Optimized `PathMatcher` to eliminate redundant allocations in hot loops.
- Expanded `infra` preset exclusions to filter out modern tool noise.

### Fixed
- Monorepo metrics: isolated Git activity to filtered subdirectory paths.
- Cache Collision: unique cache keys for subdirectory analysis.
- Accuracy: context-aware IaC detection for Ruby (`.rb`) and TOML files.

## [1.18.0] - 2026-04-18

### Added
- Comprehensive benchmark suite for core, schema, and iocache packages.
- Added `busy_timeout` for SQLite backend to improve concurrency.

### Changed
- Reduced aggregation allocations by 99% via struct-based aggregation
- Implemented high-performance, zero-allocation Git log parser.
- Optimized Gini coefficient scoring calculations and file stat implementation
- Streamlined exclusion logic by optimizing recursive glob calls.
- Refined complexity scoring with intelligent file detection.
- Updated Go requirement to 1.26.0 and refreshed all dependencies

### Fixed
- Line counting logic in `FetchFileStats` to handle trailing newlines correctly
- PostgreSQL compatibility issue when pruning analysis entries
- Transactional safety in global stores and refined config detection test cases

## [1.17.0] - 2026-04-18

### Added
- Persisted ROI/Recency metrics across SQLite, MySQL, and PostgreSQL.
- Added Analytical "Intelligence Layer" for high-fidelity reasoning.
- Full parity for file/folder comparison MCP tools.
- `run_check` MCP tool for automated policy gating.
- Full metrics and reasoning signal parity for Parquet lake exports.

### Changed
- Optimized analysis tracking with batched, transactional recording.
- Unified exclusion logic across CLI presets and MCP server invocations
- Refined `small` preset for tighter microservice analysis defaults
- Refreshed demo assets and updated performance benchmarks in documentation

### Fixed
- Transactional safety in integration tests with dialect assertions.
- Preset handling for MCP to ensure consistent recursive glob filtering
- Dependency hygiene with Go module updates and `go mod tidy`

## [1.16.2] - 2026-04-17

### Added
- `hotspot analysis history` command for browsing historical analysis runs
- Full parity for `--explain` flag in `CSV` and `Markdown` export formats
- Multi-format support (JSON, CSV, Markdown) for analysis tracking history

### Changed
- Refactored output orchestration layer for consistent cross-format reporting
- Improved CSV stability by standardizing columns regardless of filtering flags
- Sanitized history views by moving raw JSON parameters to metadata-only fields
- Reorganized documentation into `docs/` for better maintainability.
- Refactored macro-benchmark to use `bench-repos` target

## [1.16.1] - 2026-04-16

### Added
- Standardized `hotspot.small.yml` preset for microservices and libraries
- Shape-aware recency intelligence with dynamic thresholds
- Recursive wildcard support (`**/`) for file and directory exclusions
- Extensive unit testing and fuzzing for path ignore logic
- Bazel, Buck, and Pants build artifact exclusions for monorepo hardening

### Changed
- Refined presets with modern artifact patterns (Terraform, Next.js, Vercel).
- Streamlined `small` preset to leverage built-in system defaults

### Fixed
- Fixed recursive glob matching for multi-level subdirectory exclusions.

## [1.16.0] - 2026-04-16

### Added
- `hotspot init` command for automated repository setup and presets.
- `get_release_journey` and `get_blast_radius` architectural tools.
- Native MCP documentation resources served directly from binary.
- Guided analysis playbooks for repository audits and prioritization.
- Enhanced MCP tool intelligence with annotations and synced defaults.
- Standardized parameter descriptions and mappings across all tools.

### Changed
- Disabled analysis tracking by default to reduce overhead for local use
- Removed "stale" mode from the entire codebase
- Hardened scoring engine with edge-case tests for Git boundaries
- Added strict MCP tool registration verification to ensure API schema stability

### Fixed
- SQLite/PostgreSQL scanning warnings for nullable integer fields
- Default database backend selection logic for consistent persistence
- Descriptive error logging and reporting for early command failures
- Synchronization of CLI help strings and documentation across all guides

## [1.15.0] - 2026-04-15

### Added
- `hotspot shape` for lightweight characterization and presets.
- `get_repo_shape` MCP tool to expose shape analysis as JSON for AI agents
- Preset system (small, large, infra) with embedded configuration templates
- Preset support for all MCP analysis tools.

### Changed
- MCP presets now treat invalid names as optional convenience.

### Fixed
- Help text alignment for `--mode` flag across CLI and documentation

## [1.14.0] - 2026-04-14

### Added
- ROI scoring mode to prioritize high-return refactoring.
- Time-weighted activity (decay) to prioritize recent development.
- Repository URN tracking for portable repository identity across machines
- Markdown and Describe (Executive Summary) output formats
- Structured reasoning signals for AI and human interpretability.
- Database pagination and URN filtering for historical analysis queries
- Hardened Agentic documentation and expanded example configuration suite

### Changed
- Modernized output architecture with a modular, extensible provider pattern
- Standardized metadata for consistent API responses across modes.
- Performance optimizations resulting in 15-20% faster cold analysis times
- Hardened database persistence layer with pluggable SQL dialects
- Updated all direct and indirect dependencies

### Fixed
- Improved debuff logic for autogenerated and test code.
- Synced CLI help strings and docs for all output formats.

## [1.13.0] - 2026-04-11

### Added
- Support for human-readable relative time expressions (e.g., "30d", "6mo")
- Enhanced MCP tools with dynamic `repo_path` and full parameter parity
- Unified analysis pipeline for robust orchestration across all scoring modes

### Changed
- Finalized decoupling of configuration via interface-bound settings
- Consolidated parsing logic for relative time and historical lookbacks

### Fixed
- Restored analysis tracking in timeseries results to fix unit test failures

## [1.12.1] - 2026-02-22

### Changed
- Migrate from `go-sqlite3` (CGO) to pure-Go `modernc.org/sqlite` driver to
  support zero-dependency cross-compilation for release binaries

## [1.12.0] - 2026-02-22

### Added
- Model Context Protocol (MCP) server for native AI Agent integration
- `hotspot mcp` subcommand to expose core analysis tools via JSON-RPC.
- Agentic documentation (AGENTS.md) and playbook examples for MCP setup

### Changed
- Modularized core dependencies into `toolHandler` for MCP integration.
- Decoupled stdout rendering from getters for seamless JSON-RPC transport.
- Update Go setup to 1.26 in GitHub Actions CI

### Fixed
- Eliminate plain-text stdout pollution during backend analysis
- Fix Goreleaser linker flag injections resulting from CLI package refactor

## [1.11.0] - 2026-02-21

### Added
- Tests to check for race conditions

### Changed
- Split monolithic `main.go` into `cmd/` package
- Improve command documentation and verbiage across commands
- Remove emojis from CLI outputs for cleaner presentation
- Enhance error messages across core and internal packages
- Update Go dependencies and bump multiple modules
- Update GitHub Actions checkout action to v6

### Fixed
- Fix comments and whitespace in initialization logic

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
- Refined schema structure: reordered fields and improved comments.
- Refactor schema terms across multiple files
- Reorganize all core tests for readability
- Migrate filter logic to check builder and split print functions
- Updated benchmarks, performance details, and agentic documentation.

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
- Unified metrics into `hotspot_file_scores_metrics` table.

### Changed
- Cleaned up `AnalysisStore` by removing deprecated methods.

### Deprecated
- Retract v1.6.0 due to breaking API changes that would cause incompatibility

## [1.6.0] - 2025-11-20

### Added
- Analysis tracking with support for SQLite, MySQL, and PostgreSQL.
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
- Persistent SQLite caching for ~35x faster repeated analyses.

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
- Use pipe delimiters for owners in CSV to prevent parsing issues.
- Separate before/after owner columns for clearer comparison data

## [1.1.3] - 2025-11-09

### Changed
- Consolidated weight processing logic into `ProcessWeightsRawInput`.
- Combined weight loading and metrics execution for simpler abstraction.

## [1.1.2] - 2025-11-09

### Changed
- Optimized file analysis using pre-aggregated commit data (O(1)).
- Optimized integration tests: single binary build reduces time by 37%.
- Remove unused `GetFileFirstCommitTime` method and related dead code
- Ignored `hotspot-integration-*/` to prevent committing test dirs.
- Unified integration tests using `parseHotspotDetailOutput` helper.

### Fixed
- Fixed age tests using relative time windows from aggregated data.

## [1.1.1] - 2025-11-08

### Changed
- Added `context.Context` for better cancellability across the codebase.
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

[1.20.0]: https://github.com/huangsam/hotspot/compare/v1.19.0...v1.20.0
[1.19.0]: https://github.com/huangsam/hotspot/compare/v1.18.1...v1.19.0
[1.18.1]: https://github.com/huangsam/hotspot/compare/v1.18.0...v1.18.1
[1.18.0]: https://github.com/huangsam/hotspot/compare/v1.17.0...v1.18.0
[1.17.0]: https://github.com/huangsam/hotspot/compare/v1.16.2...v1.17.0
[1.16.2]: https://github.com/huangsam/hotspot/compare/v1.16.1...v1.16.2
[1.16.1]: https://github.com/huangsam/hotspot/compare/v1.16.0...v1.16.1
[1.16.0]: https://github.com/huangsam/hotspot/compare/v1.15.0...v1.16.0
[1.15.0]: https://github.com/huangsam/hotspot/compare/v1.14.0...v1.15.0
[1.14.0]: https://github.com/huangsam/hotspot/compare/v1.13.0...v1.14.0
[1.13.0]: https://github.com/huangsam/hotspot/compare/v1.12.1...v1.13.0
[1.12.1]: https://github.com/huangsam/hotspot/compare/v1.12.0...v1.12.1
[1.12.0]: https://github.com/huangsam/hotspot/compare/v1.11.0...v1.12.0
[1.11.0]: https://github.com/huangsam/hotspot/compare/v1.10.4...v1.11.0
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
