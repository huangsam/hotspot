// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper" // Import Viper
)

// All linker flags will be set by goreleaser infra at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// cfg will hold the validated, final configuration.
var cfg = &internal.Config{}

// input holds the raw, unvalidated configuration from all sources (file, env, flags).
// Viper will unmarshal into this struct.
var input = &internal.ConfigRawInput{}

// rootCmd is the command-line entrypoint for all other commands.
var rootCmd = &cobra.Command{
	Use:           "hotspot",
	Short:         "Analyze Git repository activity to find code hotspots.",
	Long:          `Hotspot cuts through Git history to show you which files and folders are your greatest risk.`,
	Version:       version,
	SilenceErrors: true,
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
func sharedSetup(_ *cobra.Command, args []string) error {
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
	return internal.ProcessAndValidate(cfg, client, input)
}

// filesCmd focuses on tactical, file-level analysis.
var filesCmd = &cobra.Command{
	Use:     "files [repo-path]",
	Short:   "Show the top files ranked by risk score.",
	Long:    `The files command performs deep Git analysis and ranks individual files.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetup,
	Run: func(_ *cobra.Command, _ []string) {
		core.ExecuteHotspotFiles(cfg)
	},
}

// foldersCmd focuses on tactical, folder-level analysis.
var foldersCmd = &cobra.Command{
	Use:     "folders [repo-path]",
	Short:   "Show the top folders ranked by risk score.",
	Long:    `The folders command performs deep Git analysis and ranks individual folders.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetup,
	Run: func(_ *cobra.Command, _ []string) {
		core.ExecuteHotspotFolders(cfg)
	},
}

// compareCmd focused on strategic per-file comparisons.
var compareCmd = &cobra.Command{
	Use:   "compare [repo-path]",
	Short: "Compare analysis results between two Git references.",
	Long:  `The compare command provides insight into how risk metrics have changed for different units (files, folders, etc.).`,
}

var compareFilesCmd = &cobra.Command{
	Use:     "files [repo-path]",
	Short:   "Compare file-level risk metrics (the default unit of comparison).",
	Long:    `The files subcommand runs two separate file analyses (Base vs. Target) and reports change in risk scores.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetup,
	Run: func(_ *cobra.Command, _ []string) {
		if cfg.CompareMode {
			core.ExecuteHotspotCompare(cfg)
		} else {
			internal.LogFatal("Cannot run compare analysis", errors.New("compare mode is off"))
		}
	},
}

var compareFoldersCmd = &cobra.Command{
	Use:     "folders [repo-path]",
	Short:   "Compare folder-level risk metrics (the default unit of comparison).",
	Long:    `The files subcommand runs two separate folder analyses (Base vs. Target) and reports change in risk scores.`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetup,
	Run: func(_ *cobra.Command, _ []string) {
		if cfg.CompareMode {
			core.ExecuteHotspotCompareFolders(cfg)
		} else {
			internal.LogFatal("Cannot run compare analysis", errors.New("compare mode is off"))
		}
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

// init defines and binds all flags.
func init() {
	// Call initConfig on Cobra's initialization
	cobra.OnInitialize(initConfig)

	// Add primary subcommands to the root command
	rootCmd.AddCommand(filesCmd)
	rootCmd.AddCommand(foldersCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(versionCmd)

	// Add the file comparison subcommand to the parent compare command
	compareCmd.AddCommand(compareFilesCmd)
	compareCmd.AddCommand(compareFoldersCmd)

	// --- Bind Simple Global Flags as PERSISTENT Flags ---
	// Note: We no longer bind to a struct variable, just define the flag.
	rootCmd.PersistentFlags().StringP("filter", "f", "", "Filter files by path prefix")
	rootCmd.PersistentFlags().String("output-file", "", "Optional path to write output to")

	// --- Bind Complex Global Flags as PERSISTENT Flags ---
	rootCmd.PersistentFlags().IntP("limit", "l", internal.DefaultResultLimit, "Number of results to display (files or folders)")
	rootCmd.PersistentFlags().String("start", "", "Start date in ISO8601 or time ago")
	rootCmd.PersistentFlags().String("end", "", "End date in ISO8601 or time ago")
	rootCmd.PersistentFlags().Int("workers", internal.DefaultWorkers, "Number of concurrent workers")
	rootCmd.PersistentFlags().String("mode", schema.HotMode, "Scoring mode: hot or risk or complexity or stale")
	rootCmd.PersistentFlags().String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore")
	rootCmd.PersistentFlags().Int("precision", internal.DefaultPrecision, "Decimal precision for numeric columns")
	rootCmd.PersistentFlags().String("output", schema.TextOut, "Output format: text or csv or json")

	// Bind all persistent flags of rootCmd to Viper
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}

	// --- Bind Flags Specific to `hotspot files` ---
	filesCmd.Flags().Bool("detail", false, "Print per-file metadata (lines of code, size, age)")
	filesCmd.Flags().Bool("explain", false, "Print per-file component score breakdown")
	filesCmd.Flags().Bool("owner", false, "Print per-file owner")
	filesCmd.Flags().Bool("follow", false, "Re-run per-file analysis with --follow (slower)")

	// Bind all flags of filesCmd to Viper
	if err := viper.BindPFlags(filesCmd.Flags()); err != nil {
		panic(err)
	}

	// --- Bind Flags Specific to `hotspot folders` ---
	foldersCmd.Flags().Bool("owner", false, "Print per-folder owner")

	// Bind all flags of foldersCmd to Viper
	if err := viper.BindPFlags(foldersCmd.Flags()); err != nil {
		panic(err)
	}

	// --- Bind Complex Flags as PERSISTENT Flags to ALL compare subcommands ---
	compareCmd.PersistentFlags().String("base-ref", "", "Base Git reference for the BEFORE state")
	compareCmd.PersistentFlags().String("target-ref", "", "Target Git reference for the AFTER state")
	compareCmd.PersistentFlags().String("lookback", "6 months", "Time duration to look back from Base/Target ref commit time")
	compareCmd.PersistentFlags().Bool("detail", false, "Print additional per-target info")

	// Bind all persistent flags of compareCmd to Viper
	if err := viper.BindPFlags(compareCmd.PersistentFlags()); err != nil {
		panic(err)
	}
}

// main starts the execution of the logic.
func main() {
	if err := rootCmd.Execute(); err != nil {
		internal.LogFatal("Error", err)
	}
}
