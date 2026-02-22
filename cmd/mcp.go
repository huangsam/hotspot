package cmd

import (
	"github.com/huangsam/hotspot/internal/mcp"
	"github.com/spf13/cobra"
)

// mcpCmd represents the mcp command.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Hotspot MCP server",
	Long:  `Launch an MCP server that allows AI agents to perform hotspot analysis via standard tools.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Suppress the normal header logs when running in MCP mode
		// to avoid polluting stdio which is used for the protocol.
		return sharedSetup(rootCtx, cmd, args)
	},
	RunE: func(_ *cobra.Command, _ []string) error {
		return mcp.StartMCPServer(rootCtx, cfg, cacheManager)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
