// Package mcp provides the Model Context Protocol (MCP) server implementation.
package mcp

import (
	"context"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewMCPServer initializes and configures the Hotspot MCP server without starting it.
// This is exposed for unit testing.
func NewMCPServer(baseCfg *config.Config, mgr iocache.CacheManager, client git.Client, agentsDoc, userGuideDoc string) *server.MCPServer {
	s := server.NewMCPServer(
		"Hotspot Analysis Server",
		"1.0.0",
		server.WithLogging(),
	)

	h := &toolHandler{
		baseCfg:      baseCfg,
		mgr:          mgr,
		client:       client,
		agentsDoc:    agentsDoc,
		userGuideDoc: userGuideDoc,
	}

	// Shared hints for analytical tools
	readOnly := true
	idempotent := true

	// Common parameter descriptions
	urnDesc := "Universal Resource Name (e.g., 'git:github.com/org/repo' or 'local:hash'). If provided, repo_path is optional and utilizes cached/historical analysis results."
	repoPathDesc := "Path to the Git repository (defaults to current directory if not specified)."
	modeDesc := "Scoring mode (hot, risk, complexity, stale, roi). ROI mode identifies refactoring priority. Defaults to 'hot'."
	startDesc := "Start date for the analysis window (ISO8601 e.g. '2024-01-01T00:00:00Z', or relative e.g. '30d ago', '6 months ago')."
	endDesc := "End date for the analysis window (ISO8601 or relative). Defaults to now."

	// --- 0. Tool: get_repo_shape ---
	s.AddTool(mcp.NewTool("get_repo_shape",
		mcp.WithDescription("Recommended first step. Analyzes repository shape via a lightweight aggregation pass and returns a recommended configuration preset (small, large, or infra) to be used with other tools."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Get Repository Shape",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
	), h.handleGetRepoShape)

	// --- 1. Tool: get_files_hotspots ---
	s.AddTool(mcp.NewTool("get_files_hotspots",
		mcp.WithDescription("Analyze git history to find code hotspots at the file level."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Get File Hotspots",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("preset", mcp.Description("Apply a named configuration preset (small, large, infra). It is recommended to run 'get_repo_shape' first to identify the correct preset for this repository."), mcp.Enum("small", "large", "infra")),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "stale", "roi"), mcp.DefaultString("hot")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results returned."), mcp.DefaultNumber(10)),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
	), h.handleGetFilesHotspots)

	// --- 2. Tool: get_folders_hotspots ---
	s.AddTool(mcp.NewTool("get_folders_hotspots",
		mcp.WithDescription("Analyze git history to find code hotspots aggregated by folder."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Get Folder Hotspots",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("preset", mcp.Description("Apply a named configuration preset (small, large, infra). It is recommended to run 'get_repo_shape' first to identify the correct preset for this repository."), mcp.Enum("small", "large", "infra")),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "stale", "roi"), mcp.DefaultString("hot")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results."), mcp.DefaultNumber(10)),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
	), h.handleGetFoldersHotspots)

	// --- 3. Tool: compare_hotspots ---
	s.AddTool(mcp.NewTool("compare_hotspots",
		mcp.WithDescription("Compare hotspots between two Git references (e.g., branches, tags, or commits)."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Compare Hotspots",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("base_ref", mcp.Description("The base reference for comparison."), mcp.Required()),
		mcp.WithString("target_ref", mcp.Description("The target reference for comparison."), mcp.Required()),
		mcp.WithString("preset", mcp.Description("Apply a named configuration preset (small, large, infra). It is recommended to run 'get_repo_shape' first to identify the correct preset for this repository."), mcp.Enum("small", "large", "infra")),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("lookback", mcp.Description("Time window for analysis (e.g., '6 months', '30d').")),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "stale", "roi"), mcp.DefaultString("hot")),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
	), h.handleCompareHotspots)

	// --- 4. Tool: get_timeseries ---
	s.AddTool(mcp.NewTool("get_timeseries",
		mcp.WithDescription("Perform timeseries analysis on a specific file or folder path."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Get Path Timeseries",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("path", mcp.Description("The file or folder path to analyze."), mcp.Required()),
		mcp.WithString("interval", mcp.Description("Timeseries interval (e.g., '1 month', '3 months', '1 year')."), mcp.Required()),
		mcp.WithNumber("points", mcp.Description("Number of data points to generate (trends)."), mcp.Required()),
		mcp.WithString("preset", mcp.Description("Apply a named configuration preset (small, large, infra). It is recommended to run 'get_repo_shape' first to identify the correct preset for this repository."), mcp.Enum("small", "large", "infra")),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "stale", "roi"), mcp.DefaultString("hot")),
		mcp.WithString("start", mcp.Description("Start date for the entire timeseries window (anchors the first point).")),
		mcp.WithString("end", mcp.Description("End date for the entire timeseries window.")),
	), h.handleGetTimeseries)

	// --- Resources ---
	s.AddResource(mcp.NewResource("hotspot://config", "Local Configuration", mcp.WithResourceDescription("The content of .hotspot.yml if available."), mcp.WithMIMEType("application/x-yaml")), h.handleReadResource)
	s.AddResource(mcp.NewResource("hotspot://docs/agents", "Agent Documentation", mcp.WithResourceDescription("High-level architectural context and domain concepts for AI agents."), mcp.WithMIMEType("text/markdown")), h.handleReadResource)
	s.AddResource(mcp.NewResource("hotspot://docs/user-guide", "User Guide", mcp.WithResourceDescription("Detailed user guide for Hotspot CLI."), mcp.WithMIMEType("text/markdown")), h.handleReadResource)

	// --- Prompts ---
	s.AddPrompt(mcp.NewPrompt("repository-audit", mcp.WithPromptDescription("Guided workflow for performing a comprehensive hotspots audit.")), h.handleGetPrompt)
	s.AddPrompt(mcp.NewPrompt("refactor-prioritization", mcp.WithPromptDescription("Guided workflow for prioritizing refactoring targets using ROI mode.")), h.handleGetPrompt)

	return s
}

// StartMCPServer starts the Hotspot MCP server.
func StartMCPServer(_ context.Context, baseCfg *config.Config, mgr iocache.CacheManager, client git.Client, agentsDoc, userGuideDoc string) error {
	s := NewMCPServer(baseCfg, mgr, client, agentsDoc, userGuideDoc)
	return server.ServeStdio(s)
}
