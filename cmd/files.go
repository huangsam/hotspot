package cmd

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/spf13/cobra"
)

// filesCmd performs file-level hotspot analysis.
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
