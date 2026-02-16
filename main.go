// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// All linker flags will be set by goreleaser infra at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// rootCtx is the root context for all operations.
var rootCtx = context.Background()

// cfg will hold the validated, final configuration.
var cfg = &contract.Config{}

// input holds the raw, unvalidated configuration from all sources (file, env, flags).
// Viper will unmarshal into this struct.
var input = &contract.ConfigRawInput{}

// profile holds profiling configuration.
var profile = &contract.ProfileConfig{}

// cacheManager is the global persistence manager instance.
var cacheManager contract.CacheManager

// startProfiling starts CPU and memory profiling if enabled.
func startProfiling() error {
	if !profile.Enabled {
		return nil
	}

	// Start CPU profiling
	cpuFile, err := os.Create(profile.Prefix + ".cpu.prof")
	if err != nil {
		return fmt.Errorf("could not create CPU profile: %w", err)
	}
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return fmt.Errorf("could not start CPU profiling: %w", err)
	}

	// Memory profiling will be captured at the end
	_, err = fmt.Fprintf(os.Stdout, "Profiling enabled. CPU profile: %s.cpu.prof, Memory profile: %s.mem.prof\n", profile.Prefix, profile.Prefix)
	return err
}

// stopProfiling stops profiling and writes memory profile.
func stopProfiling() error {
	if !profile.Enabled {
		return nil
	}

	pprof.StopCPUProfile()

	// Write memory profile
	memFile, err := os.Create(profile.Prefix + ".mem.prof")
	if err != nil {
		return fmt.Errorf("could not create memory profile: %w", err)
	}
	defer func() { _ = memFile.Close() }()

	if err := pprof.WriteHeapProfile(memFile); err != nil {
		return fmt.Errorf("could not write memory profile: %w", err)
	}

	_, err = fmt.Fprintf(os.Stdout, "Profiling complete. Use 'go tool pprof %s.cpu.prof' to analyze.\n", profile.Prefix)
	return err
}

// rootCmd is the command-line entrypoint for all other commands.
var rootCmd = &cobra.Command{
	Use:                "hotspot",
	Short:              "Analyze Git repository activity to find code hotspots.",
	Long:               `Hotspot cuts through Git history to show you which files and folders are your greatest risk.`,
	Version:            version,
	SilenceErrors:      true,
	SilenceUsage:       true,
	DisableSuggestions: true,
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Check if a specific config file is provided
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Set config file name and paths
		viper.SetConfigName(".hotspot") // Name of config file (without extension)
		viper.SetConfigType("yaml")     // We'll use YAML format
		viper.AddConfigPath(".")        // Look in the current directory
		viper.AddConfigPath("$HOME")    // Look in the home directory
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("HOTSPOT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Read in environment variables that match

	// Set defaults in Viper
	viper.SetDefault("limit", contract.DefaultResultLimit)
	viper.SetDefault("workers", contract.DefaultWorkers)
	viper.SetDefault("mode", schema.HotMode)
	viper.SetDefault("precision", contract.DefaultPrecision)
	viper.SetDefault("output", schema.TextOut)
	viper.SetDefault("lookback", "6 months")
	viper.SetDefault("cache-backend", schema.SQLiteBackend)
	viper.SetDefault("cache-db-connect", "")
	viper.SetDefault("analysis-backend", "")
	viper.SetDefault("analysis-db-connect", "")
	viper.SetDefault("color", "yes")
}

// sharedSetup unmarshals config and runs validation.
func sharedSetup(ctx context.Context, _ *cobra.Command, args []string) error {
	// Handle profiling flag
	profilePrefix := viper.GetString("profile")
	if err := contract.ProcessProfilingConfig(profile, profilePrefix); err != nil {
		return fmt.Errorf("failed to process profiling config: %w", err)
	}
	if profile.Enabled {
		if err := startProfiling(); err != nil {
			return fmt.Errorf("failed to start profiling: %w", err)
		}
	}

	// 1. Read config file. This merges defaults, file, env, and flags.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, which is fine; we'll use defaults/env/flags.
	}

	// 2. Unmarshal all resolved values from Viper into our raw input struct.
	if err := viper.Unmarshal(input); err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
	}

	// 3. Handle positional arguments (which Viper doesn't do).
	if len(args) == 1 {
		input.RepoPathStr = args[0]
	} else {
		input.RepoPathStr = "."
	}

	// 4. Run all validation and complex parsing.
	// This function now populates the global 'cfg' from 'input'.
	client := contract.NewLocalGitClient()
	if err := contract.ProcessAndValidate(ctx, cfg, client, input); err != nil {
		return err
	}

	// 5. Initialize persistence layer with validated config
	if err := iocache.InitStores(cfg.CacheBackend, cfg.CacheDBConnect, cfg.AnalysisBackend, cfg.AnalysisDBConnect); err != nil {
		return fmt.Errorf("failed to initialize persistence: %w", err)
	}

	return nil
}

