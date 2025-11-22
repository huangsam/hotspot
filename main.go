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
	"github.com/huangsam/hotspot/internal/parquet"
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
	viper.SetDefault("emoji", "no")
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

// cacheSetup loads minimal configuration needed for cache operations.
// This is used by commands that need cache access without full shared setup.
func cacheSetup() error {
	// Handle config file
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".hotspot")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}

	// Load config file if present (similar to sharedSetup but minimal)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults/env/flags
	}

	// Get cache-related config values
	backend := schema.CacheBackend(viper.GetString("cache-backend"))
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
	// Handle config file
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".hotspot")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}

	// Load config file if present (similar to sharedSetup but minimal)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults/env/flags
	}

	// Get analysis-related config values
	backendStr := viper.GetString("analysis-backend")
	connStr := viper.GetString("analysis-db-connect")

	// Handle empty backend as NoneBackend
	var backend schema.CacheBackend
	if backendStr == "" {
		backend = schema.NoneBackend
	} else {
		backend = schema.CacheBackend(backendStr)
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

	// Get analysis-related config values
	backendStr := viper.GetString("analysis-backend")
	connStr := viper.GetString("analysis-db-connect")

	// Handle empty backend as NoneBackend
	var backend schema.CacheBackend
	if backendStr == "" {
		backend = schema.NoneBackend
	} else {
		backend = schema.CacheBackend(backendStr)
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
	Use:     "files [repo-path]",
	Short:   "Show the top files ranked by risk score.",
	Long:    `The files command performs deep Git analysis and ranks individual files.`,
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
	Use:     "folders [repo-path]",
	Short:   "Show the top folders ranked by risk score.",
	Long:    `The folders command performs deep Git analysis and ranks individual folders.`,
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
	Long:  `The compare command provides insight into how risk metrics have changed for different units (files, folders, etc.).`,
}

// checkCmd focused on CI/CD policy enforcement.
var checkCmd = &cobra.Command{
	Use:     "check [repo-path]",
	Short:   "Check files changed between refs against risk thresholds for CI/CD gating.",
	Long:    `The check command analyzes only files changed between two Git references and enforces policy thresholds. Designed specifically for CI/CD pipelines, it fails with a non-zero exit code if any file exceeds configured risk thresholds (default: 50.0 for all modes).`,
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
	Use:     "files [repo-path]",
	Short:   "Compare file-level risk metrics (the default unit of comparison).",
	Long:    `The files subcommand runs two separate file analyses (Base vs. Target) and reports change in risk scores.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		checkCompareAndExecute(core.ExecuteHotspotCompare)
	},
}

// compareFoldersCmd looks at folder deltas.
var compareFoldersCmd = &cobra.Command{
	Use:     "folders [repo-path]",
	Short:   "Compare folder-level risk metrics (the default unit of comparison).",
	Long:    `The folders subcommand runs two separate folder analyses (Base vs. Target) and reports change in risk scores.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		checkCompareAndExecute(core.ExecuteHotspotCompareFolders)
	},
}

// versionCmd show the verbose version for diagnostic purposes.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of hotspot",
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
	Use:     "metrics",
	Short:   "Display formal definitions of all scoring modes",
	Long:    `The metrics command shows the purpose, factors, and mathematical formulas for all four core scoring modes without performing Git analysis.`,
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotMetrics(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Cannot display metrics", err)
		}
	},
}

// timeseriesCmd analyzes the timeseries of hotspot scores for a specific path.
var timeseriesCmd = &cobra.Command{
	Use:     "timeseries [repo-path]",
	Short:   "Show timeseries of hotspot scores for a specific path.",
	Long:    `The timeseries command analyzes the hotspot score over time for a single file or folder.`,
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
	Short: "Manage cache operations.",
	Long:  `The cache command provides subcommands for managing the application's cache.`,
}

// cacheClearCmd clears the cache.
var cacheClearCmd = &cobra.Command{
	Use:     "clear",
	Short:   "Clear the cache for the configured backend.",
	Long:    `The clear subcommand removes all cached data for the current backend configuration.`,
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
	Use:     "status",
	Short:   "Show cache status information.",
	Long:    `The status subcommand displays information about the cache store including total entries, last run, and connection status.`,
	PreRunE: cacheSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		status, err := iocache.Manager.GetActivityStore().GetStatus()
		if err != nil {
			contract.LogFatal("Failed to get cache status", err)
		}
		printCacheStatus(status)
	},
}

// analysisCmd focused on analysis data management.
//
// Note: Analysis subcommands use minimal initialization (analysisSetup) instead of
// the full sharedSetup used by analysis commands. This avoids Git repo validation
// and complex config processing for simple analysis operations.
var analysisCmd = &cobra.Command{
	Use:   "analysis",
	Short: "Manage analysis data operations.",
	Long:  `The analysis command provides subcommands for managing the application's analysis data.`,
}

// analysisClearCmd clears the analysis data.
var analysisClearCmd = &cobra.Command{
	Use:     "clear",
	Short:   "Clear the analysis data for the configured backend.",
	Long:    `The clear subcommand removes all analysis tracking data for the current backend configuration.`,
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
	Use:     "status",
	Short:   "Show analysis status information.",
	Long:    `The status subcommand displays information about the analysis store including total runs, last run, and connection status.`,
	PreRunE: analysisSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		status, err := iocache.Manager.GetAnalysisStore().GetStatus()
		if err != nil {
			contract.LogFatal("Failed to get analysis status", err)
		}
		printAnalysisStatus(status)
	},
}

