package cmd

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/spf13/cobra"
)

// timeseriesCmd analyzes hotspot scores over time for a specific path.
var timeseriesCmd = &cobra.Command{
	Use:   "timeseries [repo-path]",
	Short: "Track how risk scores change over time for a specific file or folder",
	Long: `Analyze the trend of hotspot scores over time for a single file or folder path.

Shows score evolution across multiple time periods, helping you:
- Identify when risk started increasing
- Validate that refactoring reduced risk over time
- Track maintenance debt accumulation
- Understand long-term file health trends

The analysis divides your specified interval into equal time windows and computes
the score for each period, showing the complete historical trajectory.

Examples:
  # Track complexity of main.go over 6 months (3 data points)
  hotspot timeseries --path main.go --mode complexity --interval "6 months" --points 3

  # See how core/ folder risk evolved over a year
  hotspot timeseries --path core/ --mode risk --interval "1 year" --points 4

  # Monitor stale score changes recently
  hotspot timeseries --path src/api.go --mode stale --interval "90 days" --points 6

  # Check if refactoring improved utils/ folder
  hotspot timeseries --path internal/utils/ --interval "120 days" --points 4`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := core.ExecuteHotspotTimeseries(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Cannot run timeseries analysis", err)
		}
	},
}