// sharedSetupWrapper wraps sharedSetup to provide context for Cobra's PreRunE.
func sharedSetupWrapper(cmd *cobra.Command, args []string) error {
	return sharedSetup(rootCtx, cmd, args)
}

// loadConfigFile handles config file loading logic common to all setup functions.
func loadConfigFile() error {
	// Handle config file
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".hotspot")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}

	// Load config file if present
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults/env/flags
	}
	return nil
}

// cacheSetup loads minimal configuration needed for cache operations.
// This is used by commands that need cache access without full shared setup.
func cacheSetup() error {
	if err := loadConfigFile(); err != nil {
		return err
	}

	// Get cache-related config values
	backend := schema.DatabaseBackend(viper.GetString("cache-backend"))
	connStr := viper.GetString("cache-db-connect")

	// Basic validation for database backends
	if err := contract.ValidateDatabaseConnectionString(backend, connStr); err != nil {
		return err
	}

	// Initialize caching with the loaded config (no analysis tracking for cache commands)
	if err := iocache.InitStores(backend, connStr, "", ""); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	cfg.CacheBackend = backend
	cfg.CacheDBConnect = connStr

	return nil
}

// cacheSetupWrapper wraps cacheSetup to provide PreRunE for cache commands.
func cacheSetupWrapper(_ *cobra.Command, _ []string) error {
	return cacheSetup()
}

// analysisSetup loads minimal configuration needed for analysis operations.
// This is used by commands that need analysis access without full shared setup.
func analysisSetup() error {
	if err := loadConfigFile(); err != nil {
		return err
	}

	// Get analysis-related config values
	backendStr := viper.GetString("analysis-backend")
	connStr := viper.GetString("analysis-db-connect")

	// Handle empty backend as NoneBackend
	var backend schema.DatabaseBackend
	if backendStr == "" {
		backend = schema.NoneBackend
	} else {
		backend = schema.DatabaseBackend(backendStr)
	}

	// Basic validation for database backends
	if err := contract.ValidateDatabaseConnectionString(backend, connStr); err != nil {
		return err
	}

	// Get output-related config values (used by export command)
	outputFile := viper.GetString("output-file")

	// Initialize stores with the loaded config (no cache tracking for analysis commands)
	if err := iocache.InitStores(schema.NoneBackend, "", backend, connStr); err != nil {
		return fmt.Errorf("failed to initialize analysis: %w", err)
	}

	cfg.AnalysisBackend = backend
	cfg.AnalysisDBConnect = connStr
	cfg.OutputFile = outputFile

	return nil
}

// analysisSetupWrapper wraps analysisSetup to provide PreRunE for analysis commands.
func analysisSetupWrapper(_ *cobra.Command, _ []string) error {
	return analysisSetup()
}

