package cmd

import (
	"fmt"

	"github.com/huangsam/hotspot/core"
	"github.com/spf13/cobra"
)

// blastRadiusCmd performs blast radius analysis.
var blastRadiusCmd = &cobra.Command{
	Use:   "blast-radius [repo-path]",
	Short: "Identify files that historically change together (co-change coupling)",
	Long: `Analyze Git history to identify files that are logically coupled.
Coupling is measured using the Jaccard Index, which looks at how often
files appear together in the same commit.

This tool helps reveal "hidden" dependencies that are not visible in
the explicit import graph, such as:
- Code and documentation that must always stay in sync.
- Microservices that share a database schema.
- Configuration files married to specific binary logic.

Examples:
  # Find the top coupled file pairs in the repo
  hotspot blast-radius

  # Limit to top 20 pairs
  hotspot blast-radius --limit 20

  # Analyze a specific repository
  hotspot blast-radius /path/to/repo
`,
	Args:    cobra.MaximumNArgs(1),
	PreRunE: sharedSetupWrapper,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := core.ExecuteHotspotBlastRadius(cmd.Context(), cfg, gitClient, resultWriter); err != nil {
			return fmt.Errorf("cannot run blast radius analysis: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(blastRadiusCmd)
}
