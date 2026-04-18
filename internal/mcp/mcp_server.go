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
func NewMCPServer(baseCfg *config.Config, mgr iocache.CacheManager, client git.Client, agentsDoc string) *server.MCPServer {
	s := server.NewMCPServer(
		"Hotspot Analysis Server",
		"1.0.0",
		server.WithLogging(),
	)

	h := &toolHandler{
		baseCfg:   baseCfg,
		mgr:       mgr,
		client:    client,
		agentsDoc: agentsDoc,
	}

	// Shared hints for analytical tools
	readOnly := true
	idempotent := true

	// Common parameter descriptions
	urnDesc := "Universal Resource Name (e.g., 'git:github.com/org/repo' or 'local:hash'). If provided, repo_path is optional and utilizes cached/historical analysis results."
	repoPathDesc := "Path to the Git repository (defaults to current directory if not specified)."
	modeDesc := "Scoring mode (hot, risk, complexity, roi). ROI mode identifies refactoring priority. Defaults to 'hot'."
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
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "roi"), mcp.DefaultString("hot")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results returned."), mcp.DefaultNumber(10)),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude (e.g. '**/vendor/, **/*.pb.go').")),
		mcp.WithString("filter", mcp.Description("Path prefix to filter analysis to a specific directory (e.g. 'src/main/').")),
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
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "roi"), mcp.DefaultString("hot")),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results."), mcp.DefaultNumber(10)),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude (e.g. '**/vendor/, **/*.pb.go').")),
		mcp.WithString("filter", mcp.Description("Path prefix to filter analysis to a specific directory (e.g. 'src/main/').")),
	), h.handleGetFoldersHotspots)

	// --- 3. Tool: compare_file_hotspots ---
	s.AddTool(mcp.NewTool("compare_file_hotspots",
		mcp.WithDescription("Compare hotspots between two Git references (e.g., branches, tags, or commits)."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Compare File Hotspots",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("base_ref", mcp.Description("The base reference for comparison."), mcp.Required()),
		mcp.WithString("target_ref", mcp.Description("The target reference for comparison."), mcp.Required()),
		mcp.WithString("preset", mcp.Description("Apply a named configuration preset (small, large, infra). It is recommended to run 'get_repo_shape' first to identify the correct preset for this repository."), mcp.Enum("small", "large", "infra")),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("lookback", mcp.Description("Time window for analysis (e.g., '6 months', '30d').")),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "roi"), mcp.DefaultString("hot")),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude (e.g. '**/vendor/, **/*.pb.go').")),
		mcp.WithString("filter", mcp.Description("Path prefix to filter analysis to a specific directory (e.g. 'src/main/').")),
	), h.handleCompareFileHotspots)

	// --- 4. Tool: compare_folder_hotspots ---
	s.AddTool(mcp.NewTool("compare_folder_hotspots",
		mcp.WithDescription("Compare hotspots between two Git references (e.g., branches, tags, or commits) aggregated by folder."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Compare Folder Hotspots",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("base_ref", mcp.Description("The base reference for comparison."), mcp.Required()),
		mcp.WithString("target_ref", mcp.Description("The target reference for comparison."), mcp.Required()),
		mcp.WithString("preset", mcp.Description("Apply a named configuration preset (small, large, infra). It is recommended to run 'get_repo_shape' first to identify the correct preset for this repository."), mcp.Enum("small", "large", "infra")),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("lookback", mcp.Description("Time window for analysis (e.g., '6 months', '30d').")),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "roi"), mcp.DefaultString("hot")),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude (e.g. '**/vendor/, **/*.pb.go').")),
		mcp.WithString("filter", mcp.Description("Path prefix to filter analysis to a specific directory (e.g. 'src/main/').")),
	), h.handleCompareFolderHotspots)

	// --- 5. Tool: get_timeseries ---
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
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "roi"), mcp.DefaultString("hot")),
		mcp.WithString("start", mcp.Description("Start date for the entire timeseries window (anchors the first point).")),
		mcp.WithString("end", mcp.Description("End date for the entire timeseries window.")),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude (e.g. '**/vendor/, **/*.pb.go').")),
		mcp.WithString("filter", mcp.Description("Path prefix to filter analysis to a specific directory (e.g. 'src/main/').")),
	), h.handleGetTimeseries)

	// --- 5. Tool: get_release_journey ---
	s.AddTool(mcp.NewTool("get_release_journey",
		mcp.WithDescription("Automatically discovers the most recent release tags and computes successive hotspot comparisons between each pair, producing a 'State of the Union' for the repository's technical trajectory."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Get Release Journey",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("preset", mcp.Description("Apply a named configuration preset (small, large, infra). It is recommended to run 'get_repo_shape' first to identify the correct preset for this repository."), mcp.Enum("small", "large", "infra")),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("mode", mcp.Description(modeDesc), mcp.Enum("hot", "risk", "complexity", "roi"), mcp.DefaultString("hot")),
		mcp.WithNumber("transitions", mcp.Description("Number of successive tag transitions to analyze (e.g. 3 = last 4 tags). Defaults to 3."), mcp.DefaultNumber(3)),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude (e.g. '**/vendor/, **/*.pb.go').")),
		mcp.WithString("filter", mcp.Description("Path prefix to filter analysis to a specific directory (e.g. 'src/main/').")),
	), h.handleGetReleaseJourney)

	// --- 6. Tool: get_blast_radius ---
	s.AddTool(mcp.NewTool("get_blast_radius",
		mcp.WithDescription("Identifies files that historically change together (co-change coupling). Reveals 'married' files that may lack proper abstraction."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Get Blast Radius",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithNumber("limit", mcp.Description("Limit the number of results."), mcp.DefaultNumber(10)),
		mcp.WithNumber("threshold", mcp.Description("Minimum coupling score (Jaccard Index, 0.0 to 1.0) to include a pair in the results."), mcp.DefaultNumber(0.3)),
		mcp.WithString("start", mcp.Description(startDesc)),
		mcp.WithString("end", mcp.Description(endDesc)),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude (e.g. '**/vendor/, **/*.pb.go').")),
		mcp.WithString("filter", mcp.Description("Path prefix to filter analysis to a specific directory (e.g. 'src/main/').")),
	), h.handleGetBlastRadius)

	// --- 8. Tool: run_check ---
	s.AddTool(mcp.NewTool("run_check",
		mcp.WithDescription("Run a policy check for CI/CD gating. Analyzes files changed between base and target refs against configured thresholds."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:          "Run Policy Check",
			ReadOnlyHint:   &readOnly,
			IdempotentHint: &idempotent,
		}),
		mcp.WithString("base_ref", mcp.Description("The base reference for comparison."), mcp.Required()),
		mcp.WithString("target_ref", mcp.Description("The target reference for comparison."), mcp.Required()),
		mcp.WithString("urn", mcp.Description(urnDesc)),
		mcp.WithString("repo_path", mcp.Description(repoPathDesc)),
		mcp.WithString("lookback", mcp.Description("Time window for analysis.")),
		mcp.WithString("exclude", mcp.Description("Comma-separated list of glob patterns to exclude.")),
	), h.handleRunCheck)

	s.AddResource(mcp.NewResource("hotspot://docs/agents", "Agent Documentation", mcp.WithResourceDescription("High-level architectural context and domain concepts for AI agents."), mcp.WithMIMEType("text/markdown")), h.handleReadResource)
	s.AddResource(mcp.NewResource("hotspot://docs/metrics", "Scoring Metrics Definition", mcp.WithResourceDescription("Markdown definition of scoring modes, factors, and formulas."), mcp.WithMIMEType("text/markdown")), h.handleReadResource)

	// --- Prompts ---
	s.AddPrompt(mcp.NewPrompt("refactor-prioritization", mcp.WithPromptDescription("Guided workflow for prioritizing refactoring targets using ROI mode.")), h.handleGetPrompt)
	s.AddPrompt(mcp.NewPrompt("release-readiness", mcp.WithPromptDescription("Guided workflow for assessing release readiness by comparing HEAD against the last tag and surfacing new risk patterns.")), h.handleGetPrompt)

	return s
}

// StartMCPServer starts the Hotspot MCP server.
func StartMCPServer(_ context.Context, baseCfg *config.Config, mgr iocache.CacheManager, client git.Client, agentsDoc string) error {
	s := NewMCPServer(baseCfg, mgr, client, agentsDoc)
	return server.ServeStdio(s)
}