// analysisMigrateSetup loads minimal configuration needed for migrate operations.
// This is a specialized setup that does NOT initialize stores or create tables,
// allowing migrations to run on a fresh database.
func analysisMigrateSetup() error {
	if err := loadConfigFile(); err != nil {
		return err
	}

	// Get analysis-related config values
	backendStr := viper.GetString("analysis-backend")
	connStr := viper.GetString("analysis-db-connect")

	// Handle empty backend as NoneBackend
	var backend schema.DatabaseBackend
	if backendStr == "" {
		backend = schema.NoneBackend
	} else {
		backend = schema.DatabaseBackend(backendStr)
	}

	// Basic validation for database backends
	if err := contract.ValidateDatabaseConnectionString(backend, connStr); err != nil {
		return err
	}

	// For SQLite backend with empty connection string, use default path
	if backend == schema.SQLiteBackend && connStr == "" {
		connStr = contract.GetAnalysisDBFilePath()
	}

	cfg.AnalysisBackend = backend
	cfg.AnalysisDBConnect = connStr

	return nil
}

// analysisMigrateSetupWrapper wraps analysisMigrateSetup to provide PreRunE for migrate command.
func analysisMigrateSetupWrapper(_ *cobra.Command, _ []string) error {
	return analysisMigrateSetup()
}

// filesCmd focuses on tactical, file-level analysis.
var filesCmd = &cobra.Command{
	Use:   "files [repo-path]",
	Short: "Show the top files ranked by risk score.",
	Long: `Perform deep Git analysis and rank individual files by risk score.

Analyzes the entire history of each file to compute risk metrics, helping you:
- Identify which files are most critical to your codebase
- Find files that are changing too frequently (churn hotspots)
- Spot files with uneven ownership and knowledge silos
- Locate large, complex files that are difficult to maintain
- Discover important files that haven't been maintained recently

Scores files based on your selected mode (hot, risk, complexity, stale),
ranking them from highest to lowest risk.

Examples:
  # Find the most active/volatile files
  hotspot files --mode hot --limit 20

  # Identify files with knowledge concentration risk
  hotspot files --mode risk

  # Find complex files by age and size
  hotspot files --mode complexity

  # Show stale but important files
  hotspot files --mode stale

  # Include detailed metrics and component breakdown
  hotspot files --detail --explain --owner

  # Export findings to CSV for tracking
  hotspot files --mode hot --output csv --output-file hotspots.csv`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotFiles(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Cannot run files analysis", err)
		}
	},
}

// foldersCmd focuses on tactical, folder-level analysis.
var foldersCmd = &cobra.Command{
	Use:   "folders [repo-path]",
	Short: "Show the top folders ranked by risk score.",
	Long: `Perform deep Git analysis and rank directories/folders by risk score.

Aggregates file-level analysis to folder level. Helps you:
- Identify which subsystems are risky or volatile
- Assess team/module boundaries
- Find areas that need architectural attention
- Plan refactoring efforts strategically
- Allocate maintenance resources effectively

Each folder's score is weighted by file size and activity.

Examples:
  # Find the riskiest subsystems
  hotspot folders --mode hot

  # See which modules have knowledge concentration issues
  hotspot folders --mode risk

  # Identify complex subsystems worth refactoring
  hotspot folders --mode complexity

  # Find neglected important modules
  hotspot folders --mode stale

  # Include metrics and owner information
  hotspot folders --detail --owner`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotFolders(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Cannot run folders analysis", err)
		}
	},
}

// compareCmd focused on strategic per-target comparisons.
var compareCmd = &cobra.Command{
	Use:   "compare [repo-path]",
	Short: "Compare analysis results between two Git references.",
	Long: `Compare analysis results between two Git references to track how risk has evolved.

Ideal for:
- Release comparisons - see what changed between versions
- Refactoring validation - verify changes reduced risk
- Feature branch reviews - ensure PRs don't introduce high-risk files
- Progress tracking - monitor improvements over time
- Regression detection - catch files becoming riskier

Available comparison types:
  compare files   - Track individual file risk changes
  compare folders - Monitor subsystem health changes

Each comparison shows before/after scores, deltas, and ranking changes.`,
}

