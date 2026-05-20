package cmd

import (
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/mcp"
	"github.com/spf13/cobra"
)

// mcpCmd represents the mcp command.
var (
	AgentsDoc string
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Hotspot MCP server",
	Long: `Launch an MCP (Model Context Protocol) server that exposes Hotspot analysis
as structured JSON-RPC tools for AI agents and IDE integrations.

When running, AI agents can call tools like get_files_hotspots, get_heatmap,
run_batch_analysis, and more — with full parameter parity to the CLI.

The server communicates over stdio using the MCP protocol. Configure your
AI client (e.g. Claude, Cursor, Windsurf) to invoke 'hotspot mcp' as a
subprocess tool provider.

Examples:
  # Start the MCP server (typically invoked by your AI client)
  hotspot mcp

  # Start with a specific cache backend
  hotspot mcp --cache-backend sqlite`,
	PreRunE: sharedSetup,
	RunE: func(cmd *cobra.Command, _ []string) error {
		client := git.NewLocalGitClient()
		return mcp.StartMCPServer(cmd.Context(), cfg, cacheManager, client, AgentsDoc)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
