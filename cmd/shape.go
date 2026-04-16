package cmd

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/spf13/cobra"
)

var shapeCmd = &cobra.Command{
	Use:   "shape [repo-path]",
	Short: "Analyze repository shape and recommend a configuration preset.",
	Long: `Runs a lightweight aggregation pass over Git history to characterize the
repository and recommend a configuration preset (small, large, or infra).

The shape is derived from the first aggregation pass only — no per-file scoring
is performed — making it fast even for large repositories.

Use the recommended preset with 'hotspot files --preset <name>' or pass it to
any MCP tool as the 'preset' parameter.

Examples:
  # Analyze current repo shape
  hotspot shape

  # Analyze a different repo by path
  hotspot shape /path/to/repo`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotShape(rootCtx, cfg, gitClient, cacheManager); err != nil {
			logger.Fatal("Cannot run shape analysis", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(shapeCmd)
}