// checkCmd focused on CI/CD policy enforcement.
var checkCmd = &cobra.Command{
	Use:   "check [repo-path]",
	Short: "Enforce risk thresholds for CI/CD pipelines (fails build on violations)",
	Long: `Analyze ONLY changed files between Git references and enforce risk policy thresholds.

Designed specifically for CI/CD integration - fails with non-zero exit code when files
exceed acceptable risk levels. Analyzes only the changed files, making it fast and focused.

Default thresholds: 50.0 for all modes (hot, risk, complexity, stale)

Use cases:
- Pull request gates - block merges with high-risk changes
- Release validation - ensure no critical files before deployment
- Quality enforcement - maintain code health standards
- Prevent regression - catch risk increases automatically

Examples:
  # Check PR changes against main branch
  hotspot check --base-ref origin/main --target-ref HEAD

  # Custom thresholds per mode
  hotspot check --base-ref main --target-ref feature --thresholds-override "hot:75,risk:60,complexity:80,stale:70"

  # Check release candidate
  hotspot check --base-ref v1.0.0 --target-ref v1.1.0-rc1

  # Focus on complexity in recent changes
  hotspot check --mode complexity --lookback "7 days" --thresholds-override "complexity:70"`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		// Validation is done in ExecuteHotspotCheck
		if err := core.ExecuteHotspotCheck(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Policy check failed", err)
		}
	},
}

// checkCompareAndExecute validates compare mode and executes the given function.
func checkCompareAndExecute(executeFunc core.ExecutorFunc) {
	if !cfg.CompareMode {
		contract.LogFatal("Cannot run compare analysis", errors.New("base and target refs must be provided"))
	}
	if err := executeFunc(rootCtx, cfg, cacheManager); err != nil {
		contract.LogFatal("Cannot run compare analysis", err)
	}
}

// compareFilesCmd looks at file deltas.
var compareFilesCmd = &cobra.Command{
	Use:   "files [repo-path]",
	Short: "Compare file-level risk metrics between Git references",
	Long: `Compare individual file risk scores between two points in repository history.

This helps you understand which files have become riskier or safer, making it ideal for:
- Release audits - see what changed between versions
- Refactoring validation - verify improvements actually reduced risk
- Sprint reviews - track risk trends over development cycles
- Pre-merge checks - ensure PRs don't introduce high-risk files

The comparison shows before/after scores, deltas, and ranking changes for each file.

Examples:
  # Compare files between releases
  hotspot compare files --base-ref v1.0.0 --target-ref v1.1.0

  # See complexity changes in feature branch
  hotspot compare files --mode complexity --base-ref main --target-ref feature-xyz

  # Check last 30 days of changes
  hotspot compare files --lookback "30 days"

  # Export comparison to CSV for tracking
  hotspot compare files --base-ref v1.0.0 --target-ref HEAD --output csv --output-file comparison.csv`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		checkCompareAndExecute(core.ExecuteHotspotCompare)
	},
}

// compareFoldersCmd looks at folder deltas.
var compareFoldersCmd = &cobra.Command{
	Use:   "folders [repo-path]",
	Short: "Compare folder-level risk metrics between Git references",
	Long: `Compare folder/directory risk scores between two points in repository history.

Provides a high-level view of subsystem health changes, ideal for:
- Architecture reviews - identify which subsystems are deteriorating
- Team allocation - find areas needing more attention
- Migration planning - track improvements during rewrites
- Quarterly planning - strategic risk assessment

Examples:
  # Compare subsystem health between releases
  hotspot compare folders --base-ref v2.0.0 --target-ref v2.1.0

  # Check if refactoring improved core modules
  hotspot compare folders --mode complexity --base-ref before-refactor --target-ref after-refactor

  # Monitor risk trends over 6 months
  hotspot compare folders --lookback "6 months"`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		checkCompareAndExecute(core.ExecuteHotspotCompareFolders)
	},
}

