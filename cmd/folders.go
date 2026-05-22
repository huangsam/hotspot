package cmd

import (
	"fmt"

	"github.com/huangsam/hotspot/core"
	"github.com/spf13/cobra"
)

// foldersCmd performs folder-level hotspot analysis.
var foldersCmd = &cobra.Command{
	Use:   "folders [repo-path]",
	Short: "Rank folders by activity, risk, complexity, or refactoring ROI",
	Long: `Perform deep Git analysis and rank directories/folders by risk score.

Aggregates file-level analysis to folder level. Helps you:
- Identify which subsystems are risky or volatile
- Assess team/module boundaries
- Find areas that need architectural attention
- Plan refactoring efforts strategically

Each folder's score is weighted by file size and activity.

Examples:
  # Find the riskiest subsystems
  hotspot folders --mode hot

  # See which modules have knowledge concentration issues
  hotspot folders --mode risk

  # Identify complex subsystems worth refactoring
  hotspot folders --mode complexity

  # Prioritize large refactoring targets by ROI
  hotspot folders --mode roi

  # Include metrics and owner information
  hotspot folders --detail --owner`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := core.ExecuteHotspotFolders(cmd.Context(), cfg, gitClient, cacheManager, resultWriter); err != nil {
			return fmt.Errorf("cannot run folders analysis: %w", err)
		}
		return nil
	},
}
