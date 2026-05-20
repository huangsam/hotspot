package cmd

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/spf13/cobra"
)

// filesCmd performs file-level hotspot analysis.
var filesCmd = &cobra.Command{
	Use:   "files [repo-path]",
	Short: "Rank files by activity, risk, complexity, refactoring ROI, or composite signals",
	Long: `Perform deep Git analysis and rank individual files by risk score.

Analyzes the entire history of each file to compute risk metrics, helping you:
- Identify which files are most critical to your codebase
- Find files that are changing too frequently (churn hotspots)
- Spot files with uneven ownership and knowledge silos
- Locate large, complex files that are difficult to maintain

Scores files based on your selected mode:

Base Modes:
- hot: High recent activity and volatility (activity hotspots)
- risk: Few contributors or concentrated ownership (knowledge risk)
- complexity: Large, old, volatile files (technical debt)
- roi: High churn on complex files (refactoring priority)

Composite Modes (blend multiple base modes):
- active_owners: 50% hot + 50% risk (volatile + siloed code)
- refactor_now: 60% complexity + 40% roi (high ROI targets)
- legacy_debt: 70% complexity + 30% risk (fragile + under-maintained)

Examples:
  # Find the most active/volatile files
  hotspot files --mode hot --limit 20

  # Identify files with knowledge concentration risk
  hotspot files --mode risk

  # Find complex files by age and size
  hotspot files --mode complexity

  # Prioritize refactoring targets by ROI
  hotspot files --mode roi

  # Find files that are volatile AND have concentrated ownership
  hotspot files --mode active_owners --owner

  # Include detailed metrics and component breakdown
  hotspot files --detail --explain --owner

  # Export findings to CSV for tracking
  hotspot files --mode hot --output csv --output-file hotspots.csv`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(cmd *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotFiles(cmd.Context(), cfg, gitClient, cacheManager, resultWriter); err != nil {
			logger.Fatal("Cannot run files analysis", err)
		}
	},
}