// versionCmd show the verbose version for diagnostic purposes.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of hotspot.",
	Long: `Display version information including build details.

Shows:
- Release version
- Git commit hash
- Build timestamp
- Go runtime version

Useful for:
- Debugging compatibility issues
- Verifying correct binary installation
- Reporting bugs with version details`,
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Printf("hotspot CLI\n")
		cmd.Printf("  Version: %s\n", version)
		cmd.Printf("  Commit:  %s\n", commit)
		cmd.Printf("  Built:   %s\n", date)
		cmd.Printf("  Runtime: %s\n", runtime.Version())
	},
}

// metricsCmd displays the formal definitions of all scoring modes.
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display mathematical formulas and definitions for all scoring modes",
	Long: `Show the formal definitions, formulas, and factor weights for all scoring modes.

Provides complete transparency into how files are ranked, including:
- Scoring mode purpose and focus
- Factor names and their contribution weights
- Mathematical formula for score calculation
- Custom weights if configured via .hotspot.yaml

No Git analysis is performed - this is purely informational.

Use this to:
- Understand what each scoring mode measures
- Explain scoring logic to your team
- Validate custom weight configurations
- Document scoring methodology

Examples:
  # Show default scoring formulas
  hotspot metrics

  # View with custom weights from config file
  hotspot metrics --config .hotspot.yaml`,
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotMetrics(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Cannot display metrics", err)
		}
	},
}

// timeseriesCmd analyzes the timeseries of hotspot scores for a specific path.
var timeseriesCmd = &cobra.Command{
	Use:   "timeseries [repo-path]",
	Short: "Track how risk scores change over time for a specific file or folder",
	Long: `Analyze the trend of hotspot scores over time for a single file or folder path.

Shows score evolution across multiple time periods, helping you:
- Identify when risk started increasing
- Validate that refactoring reduced risk over time
- Track maintenance debt accumulation
- Understand long-term file health trends

The analysis divides your specified interval into equal time windows and computes
the score for each period, showing the complete historical trajectory.

Examples:
  # Track complexity of main.go over 6 months (3 data points)
  hotspot timeseries --path main.go --mode complexity --interval "6 months" --points 3

  # See how core/ folder risk evolved over a year
  hotspot timeseries --path core/ --mode risk --interval "1 year" --points 4

  # Monitor stale score changes recently
  hotspot timeseries --path src/api.go --mode stale --interval "90 days" --points 6

  # Check if refactoring improved utils/ folder
  hotspot timeseries --path internal/utils/ --interval "120 days" --points 4`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotTimeseries(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Cannot run timeseries analysis", err)
		}
	},
}

// cacheCmd focused on cache management.
//
// Note: Cache subcommands use minimal initialization (initCacheConfig) instead of
// the full sharedSetup used by analysis commands. This avoids Git repo validation
// and complex config processing for simple cache operations.
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage Git activity cache (improves performance)",
	Long: `Manage the Git activity cache that speeds up repeated analyses.

Hotspot caches Git log aggregation results to avoid re-parsing history on every run.
This dramatically improves performance when analyzing the same repository multiple times.

Supported backends: SQLite (default), MySQL, PostgreSQL, or None (in-memory)

Subcommands:
  status - Show cache statistics and connection info
  clear  - Remove all cached data

Examples:
  # Check cache status
  hotspot cache status

  # Clear cache after major repository changes
  hotspot cache clear`,
}

