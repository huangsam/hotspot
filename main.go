// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	"fmt"
	"strings"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
)

// cfg will hold the validated, final configuration.
var cfg = &schema.Config{}

// input require processing/validation after flag parsing.
var input = &schema.ConfigRawInput{
	ResultLimit: schema.DefaultResultLimit,
	Workers:     schema.DefaultWorkers,
	Mode:        "hot",
	Precision:   schema.DefaultPrecision,
	Output:      "text",
}

// rootCmd is the command-line entrypoint for all other commands.
var rootCmd = &cobra.Command{
	Use:   "hotspot [repo-path]",
	Short: "Analyze Git repository activity to find code hotspots.",
	Long:  `Hotspot cuts through history to show you which files are your greatest risk.`,
	Args:  cobra.MaximumNArgs(1),

	// Just let the main function print the error. The cobra library
	// does not need to do it in this case
	SilenceErrors: true,

	// PreRunE handles validation and processing using the logic in schema/config.go
	PreRunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 1 {
			// Assume provided path is contained by a Git repo
			cfg.RepoPath = args[0]
		} else {
			// Assume current path is contained by a Git repo
			cfg.RepoPath = "."
		}

		// Run all validation and complex parsing
		return schema.ProcessAndValidate(cfg, input)
	},

	// Run executes the core business logic.
	Run: func(_ *cobra.Command, _ []string) {
		executeHotspot()
	},
}

// init defines and binds all flags.
func init() {
	// --- Bind Simple Flags Directly to Config ---
	// Note: We bind simple, non-parsing flags directly to the final 'cfg' to keep them clean.
	rootCmd.Flags().StringVarP(&cfg.PathFilter, "filter", "f", "", "Filter files by path prefix")
	rootCmd.Flags().BoolVar(&cfg.Detail, "detail", false, "Print per-file metadata")
	rootCmd.Flags().BoolVar(&cfg.Explain, "explain", false, "Print per-file component score breakdown")
	rootCmd.Flags().StringVar(&cfg.CSVFile, "csv-file", "", "Optional path to write CSV output directly")
	rootCmd.Flags().StringVar(&cfg.JSONFile, "json-file", "", "Optional path to write JSON output directly")
	rootCmd.Flags().BoolVar(&cfg.Follow, "follow", false, "Re-run per-file analysis with --follow (slower)")

	// --- Bind Complex Flags to Raw Input Struct ---
	// These flags use the ConfigInput struct as they require post-parsing validation/conversion.
	rootCmd.Flags().IntVarP(&input.ResultLimit, "limit", "l", input.ResultLimit, "Number of files to display")
	rootCmd.Flags().StringVar(&input.StartTimeStr, "start", "", "Start date in ISO8601 format")
	rootCmd.Flags().StringVar(&input.EndTimeStr, "end", "", "End date in ISO8601 format")
	rootCmd.Flags().IntVar(&input.Workers, "workers", input.Workers, "Number of concurrent workers")
	rootCmd.Flags().StringVar(&input.Mode, "mode", input.Mode, "Scoring mode: hot or risk or complexity or stale")
	rootCmd.Flags().StringVar(&input.ExcludeStr, "exclude", "", "Comma-separated list of path prefixes or patterns to ignore")
	rootCmd.Flags().IntVar(&input.Precision, "precision", input.Precision, "Decimal precision for numeric columns")
	rootCmd.Flags().StringVar(&input.Output, "output", input.Output, "Output format: text or csv or json")
}

// main starts the execution of the logic.
func main() {
	if err := rootCmd.Execute(); err != nil {
		internal.LogFatal("CLI error", err)
	}
}

// executeHotspot contains the application's main business logic.
func executeHotspot() {
	var files []string

	// --- 1. Aggregation Phase ---
	fmt.Printf("🔎 Aggregating recent activity since %s\n", cfg.StartTime.Format(schema.TimeFormat))
	if err := core.AggregateRecent(cfg); err != nil {
		internal.LogWarning("Cannot aggregate recent activity")
	}

	// --- 2. File List Building and Filtering ---
	// Build file list from union of recent maps so we only analyze files touched since StartTime
	seen := make(map[string]bool)

	// Add files seen in recent commit activity
	for k := range schema.GetRecentCommitsMapGlobal() {
		seen[k] = true
	}
	// Add files seen in recent churn activity
	for k := range schema.GetRecentChurnMapGlobal() {
		seen[k] = true
	}
	// Add files seen in recent contributor activity
	for k := range schema.GetRecentContribMapGlobal() {
		seen[k] = true
	}

	for f := range seen {
		// apply path filter
		if cfg.PathFilter != "" && !strings.HasPrefix(f, cfg.PathFilter) {
			continue
		}

		// apply excludes filter
		if internal.ShouldIgnore(f, cfg.Excludes) {
			continue
		}
		files = append(files, f)
	}

	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return
	}

	// --- 3. Core Analysis and Initial Ranking ---
	fmt.Printf("🧠 hotspot: Analyzing %s\n", cfg.RepoPath)
	fmt.Printf("📅 Range: %s → %s\n", cfg.StartTime.Format(schema.TimeFormat), cfg.EndTime.Format(schema.TimeFormat))

	results := core.AnalyzeRepo(cfg, files)
	ranked := core.RankFiles(results, cfg.ResultLimit)

	// --- 4. Optional --follow Re-analysis and Re-ranking ---
	// If the user requested a follow-pass, re-analyze the top N files using
	// git --follow to account for renames/history and then re-rank.
	if cfg.Follow && len(ranked) > 0 {
		// Determine the number of files to re-analyze (min of limit or actual results)
		n := min(cfg.ResultLimit, len(ranked))

		fmt.Printf("🔁 Running --follow re-analysis for top %d files...\n", n)

		for i := range n {
			f := ranked[i]

			// re-analyze with follow enabled (passing 'true' for the follow flag)
			rean := core.AnalyzeFileCommon(cfg, f.Path, true)

			// preserve path but update metrics and score
			rean.Path = f.Path
			ranked[i] = rean
		}

		// re-rank after follow pass
		ranked = core.RankFiles(ranked, cfg.ResultLimit)
	}

	// --- 5. Output Results ---
	internal.PrintResults(ranked, cfg)
}
