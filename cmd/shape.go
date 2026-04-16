package cmd

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/spf13/cobra"
)

var initShape bool

// shapeCmd analyzes the repository shape to recommend a configuration preset.
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
  # Print repo shape as JSON
  hotspot shape

  # Write recommended preset config as .hotspot.yml in the repo root
  hotspot shape --init`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotShape(rootCtx, cfg, gitClient, cacheManager, initShape); err != nil {
			logger.Fatal("Cannot run shape analysis", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(shapeCmd)
	shapeCmd.Flags().BoolVar(&initShape, "init", false, "Write recommended preset config as .hotspot.yml in the repo root (DEPRECATED: use 'hotspot init' instead)")
	_ = shapeCmd.Flags().MarkDeprecated("init", "use 'hotspot init' instead")
}
