package cmd

import (
	"fmt"

	"github.com/huangsam/hotspot/core"
	"github.com/spf13/cobra"
)

// metricsCmd displays the formal definitions of all scoring modes.
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display mathematical formulas and definitions for all scoring modes",
	Long: `Show the formal definitions, formulas, and factor weights for all scoring modes.

Provides complete transparency into how files are ranked, including:
- Scoring mode purpose and focus
- Factor names and their contribution weights
- Mathematical formula for score calculation
- Custom weights if configured via .hotspot.yaml

No Git analysis is performed - this is purely informational.

Use this to:
- Understand what each scoring mode measures
- Explain scoring logic to your team
- Validate custom weight configurations
- Document scoring methodology

Examples:
  # Show default scoring formulas
  hotspot metrics

  # View with custom weights from config file
  hotspot metrics --config .hotspot.yaml`,
	PreRunE: sharedSetupWrapper,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := core.ExecuteHotspotMetrics(cmd.Context(), cfg, gitClient, cacheManager, resultWriter); err != nil {
			return fmt.Errorf("cannot display metrics: %w", err)
		}
		return nil
	},
}
