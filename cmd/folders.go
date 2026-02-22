package cmd

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/spf13/cobra"
)

// foldersCmd performs folder-level hotspot analysis.
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