// analysisExportCmd exports analysis data to Parquet files.
var analysisExportCmd = &cobra.Command{
	Use:     "export",
	Short:   "Export analysis data to Parquet files.",
	Long:    `The export command reads analysis data and exports it to Parquet for analytics via Spark, Pandas, and DuckDB. Requires --output-file.`,
	PreRunE: analysisSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := executeAnalysisExport(); err != nil {
			contract.LogFatal("Failed to export analysis data", err)
		}
	},
}

// analysisMigrateCmd runs database migrations for the analysis store.
var analysisMigrateCmd = &cobra.Command{
	Use:     "migrate",
	Short:   "Run database schema migrations for the analysis store.",
	Long:    `The migrate command manages database schema evolution for the analysis store. By default, it migrates to the latest version.`,
	PreRunE: analysisMigrateSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		targetVersion := viper.GetInt("target-version")
		if err := iocache.MigrateAnalysis(cfg.AnalysisBackend, cfg.AnalysisDBConnect, targetVersion); err != nil {
			contract.LogFatal("Failed to run migrations", err)
		}
	},
}

// executeAnalysisExport performs the actual export of analysis data to Parquet files.
func executeAnalysisExport() error {
	// Export always uses parquet format, regardless of --output flag
	cfg.Output = schema.ParquetOut

	// Validate that output file is specified
	if cfg.OutputFile == "" {
		return errors.New("--output-file is required for export command")
	}

	// Get the analysis store
	store := iocache.Manager.GetAnalysisStore()

	// Check if there's any data to export
	status, err := store.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get analysis status: %w", err)
	}

	if status.TotalRuns == 0 {
		return errors.New("no analysis data found to export")
	}

	fmt.Printf("Exporting data from %s backend...\n", status.Backend)
	fmt.Printf("Total analysis runs: %d\n", status.TotalRuns)
	fmt.Printf("Total file records: %d\n", status.TableSizes["hotspot_file_scores_metrics"])

	// Retrieve all analysis runs
	analysisRuns, err := store.GetAllAnalysisRuns()
	if err != nil {
		return fmt.Errorf("failed to retrieve analysis runs: %w", err)
	}

	// Retrieve all file scores metrics
	fileMetrics, err := store.GetAllFileScoresMetrics()
	if err != nil {
		return fmt.Errorf("failed to retrieve file scores metrics: %w", err)
	}

	// Convert to Parquet format
	parquetAnalysisRuns := parquet.ConvertAnalysisRunRecords(analysisRuns)
	parquetFileMetrics := parquet.ConvertFileScoresMetricsRecords(fileMetrics)

	// Write analysis runs to Parquet
	analysisRunsFile := cfg.OutputFile + ".analysis_runs.parquet"
	if err := parquet.WriteAnalysisRunsParquet(parquetAnalysisRuns, analysisRunsFile); err != nil {
		return fmt.Errorf("failed to write analysis runs: %w", err)
	}
	fmt.Printf("Exported %d analysis runs to: %s\n", len(parquetAnalysisRuns), analysisRunsFile)

	// Write file scores metrics to Parquet
	fileMetricsFile := cfg.OutputFile + ".file_scores_metrics.parquet"
	if err := parquet.WriteFileScoresMetricsParquet(parquetFileMetrics, fileMetricsFile); err != nil {
		return fmt.Errorf("failed to write file scores metrics: %w", err)
	}
	fmt.Printf("Exported %d file score records to: %s\n", len(parquetFileMetrics), fileMetricsFile)

	fmt.Println("\nExport complete! The Parquet files can be used with:")
	fmt.Println("  - Apache Spark")
	fmt.Println("  - Apache Arrow")
	fmt.Println("  - Pandas (via pyarrow)")
	fmt.Println("  - DuckDB")
	fmt.Println("  - Any other Parquet-compatible tool")

	return nil
}

// printCacheStatus prints cache status information.
func printCacheStatus(status schema.CacheStatus) {
	fmt.Printf("Cache Backend: %s\n", status.Backend)
	fmt.Printf("Connected: %t\n", status.Connected)
	if !status.Connected {
		return
	}
	fmt.Printf("Total Entries: %d\n", status.TotalEntries)
	if status.TotalEntries > 0 {
		fmt.Printf("Last Entry: %s\n", status.LastEntryTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Oldest Entry: %s\n", status.OldestEntryTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("Table Size: %d bytes\n", status.TableSizeBytes)
}

// printAnalysisStatus prints analysis status information.
func printAnalysisStatus(status schema.AnalysisStatus) {
	fmt.Printf("Analysis Backend: %s\n", status.Backend)
	fmt.Printf("Connected: %t\n", status.Connected)
	if !status.Connected {
		return
	}
	fmt.Printf("Total Runs: %d\n", status.TotalRuns)
	if status.TotalRuns > 0 {
		fmt.Printf("Last Run ID: %d\n", status.LastRunID)
		fmt.Printf("Last Run: %s\n", status.LastRunTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Oldest Run: %s\n", status.OldestRunTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Total Files Analyzed: %d\n", status.TotalFilesAnalyzed)
	}
	fmt.Println("Table Sizes:")
	for table, size := range status.TableSizes {
		fmt.Printf("  %s: %d rows\n", table, size)
	}
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

	// Add the file comparison subcommand to the parent compare command
	compareCmd.AddCommand(compareFilesCmd)
	compareCmd.AddCommand(compareFoldersCmd)

	// Add the clear subcommand to the parent cache command
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheStatusCmd)

	// Add the clear subcommand to the parent analysis command
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
	rootCmd.PersistentFlags().String("emoji", "no", "Enable emojis in output headers (yes/no/true/false/1/0)")
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