// cacheClearCmd clears the cache.
var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all cached Git activity data",
	Long: `Delete all cached Git activity data from the configured backend.

Use this when:
- Repository history was rewritten (rebase, force push)
- Cache may be stale or corrupted
- Testing performance without cache
- Switching analysis time ranges significantly

For SQLite: Deletes the database file
For MySQL/PostgreSQL: Drops the cache table

Examples:
  # Clear SQLite cache (default)
  hotspot cache clear

  # Clear MySQL cache (set connection string via env variable)
  HOTSPOT_CACHE_BACKEND=mysql HOTSPOT_CACHE_DB_CONNECT="..." hotspot cache clear`,
	PreRunE: cacheSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := iocache.ClearCache(cfg.CacheBackend, contract.GetCacheDBFilePath(), cfg.CacheDBConnect); err != nil {
			contract.LogFatal("Failed to clear cache", err)
		}
		fmt.Println("Cache cleared successfully.")
	},
}

// cacheStatusCmd shows cache status.
var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display cache statistics and connection details",
	Long: `Show detailed information about the Git activity cache.

Displays:
- Backend type and connection status
- Total number of cached entries
- Last and oldest cache entry timestamps
- Cache database size

Use this to:
- Verify cache is working and connected
- Monitor cache growth over time
- Check when cache was last updated
- Debug cache-related issues

Examples:
  # Check cache status
  hotspot cache status`,
	PreRunE: cacheSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		status, err := iocache.Manager.GetActivityStore().GetStatus()
		if err != nil {
			contract.LogFatal("Failed to get cache status", err)
		}
		iocache.PrintCacheStatus(status)
	},
}

// analysisCmd focused on analysis data management.
//
// Note: Analysis subcommands use minimal initialization (analysisSetup) instead of
// the full sharedSetup used by analysis commands. This avoids Git repo validation
// and complex config processing for simple analysis operations.
var analysisCmd = &cobra.Command{
	Use:   "analysis",
	Short: "Manage historical analysis tracking and exports",
	Long: `Manage historical analysis data used for trend tracking and reporting.

When enabled, Hotspot tracks every analysis run, storing:
- Run metadata (timestamp, configuration, duration)
- File scores across all modes (hot, risk, complexity, stale)
- Raw Git metrics (commits, churn, contributors, etc.)

This enables longitudinal analysis, trend detection, and data export for BI tools.

Supported backends: SQLite (default), MySQL, PostgreSQL, or None (disabled)

Subcommands:
  status  - Show analysis tracking statistics
  export  - Export data to Parquet for analytics
  clear   - Remove all tracking data
  migrate - Run database schema migrations

Examples:
  # Check tracking status
  hotspot analysis status

  # Export for analysis in pandas/DuckDB
  hotspot analysis export --output-file analysis-data.parquet`,
}

// analysisClearCmd clears the analysis data.
var analysisClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all historical analysis tracking data",
	Long: `Delete all stored analysis runs and file score history.

This removes:
- All analysis run metadata
- Historical file scores across all modes
- Raw Git metrics for analyzed files

WARNING: This action cannot be undone. Consider exporting data first.

Use this when:
- Resetting trend tracking
- Database storage is full
- Starting fresh analysis history
- Testing analysis features

Examples:
  # Export before clearing
  hotspot analysis export --output-file backup.parquet
  hotspot analysis clear

  # Clear and start fresh
  hotspot analysis clear`,
	PreRunE: analysisSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := iocache.ClearAnalysis(cfg.AnalysisBackend, contract.GetAnalysisDBFilePath(), cfg.AnalysisDBConnect); err != nil {
			contract.LogFatal("Failed to clear analysis data", err)
		}
		fmt.Println("Analysis data cleared successfully.")
	},
}

// analysisStatusCmd shows analysis status.
var analysisStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display analysis tracking statistics and connection details",
	Long: `Show detailed information about historical analysis tracking.

Displays:
- Backend type and connection status
- Total number of analysis runs stored
- Last and oldest analysis run timestamps
- Total files analyzed across all runs
- Database table sizes

Use this to:
- Verify analysis tracking is enabled and working
- Monitor data accumulation over time
- Check database connection health
- Estimate storage requirements

Examples:
  # Check analysis tracking status
  hotspot analysis status`,
	PreRunE: analysisSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		status, err := iocache.Manager.GetAnalysisStore().GetStatus()
		if err != nil {
			contract.LogFatal("Failed to get analysis status", err)
		}
		iocache.PrintAnalysisStatus(status)
	},
}

