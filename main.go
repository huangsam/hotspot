// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	"errors"
	"runtime"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal"
	"github.com/spf13/cobra"
)

// All linker flags will be set by goreleaser infra at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// cfg will hold the validated, final configuration.
var cfg = &internal.Config{}

// input require processing/validation after flag parsing.
var input = &internal.ConfigRawInput{
	ResultLimit: internal.DefaultResultLimit,
	Workers:     internal.DefaultWorkers,
	Mode:        "hot",
	Precision:   internal.DefaultPrecision,
	Output:      "text",
}

// rootCmd is the command-line entrypoint for all other commands.
var rootCmd = &cobra.Command{
	Use:   "hotspot",
	Short: "Analyze Git repository activity to find code hotspots.",
	Long:  `Hotspot cuts through Git history to show you which files and folders are your greatest risk.`,

	// Set the application version here
	Version: version,

	// Just let the main function print the error.
	SilenceErrors: true,

	// The root command no longer executes logic, it just lists subcommands.
	Run: func(cmd *cobra.Command, _ []string) {
		// If no subcommand is provided, print help
		_ = cmd.Help()
	},
}

// sharedSetup sets up the Config object for all subcommands.
func sharedSetup(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		input.RepoPathStr = args[0]
	} else {
		input.RepoPathStr = "."
	}

	// Run all validation and complex parsing, including Git path resolution
	client := internal.NewLocalGitClient()
	return internal.ProcessAndValidate(cfg, client, input)
}

// filesCmd focuses on tactical, file-level analysis.
var filesCmd = &cobra.Command{
	Use:   "files [repo-path]",
	Short: "Show the top files ranked by risk score.",
	Long:  `The files command performs deep Git analysis and ranks individual files.`,
	Args:  cobra.MaximumNArgs(1),

	PreRunE: sharedSetup,

	Run: func(_ *cobra.Command, _ []string) {
		core.ExecuteHotspotFiles(cfg)
	},
}

// foldersCmd focuses on tactical, folder-level analysis.
var foldersCmd = &cobra.Command{
	Use:   "folders [repo-path]",
	Short: "Show the top folders ranked by risk score.",
	Long:  `The folders command performs deep Git analysis and ranks individual folders.`,
	Args:  cobra.MaximumNArgs(1),

	PreRunE: sharedSetup,

	Run: func(_ *cobra.Command, _ []string) {
		core.ExecuteHotspotFolders(cfg)
	},
}

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare analysis results between two Git references.",
	Long:  `The compare command provides insight into how risk metrics have changed for different units (files, folders, etc.).`,
}

var compareFilesCmd = &cobra.Command{
	Use:   "files [repo-path]",
	Short: "Compare file-level risk metrics (the default unit of comparison).",
	Long:  `The files subcommand runs two separate file analyses (Base vs. Target) and reports change in risk scores.`,
	Args:  cobra.MaximumNArgs(1),

	PreRunE: sharedSetup,

	Run: func(_ *cobra.Command, _ []string) {
		// Only execute comparison if the comparison mode has been turned on
		if cfg.CompareMode {
			core.ExecuteHotspotCompare(cfg)
		} else {
			// This should ideally be caught in sharedSetup, but serves as a fallback.
			internal.LogFatal("Cannot run compare analysis", errors.New("compare mode is off"))
		}
	},
}

var compareFoldersCmd = &cobra.Command{
	Use:   "folders [repo-path]",
	Short: "Compare folder-level risk metrics (the default unit of comparison).",
	Long:  `The files subcommand runs two separate folder analyses (Base vs. Target) and reports change in risk scores.`,
	Args:  cobra.MaximumNArgs(1),

	PreRunE: sharedSetup,

	Run: func(_ *cobra.Command, _ []string) {
		// Only execute comparison if the comparison mode has been turned on
		if cfg.CompareMode {
			core.ExecuteHotspotCompareFolders(cfg)
		} else {
			// This should ideally be caught in sharedSetup, but serves as a fallback.
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
	// Add primary subcommands to the root command
	rootCmd.AddCommand(filesCmd)
	rootCmd.AddCommand(foldersCmd)
	rootCmd.AddCommand(compareCmd) // Add the new parent compare command
	rootCmd.AddCommand(versionCmd)

	// Add the file comparison subcommand to the parent compare command
	compareCmd.AddCommand(compareFilesCmd)
	compareCmd.AddCommand(compareFoldersCmd)

	// --- Bind Simple Global Flags as PERSISTENT Flags (Available and Visible to ALL subcommands) ---
	rootCmd.PersistentFlags().StringVarP(&cfg.PathFilter, "filter", "f", "", "Filter files by path prefix")
	rootCmd.PersistentFlags().StringVar(&cfg.OutputFile, "output-file", "", "Optional path to write output to")

	// --- Bind Complex Global Flags as PERSISTENT Flags to Raw Input Struct (Available and Visible to ALL subcommands) ---
	rootCmd.PersistentFlags().IntVarP(&input.ResultLimit, "limit", "l", input.ResultLimit, "Number of results to display (files or folders)")
	rootCmd.PersistentFlags().StringVar(&input.StartTimeStr, "start", "", "Start date in ISO8601 or time ago")
	rootCmd.PersistentFlags().StringVar(&input.EndTimeStr, "end", "", "End date in ISO8601 or time ago")
	rootCmd.PersistentFlags().IntVar(&input.Workers, "workers", input.Workers, "Number of concurrent workers")
	rootCmd.PersistentFlags().StringVar(&input.Mode, "mode", input.Mode, "Scoring mode: hot or risk or complexity or stale")
	rootCmd.PersistentFlags().StringVar(&input.ExcludeStr, "exclude", "", "Comma-separated list of path prefixes or patterns to ignore")
	rootCmd.PersistentFlags().IntVar(&input.Precision, "precision", input.Precision, "Decimal precision for numeric columns")
	rootCmd.PersistentFlags().StringVar(&input.Output, "output", input.Output, "Output format: text or csv or json")

	// --- Bind Flags Specific to `hotspot files` ---
	filesCmd.Flags().BoolVar(&cfg.Detail, "detail", false, "Print per-file metadata (lines of code, size, age)")
	filesCmd.Flags().BoolVar(&cfg.Explain, "explain", false, "Print per-file component score breakdown")
	filesCmd.Flags().BoolVar(&cfg.Owner, "owner", false, "Print per-file owner")
	filesCmd.Flags().BoolVar(&cfg.Follow, "follow", false, "Re-run per-file analysis with --follow (slower)")

	// --- Bind Flags Specific to `hotspot folders` ---
	foldersCmd.Flags().BoolVar(&cfg.Owner, "owner", false, "Print per-folder owner")

	// --- Bind Complex Flags as PERSISTENT Flags to ALL compare subcommands ---
	compareCmd.PersistentFlags().StringVar(&input.BaseRefStr, "base-ref", "", "Base Git reference for the BEFORE state")
	compareCmd.PersistentFlags().StringVar(&input.TargetRefStr, "target-ref", "", "Target Git reference for the AFTER state")
	compareCmd.PersistentFlags().StringVar(&input.LookbackStr, "lookback", "6 months", "Time duration to look back from Base/Target ref commit time")
	compareCmd.PersistentFlags().BoolVar(&cfg.Detail, "detail", false, "Print additional per-target info")
}

// main starts the execution of the logic.
func main() {
	if err := rootCmd.Execute(); err != nil {
		internal.LogFatal("Error", err)
	}
}
