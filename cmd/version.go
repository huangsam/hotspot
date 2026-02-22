package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
)

// versionCmd shows the verbose version for diagnostic purposes.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of hotspot.",
	Long: `Display version information including build details.

Shows:
- Release version
- Git commit hash
- Build timestamp
- Go runtime version

Useful for:
- Debugging compatibility issues
- Verifying correct binary installation
- Reporting bugs with version details`,
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Printf("hotspot CLI\n")
		cmd.Printf("  Version: %s\n", version)
		cmd.Printf("  Commit:  %s\n", commit)
		cmd.Printf("  Built:   %s\n", date)
		cmd.Printf("  Runtime: %s\n", runtime.Version())
	},
}