// analysisExportCmd exports analysis data to Parquet files.
var analysisExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export historical data to Parquet for BI tools and analytics",
	Long: `Export all stored analysis data to Parquet format for use with analytics tools.

Exports two datasets:
- Analysis runs - metadata about each analysis execution
- File scores/metrics - detailed scores and Git metrics per file

Parquet format enables:
- Fast querying with DuckDB, Apache Spark, pandas
- Efficient storage with columnar compression
- Schema evolution for future data additions
- Direct import into BI tools (Tableau, Metabase, etc.)

Requires: --output-file parameter

Use cases:
- Trend analysis across multiple runs
- Custom dashboards and visualizations
- ML model training on code metrics
- Executive reporting and KPIs

Examples:
  # Export all data
  hotspot analysis export --output-file hotspot-data.parquet

  # Use with DuckDB for analysis
  hotspot analysis export --output-file data.parquet
  duckdb -c "SELECT * FROM read_parquet('data.parquet/runs.parquet') LIMIT 10"`,
	PreRunE: analysisSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := iocache.ExecuteAnalysisExport(cfg.OutputFile); err != nil {
			contract.LogFatal("Failed to export analysis data", err)
		}
	},
}

// analysisMigrateCmd runs database migrations for the analysis store.
var analysisMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database schema migrations (upgrades/downgrades)",
	Long: `Manage database schema versions for the analysis tracking store.

Migrations allow:
- Upgrading to new schema versions when Hotspot is updated
- Safely modifying database structure without data loss
- Rolling back schema changes if needed
- Testing new features on specific schema versions

By default, migrates to the latest version. Use --target-version for specific versions.

Examples:
  # Migrate to latest version (default)
  hotspot analysis migrate

  # Migrate to specific version
  hotspot analysis migrate --target-version 2

  # Rollback to previous version
  hotspot analysis migrate --target-version 0`,
	PreRunE: analysisMigrateSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		targetVersion := viper.GetInt("target-version")
		if err := iocache.MigrateAnalysis(cfg.AnalysisBackend, cfg.AnalysisDBConnect, targetVersion); err != nil {
			contract.LogFatal("Failed to run migrations", err)
		}
	},
}

