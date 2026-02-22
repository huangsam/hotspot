package cmd

import (
	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/spf13/cobra"
)

// checkCmd focused on CI/CD policy enforcement.
var checkCmd = &cobra.Command{
	Use:   "check [repo-path]",
	Short: "Enforce risk thresholds for CI/CD pipelines (fails build on violations)",
	Long: `Analyze ONLY changed files between Git references and enforce risk policy thresholds.

Designed specifically for CI/CD integration - fails with non-zero exit code when files
exceed acceptable risk levels. Analyzes only the changed files, making it fast and focused.

Default thresholds: 50.0 for all modes (hot, risk, complexity, stale)

Use cases:
- Pull request gates - block merges with high-risk changes
- Release validation - ensure no critical files before deployment
- Quality enforcement - maintain code health standards
- Prevent regression - catch risk increases automatically

Examples:
  # Check PR changes against main branch
  hotspot check --base-ref origin/main --target-ref HEAD

  # Custom thresholds per mode
  hotspot check --base-ref main --target-ref feature --thresholds-override "hot:75,risk:60,complexity:80,stale:70"

  # Check release candidate
  hotspot check --base-ref v1.0.0 --target-ref v1.1.0-rc1

  # Focus on complexity in recent changes
  hotspot check --mode complexity --lookback "7 days" --thresholds-override "complexity:70"`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		// Validation is done in ExecuteHotspotCheck
		if err := core.ExecuteHotspotCheck(rootCtx, cfg, cacheManager); err != nil {
			contract.LogFatal("Policy check failed", err)
		}
	},
}
