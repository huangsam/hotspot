// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal"
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
var cfg = &internal.Config{}

// input holds the raw, unvalidated configuration from all sources (file, env, flags).
// Viper will unmarshal into this struct.
var input = &internal.ConfigRawInput{}

// profile holds profiling configuration
var profile = &internal.ProfileConfig{}

// startProfiling starts CPU and memory profiling if enabled
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
	fmt.Fprintf(os.Stderr, "Profiling enabled. CPU profile: %s.cpu.prof, Memory profile: %s.mem.prof\n", profile.Prefix, profile.Prefix)
	return nil
}

// stopProfiling stops profiling and writes memory profile
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

	fmt.Fprintf(os.Stderr, "Profiling complete. Use 'go tool pprof %s.cpu.prof' to analyze.\n", profile.Prefix)
	return nil
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
	// Set config file name and paths
	viper.SetConfigName(".hotspot") // Name of config file (without extension)
	viper.SetConfigType("yaml")     // We'll use YAML format
	viper.AddConfigPath(".")        // Look in the current directory
	viper.AddConfigPath("$HOME")    // Look in the home directory

	// Set environment variable prefix
	viper.SetEnvPrefix("HOTSPOT")
	viper.AutomaticEnv() // Read in environment variables that match

	// Set defaults in Viper
	viper.SetDefault("limit", internal.DefaultResultLimit)
	viper.SetDefault("workers", internal.DefaultWorkers)
	viper.SetDefault("mode", schema.HotMode)
	viper.SetDefault("precision", internal.DefaultPrecision)
	viper.SetDefault("output", schema.TextOut)
	viper.SetDefault("lookback", "6 months")
}

// sharedSetup unmarshals config and runs validation.
func sharedSetup(ctx context.Context, _ *cobra.Command, args []string) error {
	// Handle profiling flag
	profilePrefix := viper.GetString("profile")
	if err := internal.ProcessProfilingConfig(profile, profilePrefix); err != nil {
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
	client := internal.NewLocalGitClient()
	return internal.ProcessAndValidate(ctx, cfg, client, input)
}

// sharedSetupWrapper wraps sharedSetup to provide context for Cobra's PreRunE.
func sharedSetupWrapper(cmd *cobra.Command, args []string) error {
	return sharedSetup(rootCtx, cmd, args)
}

// filesCmd focuses on tactical, file-level analysis.
var filesCmd = &cobra.Command{
	Use:     "files [repo-path]",
	Short:   "Show the top files ranked by risk score.",
	Long:    `The files command performs deep Git analysis and ranks individual files.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotFiles(rootCtx, cfg); err != nil {
			internal.LogFatal("Cannot run files analysis", err)
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
		if err := core.ExecuteHotspotFolders(rootCtx, cfg); err != nil {
			internal.LogFatal("Cannot run folders analysis", err)
		}
	},
}

// compareCmd focused on strategic per-target comparisons.
var compareCmd = &cobra.Command{
	Use:   "compare [repo-path]",
	Short: "Compare analysis results between two Git references.",
	Long:  `The compare command provides insight into how risk metrics have changed for different units (files, folders, etc.).`,
}

// checkCompareAndExecute validates compare mode and executes the given function.
func checkCompareAndExecute(executeFunc core.ExecutorFunc) {
	if !cfg.CompareMode {
		internal.LogFatal("Cannot run compare analysis", errors.New("base and target refs must be provided"))
	}
	if err := executeFunc(rootCtx, cfg); err != nil {
		internal.LogFatal("Cannot run compare analysis", err)
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
	Use:   "metrics",
	Short: "Display formal definitions of all scoring modes",
	Long:  `The metrics command shows the purpose, factors, and mathematical formulas for all four core scoring modes without performing Git analysis.`,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotMetrics(); err != nil {
			internal.LogFatal("Cannot display metrics", err)
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
		if err := core.ExecuteHotspotTimeseries(rootCtx, cfg); err != nil {
			internal.LogFatal("Cannot run timeseries analysis", err)
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
	rootCmd.AddCommand(timeseriesCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(metricsCmd)

	// Add the file comparison subcommand to the parent compare command
	compareCmd.AddCommand(compareFilesCmd)
	compareCmd.AddCommand(compareFoldersCmd)

	// Bind all persistent flags of rootCmd to Viper
	rootCmd.PersistentFlags().Bool("detail", false, "Print per-target metadata (lines of code, size, age)")
	rootCmd.PersistentFlags().String("end", "", "End date in ISO8601 or time ago")
	rootCmd.PersistentFlags().String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore")
	rootCmd.PersistentFlags().StringP("filter", "f", "", "Filter targets by path prefix")
	rootCmd.PersistentFlags().IntP("limit", "l", internal.DefaultResultLimit, "Number of results to display")
	rootCmd.PersistentFlags().String("mode", schema.HotMode, "Scoring mode: hot or risk or complexity or stale")
	rootCmd.PersistentFlags().String("output", schema.TextOut, "Output format: text or csv or json")
	rootCmd.PersistentFlags().String("output-file", "", "Optional path to write output to")
	rootCmd.PersistentFlags().Bool("owner", false, "Print per-target owner")
	rootCmd.PersistentFlags().Int("precision", internal.DefaultPrecision, "Decimal precision for numeric columns")
	rootCmd.PersistentFlags().String("profile", "", "Enable profiling and write profiles to files with this prefix")
	rootCmd.PersistentFlags().String("start", "", "Start date in ISO8601 or time ago")
	rootCmd.PersistentFlags().Int("workers", internal.DefaultWorkers, "Number of concurrent workers")
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		internal.LogFatal("Error binding root flags", err)
	}

	// Bind all flags of filesCmd to Viper
	filesCmd.Flags().Bool("explain", false, "Print per-file component score breakdown")
	filesCmd.Flags().Bool("follow", false, "Re-run per-file analysis with --follow")
	if err := viper.BindPFlags(filesCmd.Flags()); err != nil {
		internal.LogFatal("Error binding files flags", err)
	}

	// Bind all persistent flags of compareCmd to Viper
	compareCmd.PersistentFlags().String("base-ref", "", "Base Git reference for the BEFORE state (required)")
	compareCmd.PersistentFlags().String("target-ref", "", "Target Git reference for the AFTER state (required)")
	compareCmd.PersistentFlags().String("lookback", "6 months", "Time duration to look back from Base/Target ref commit time")
	if err := viper.BindPFlags(compareCmd.PersistentFlags()); err != nil {
		internal.LogFatal("Error binding compare flags", err)
	}

	// Bind all flags of timeseriesCmd to Viper
	timeseriesCmd.Flags().String("path", "", "Path to the file or folder to analyze (required)")
	timeseriesCmd.Flags().String("interval", "", "Total time interval (e.g., 180d) (required)")
	timeseriesCmd.Flags().Int("points", 0, "Number of lookback points (required)")
	if err := viper.BindPFlags(timeseriesCmd.Flags()); err != nil {
		internal.LogFatal("Error binding timeseries flags", err)
	}
}

// main starts the execution of the logic.
func main() {
	defer func() {
		if err := stopProfiling(); err != nil {
			internal.LogFatal("Error stopping profiling", err)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		internal.LogFatal("Error starting CLI", err)
	}
}
