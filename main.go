// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal"
	"github.com/spf13/cobra"
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
	Long:  `Hotspot cuts through history to show you which files are your greatest risk.`,

	// Just let the main function print the error.
	SilenceErrors: true,

	// PreRunE logic is being moved to subcommands to ensure
	// command-specific flags and arguments are processed just before execution.
	// The root command only handles the global flags.

	// The root command no longer executes logic, it just lists subcommands.
	Run: func(cmd *cobra.Command, _ []string) {
		// If no subcommand is provided, print help
		_ = cmd.Help()
	},
}

// sharedSetup sets up the Config object for all subcommands.
func sharedSetup(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		// Pass the user-provided path to raw input
		input.RepoPathStr = args[0]
	} else {
		// Pass the default path (CWD) to raw input, or use the root command's argument if present
		input.RepoPathStr = "."
	}

	// Run all validation and complex parsing, including Git path resolution
	return internal.ProcessAndValidate(cfg, input) // cfg.RepoPath is set inside here now
}

// filesCmd focuses on tactical, file-level analysis.
var filesCmd = &cobra.Command{
	Use:   "files [repo-path]",
	Short: "Show the top files ranked by risk score.",
	Long:  `The files command performs deep Git analysis and ranks individual files.`,
	Args:  cobra.MaximumNArgs(1),

	// PreRunE is moved here to ensure config validation runs right before execution.
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

	// PreRunE is moved here to ensure config validation runs right before execution.
	PreRunE: sharedSetup,

	Run: func(_ *cobra.Command, _ []string) {
		core.ExecuteHotspotFolders(cfg)
	},
}

// init defines and binds all flags.
func init() {
	// Add subcommands to the root command
	rootCmd.AddCommand(filesCmd)
	rootCmd.AddCommand(foldersCmd)

	// --- Bind Simple Global Flags as PERSISTENT Flags (Available and Visible to ALL subcommands) ---
	rootCmd.PersistentFlags().StringVarP(&cfg.PathFilter, "filter", "f", "", "Filter files by path prefix")
	rootCmd.PersistentFlags().StringVar(&cfg.OutputFile, "output-file", "", "Optional path to write output to")

	// --- Bind Complex Global Flags as PERSISTENT Flags to Raw Input Struct (Available and Visible to ALL subcommands) ---
	rootCmd.PersistentFlags().IntVarP(&input.ResultLimit, "limit", "l", input.ResultLimit, "Number of results to display (files or folders)")
	rootCmd.PersistentFlags().StringVar(&input.StartTimeStr, "start", "", "Start date in ISO8601 format")
	rootCmd.PersistentFlags().StringVar(&input.EndTimeStr, "end", "", "End date in ISO8601 format")
	rootCmd.PersistentFlags().IntVar(&input.Workers, "workers", input.Workers, "Number of concurrent workers")
	rootCmd.PersistentFlags().StringVar(&input.Mode, "mode", input.Mode, "Scoring mode: hot or risk or complexity or stale")
	rootCmd.PersistentFlags().StringVar(&input.ExcludeStr, "exclude", "", "Comma-separated list of path prefixes or patterns to ignore")
	rootCmd.PersistentFlags().IntVar(&input.Precision, "precision", input.Precision, "Decimal precision for numeric columns")
	rootCmd.PersistentFlags().StringVar(&input.Output, "output", input.Output, "Output format: text or csv or json")

	// --- Bind Flags Specific to `hotspot files` ---
	// These flags remain defined only on the 'files' subcommand.
	filesCmd.Flags().BoolVar(&cfg.Detail, "detail", false, "Print per-file metadata (lines of code, size, age)")
	filesCmd.Flags().BoolVar(&cfg.Explain, "explain", false, "Print per-file component score breakdown")
	filesCmd.Flags().BoolVar(&cfg.Owner, "owner", false, "Print per-file owner")
	filesCmd.Flags().BoolVar(&cfg.Follow, "follow", false, "Re-run per-file analysis with --follow (slower)")
}

// main starts the execution of the logic.
func main() {
	if err := rootCmd.Execute(); err != nil {
		internal.LogFatal("Error", err)
	}
}
