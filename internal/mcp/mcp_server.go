// Package mcp provides the Model Context Protocol (MCP) server implementation.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
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

	// --- 1. Tool: get_files_hotspots ---
	s.AddTool(mcp.NewTool("get_files_hotspots",
		mcp.WithDescription("Analyze git history to find code hotspots at the file level."),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository (defaults to current directory if not specified).")),
		mcp.WithString("mode", mcp.Description("Scoring mode (hot, risk, complexity, stale). Defaults to 'hot'."), mcp.Enum("hot", "risk", "complexity", "stale")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results returned.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg := baseCfg.Clone()
		if p := request.GetString("repo_path", ""); p != "" {
			cfg.RepoPath = p
		}
		if m := request.GetString("mode", ""); m != "" {
			cfg.Mode = schema.ScoringMode(m)
		}
		if l := request.GetInt("limit", 0); l > 0 {
			cfg.ResultLimit = l
		}

		ranked, _, err := core.GetHotspotFilesResults(ctx, cfg, mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
		}

		enriched := schema.EnrichFiles(ranked)
		jsonData, _ := json.MarshalIndent(enriched, "", "  ")

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// --- 2. Tool: get_folders_hotspots ---
	s.AddTool(mcp.NewTool("get_folders_hotspots",
		mcp.WithDescription("Analyze git history to find code hotspots aggregated by folder."),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository.")),
		mcp.WithString("mode", mcp.Description("Scoring mode (hot, risk, complexity, stale)."), mcp.Enum("hot", "risk", "complexity", "stale")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg := baseCfg.Clone()
		if p := request.GetString("repo_path", ""); p != "" {
			cfg.RepoPath = p
		}
		if m := request.GetString("mode", ""); m != "" {
			cfg.Mode = schema.ScoringMode(m)
		}
		if l := request.GetInt("limit", 0); l > 0 {
			cfg.ResultLimit = l
		}

		ranked, _, err := core.GetHotspotFoldersResults(ctx, cfg, mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
		}

		enriched := schema.EnrichFolders(ranked)
		jsonData, _ := json.MarshalIndent(enriched, "", "  ")

		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// --- 3. Tool: compare_hotspots ---
	s.AddTool(mcp.NewTool("compare_hotspots",
		mcp.WithDescription("Compare hotspots between two Git references (e.g., branches, tags, or commits)."),
		mcp.WithString("base_ref", mcp.Description("The base reference for comparison."), mcp.Required()),
		mcp.WithString("target_ref", mcp.Description("The target reference for comparison."), mcp.Required()),
		mcp.WithString("lookback", mcp.Description("Time window for analysis (e.g., '6 months', '30d').")),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository.")),
		mcp.WithString("mode", mcp.Description("Scoring mode.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg := baseCfg.Clone()
		cfg.BaseRef = request.GetString("base_ref", "")
		cfg.TargetRef = request.GetString("target_ref", "")
		lookbackStr := request.GetString("lookback", "")
		if p := request.GetString("repo_path", ""); p != "" {
			cfg.RepoPath = p
		}
		if m := request.GetString("mode", ""); m != "" {
			cfg.Mode = schema.ScoringMode(m)
		}

		if err := contract.RevalidateCompare(cfg, lookbackStr); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
		}

		comparisonResult, _, err := core.GetHotspotCompareResults(ctx, cfg, mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("comparison failed: %v", err)), nil
		}

		jsonData, _ := json.MarshalIndent(comparisonResult, "", "  ")
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// --- 4. Tool: get_timeseries ---
	s.AddTool(mcp.NewTool("get_timeseries",
		mcp.WithDescription("Perform timeseries analysis on a specific file or folder path."),
		mcp.WithString("path", mcp.Description("The file or folder path to analyze."), mcp.Required()),
		mcp.WithString("interval", mcp.Description("Timeseries interval (e.g., '1 month', '3 months', '1 year')."), mcp.Required()),
		mcp.WithNumber("points", mcp.Description("Number of data points to generate (trends)."), mcp.Required()),
		mcp.WithString("repo_path", mcp.Description("Path to the Git repository.")),
		mcp.WithString("mode", mcp.Description("Scoring mode.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg := baseCfg.Clone()
		cfg.TimeseriesPath = request.GetString("path", "")
		cfg.TimeseriesPoints = request.GetInt("points", 0)
		intervalStr := request.GetString("interval", "")

		if p := request.GetString("repo_path", ""); p != "" {
			cfg.RepoPath = p
		}
		if m := request.GetString("mode", ""); m != "" {
			cfg.Mode = schema.ScoringMode(m)
		}

		// Re-validate specifically for timeseries interval parsing
		if err := contract.RevalidateTimeseries(cfg, intervalStr); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid timeseries parameters: %v", err)), nil
		}

		result, _, err := core.GetHotspotTimeseriesResults(ctx, cfg, mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("timeseries analysis failed: %v", err)), nil
		}

		jsonData, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	return s
}

// StartMCPServer starts the Hotspot MCP server.
func StartMCPServer(_ context.Context, baseCfg *contract.Config, mgr contract.CacheManager) error {
	s := NewMCPServer(baseCfg, mgr)
	return server.ServeStdio(s)
}
