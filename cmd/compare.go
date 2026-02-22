package cmd

import (
	"errors"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/spf13/cobra"
)

// compareCmd focused on strategic per-target comparisons.
var compareCmd = &cobra.Command{
	Use:   "compare [repo-path]",
	Short: "Compare analysis results between two Git references.",
	Long: `Compare analysis results between two Git references to track how risk has evolved.

Ideal for:
- Release comparisons - see what changed between versions
- Refactoring validation - verify changes reduced risk
- Feature branch reviews - ensure PRs don't introduce high-risk files
- Progress tracking - monitor improvements over time
- Regression detection - catch files becoming riskier

Available comparison types:
  compare files   - Track individual file risk changes
  compare folders - Monitor subsystem health changes

Each comparison shows before/after scores, deltas, and ranking changes.`,
}

// checkCompareAndExecute validates compare mode and executes the given function.
func checkCompareAndExecute(executeFunc core.ExecutorFunc) {
	if !cfg.CompareMode {
		contract.LogFatal("Cannot run compare analysis", errors.New("base and target refs must be provided"))
	}
	if err := executeFunc(rootCtx, cfg, cacheManager); err != nil {
		contract.LogFatal("Cannot run compare analysis", err)
	}
}

// compareFilesCmd looks at file deltas.
var compareFilesCmd = &cobra.Command{
	Use:   "files [repo-path]",
	Short: "Compare file-level risk metrics between Git references",
	Long: `Compare individual file risk scores between two points in repository history.

This helps you understand which files have become riskier or safer, making it ideal for:
- Release audits - see what changed between versions
- Refactoring validation - verify improvements actually reduced risk
- Sprint reviews - track risk trends over development cycles
- Pre-merge checks - ensure PRs don't introduce high-risk files

The comparison shows before/after scores, deltas, and ranking changes for each file.

Examples:
  # Compare files between releases
  hotspot compare files --base-ref v1.0.0 --target-ref v1.1.0

  # See complexity changes in feature branch
  hotspot compare files --mode complexity --base-ref main --target-ref feature-xyz

  # Check last 30 days of changes
  hotspot compare files --lookback "30 days"

  # Export comparison to CSV for tracking
  hotspot compare files --base-ref v1.0.0 --target-ref HEAD --output csv --output-file comparison.csv`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		checkCompareAndExecute(core.ExecuteHotspotCompare)
	},
}

// compareFoldersCmd looks at folder deltas.
var compareFoldersCmd = &cobra.Command{
	Use:   "folders [repo-path]",
	Short: "Compare folder-level risk metrics between Git references",
	Long: `Compare folder/directory risk scores between two points in repository history.

Provides a high-level view of subsystem health changes, ideal for:
- Architecture reviews - identify which subsystems are deteriorating
- Team allocation - find areas needing more attention
- Migration planning - track improvements during rewrites
- Quarterly planning - strategic risk assessment

Examples:
  # Compare subsystem health between releases
  hotspot compare folders --base-ref v2.0.0 --target-ref v2.1.0

  # Check if refactoring improved core modules
  hotspot compare folders --mode complexity --base-ref before-refactor --target-ref after-refactor

  # Monitor risk trends over 6 months
  hotspot compare folders --lookback "6 months"`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		checkCompareAndExecute(core.ExecuteHotspotCompareFolders)
	},
}