// init defines and binds all flags.
func init() {
	// Call initConfig on Cobra's initialization
	cobra.OnInitialize(initConfig)

	// Add primary subcommands to the root command
	rootCmd.AddCommand(filesCmd)
	rootCmd.AddCommand(foldersCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(timeseriesCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(metricsCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(analysisCmd)

	// Add the compare subcommands to the parent compare command
	compareCmd.AddCommand(compareFilesCmd)
	compareCmd.AddCommand(compareFoldersCmd)

	// Add the cache subcommands to the parent cache command
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheStatusCmd)

	// Add the analysis subcommands to the parent analysis command
	analysisCmd.AddCommand(analysisClearCmd)
	analysisCmd.AddCommand(analysisStatusCmd)
	analysisCmd.AddCommand(analysisExportCmd)
	analysisCmd.AddCommand(analysisMigrateCmd)

	// Bind all persistent flags of rootCmd to Viper
	rootCmd.PersistentFlags().Bool("detail", false, "Print per-target metadata (lines of code, size, age)")
	rootCmd.PersistentFlags().String("end", "", "End date in ISO8601 or time ago")
	rootCmd.PersistentFlags().String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore")
	rootCmd.PersistentFlags().StringP("filter", "f", "", "Filter targets by path prefix")
	rootCmd.PersistentFlags().IntP("limit", "l", contract.DefaultResultLimit, "Number of results to display")
	rootCmd.PersistentFlags().String("mode", string(schema.HotMode), "Scoring mode: hot or risk or complexity or stale")
	rootCmd.PersistentFlags().String("output", string(schema.TextOut), "Output format: text or csv or json or parquet")
	rootCmd.PersistentFlags().String("output-file", "", "Optional path to write output to")
	rootCmd.PersistentFlags().Bool("owner", false, "Print per-target owner")
	rootCmd.PersistentFlags().Int("precision", contract.DefaultPrecision, "Decimal precision for numeric columns")
	rootCmd.PersistentFlags().String("profile", "", "Enable profiling and write profiles to files with this prefix")
	rootCmd.PersistentFlags().String("start", "", "Start date in ISO8601 or time ago")
	rootCmd.PersistentFlags().Int("workers", contract.DefaultWorkers, "Number of concurrent workers")
	rootCmd.PersistentFlags().Int("width", 0, "Terminal width override (0 = auto-detect)")
	rootCmd.PersistentFlags().String("cache-backend", string(schema.SQLiteBackend), "Cache backend: sqlite or mysql or postgresql or none")
	rootCmd.PersistentFlags().String("cache-db-connect", "", "Database connection string for mysql/postgresql (e.g., user:pass@tcp(host:port)/dbname)")
	rootCmd.PersistentFlags().String("analysis-backend", "", "Analysis tracking backend: sqlite or mysql or postgresql or none")
	rootCmd.PersistentFlags().String("analysis-db-connect", "", "Database connection string for analysis tracking (must differ from cache-db-connect)")
	rootCmd.PersistentFlags().String("color", "yes", "Enable colored labels in output (yes/no/true/false/1/0)")
	rootCmd.PersistentFlags().String("lookback", "6 months", "Time duration to look back from Base/Target ref commit time")
	rootCmd.PersistentFlags().String("base-ref", "", "Base Git reference for the BEFORE state")
	rootCmd.PersistentFlags().String("target-ref", "", "Target Git reference for the AFTER state")
	rootCmd.PersistentFlags().String("config", "", "Path to config file")
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		contract.LogFatal("Error binding root flags", err)
	}

	// Bind all flags of filesCmd to Viper
	filesCmd.Flags().Bool("explain", false, "Print per-file component score breakdown")
	filesCmd.Flags().Bool("follow", false, "Re-run per-file analysis with --follow")
	if err := viper.BindPFlags(filesCmd.Flags()); err != nil {
		contract.LogFatal("Error binding files flags", err)
	}

	// Bind all flags of timeseriesCmd to Viper
	timeseriesCmd.Flags().String("path", "", "Path to the file or folder to analyze")
	timeseriesCmd.Flags().String("interval", "3 months", "Total time interval")
	timeseriesCmd.Flags().Int("points", 3, "Number of lookback points")
	if err := viper.BindPFlags(timeseriesCmd.Flags()); err != nil {
		contract.LogFatal("Error binding timeseries flags", err)
	}

	// Bind all flags of checkCmd to Viper
	checkCmd.Flags().String("thresholds-override", "", "Risk thresholds for CI/CD gating (format: 'hot:50,risk:50,complexity:50,stale:50')")
	if err := viper.BindPFlags(checkCmd.Flags()); err != nil {
		contract.LogFatal("Error binding check flags", err)
	}

	// Bind all flags of analysisMigrateCmd to Viper
	analysisMigrateCmd.Flags().Int("target-version", -1, "Target migration version (-1 means latest, 0 means rollback to initial state)")
	if err := viper.BindPFlags(analysisMigrateCmd.Flags()); err != nil {
		contract.LogFatal("Error binding analysis migrate flags", err)
	}
}

// main starts the execution of the logic.
func main() {
	// Set the global caching manager (will be initialized in sharedSetup)
	cacheManager = iocache.Manager

	defer func() {
		// Close caching on exit
		iocache.CloseCaching()

		if err := stopProfiling(); err != nil {
			contract.LogFatal("Error stopping profiling", err)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		contract.LogFatal("Error starting CLI", err)
	}
}
