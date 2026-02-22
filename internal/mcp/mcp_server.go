// Package mcp provides the Model Context Protocol (MCP) server implementation.
package mcp

import (
	"context"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewMCPServer initializes and configures the Hotspot MCP server without starting it.
// This is exposed for unit testing.
func NewMCPServer(baseCfg *contract.Config, mgr contract.CacheManager) *server.MCPServer {
	s := server.NewMCPServer(
		"Hotspot Analysis Server",
		"1.0.0",
		server.WithLogging(),
	)

	h := &toolHandler{
		baseCfg: baseCfg,
		mgr:     mgr,
	}

	// --- 1. Tool: get_files_hotspots ---
	s.AddTool(mcp.NewTool("get_files_hotspots",
		mcp.WithDescription("Analyze git history to find code hotspots at the file level."),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository (defaults to current directory if not specified).")),
		mcp.WithString("mode", mcp.Description("Scoring mode (hot, risk, complexity, stale). Defaults to 'hot'."), mcp.Enum("hot", "risk", "complexity", "stale")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results returned.")),
	), h.handleGetFilesHotspots)

	// --- 2. Tool: get_folders_hotspots ---
	s.AddTool(mcp.NewTool("get_folders_hotspots",
		mcp.WithDescription("Analyze git history to find code hotspots aggregated by folder."),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository.")),
		mcp.WithString("mode", mcp.Description("Scoring mode (hot, risk, complexity, stale)."), mcp.Enum("hot", "risk", "complexity", "stale")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results.")),
	), h.handleGetFoldersHotspots)

	// --- 3. Tool: compare_hotspots ---
	s.AddTool(mcp.NewTool("compare_hotspots",
		mcp.WithDescription("Compare hotspots between two Git references (e.g., branches, tags, or commits)."),
		mcp.WithString("base_ref", mcp.Description("The base reference for comparison."), mcp.Required()),
		mcp.WithString("target_ref", mcp.Description("The target reference for comparison."), mcp.Required()),
		mcp.WithString("lookback", mcp.Description("Time window for analysis (e.g., '6 months', '30d').")),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository.")),
		mcp.WithString("mode", mcp.Description("Scoring mode.")),
	), h.handleCompareHotspots)

	// --- 4. Tool: get_timeseries ---
	s.AddTool(mcp.NewTool("get_timeseries",
		mcp.WithDescription("Perform timeseries analysis on a specific file or folder path."),
		mcp.WithString("path", mcp.Description("The file or folder path to analyze."), mcp.Required()),
		mcp.WithString("interval", mcp.Description("Timeseries interval (e.g., '1 month', '3 months', '1 year')."), mcp.Required()),
		mcp.WithNumber("points", mcp.Description("Number of data points to generate (trends)."), mcp.Required()),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository.")),
		mcp.WithString("mode", mcp.Description("Scoring mode.")),
	), h.handleGetTimeseries)

	return s
}

// StartMCPServer starts the Hotspot MCP server.
func StartMCPServer(_ context.Context, baseCfg *contract.Config, mgr contract.CacheManager) error {
	s := NewMCPServer(baseCfg, mgr)
	return server.ServeStdio(s)
}
