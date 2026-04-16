package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/mark3labs/mcp-go/mcp"
)

// toolHandler holds common dependencies for MCP tool handlers.
type toolHandler struct {
	baseCfg      *config.Config
	mgr          iocache.CacheManager
	client       git.Client
	agentsDoc    string
	userGuideDoc string
}

// resolveRepositoryPath handles URN-based or repo_path-based resolution.
// If URN is provided, it uses the URN for portable identity (to avoid path fragmentation).
// If repo_path is also provided, it validates and uses that for git operations.
// If neither is provided, it defaults to "." for current directory.
// This ensures the analysis is path-independent when URN is used uniformly across machines.
func (h *toolHandler) resolveRepositoryPath(ctx context.Context, cfg *config.Config, urn string, repoPath string) error {
	// If URN is provided, use it for portable identity
	// The actual repo_path is needed for git operations, so we still need to resolve it
	if urn != "" {
		// Store URN as a hint (though it will be resolved again in the pipeline)
		// For now, we still need repo_path for git operations
		// If repo_path is not provided with URN, we have a few options:
		// 1. Require repo_path (backward compatible)
		// 2. Return error asking for repo_path
		// 3. Use "." as default and let URN be the identity
		// We'll go with option 3 for maximum portability
		if repoPath == "" {
			repoPath = "."
		}
	}

	// Default to current directory if nothing provided
	if repoPath == "" && cfg.Git.RepoPath == "" {
		repoPath = "."
	}

	// If we have a repo path, resolve it
	if repoPath != "" {
		if err := config.ResolveGitPathAndFilter(ctx, cfg, h.client, &config.RawInput{RepoPathStr: repoPath}); err != nil {
			return err
		}
	}

	return nil
}

// applyPresetToConfig applies a named preset to cfg if preset is non-empty.
// Unknown preset names silently fall back to defaults. Explicit request parameters
// applied after this call take precedence.
func applyPresetToConfig(cfg *config.Config, presetName string) {
	if presetName == "" {
		return
	}
	_ = config.ApplyPreset(cfg, schema.PresetName(presetName))
}

// handleGetRepoShape handles the get_repo_shape tool.
func (h *toolHandler) handleGetRepoShape(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	shape, _, err := core.GetHotspotShapeResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("shape analysis failed: %v", err)), nil
	}

	jsonData, err := json.MarshalIndent(shape, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize shape: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleGetFilesHotspots handles the get_files_hotspots tool.
func (h *toolHandler) handleGetFilesHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))

	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}
	if l := request.GetInt("limit", 0); l > 0 {
		cfg.Output.ResultLimit = l
	}
	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	ranked, _, err := core.GetHotspotFilesResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	enriched := schema.EnrichFiles(ranked)
	jsonData, _ := json.MarshalIndent(enriched, "", "  ")

	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleGetFoldersHotspots handles the get_folders_hotspots tool.
func (h *toolHandler) handleGetFoldersHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))

	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}
	if l := request.GetInt("limit", 0); l > 0 {
		cfg.Output.ResultLimit = l
	}
	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	ranked, _, err := core.GetHotspotFoldersResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	enriched := schema.EnrichFolders(ranked)
	jsonData, _ := json.MarshalIndent(enriched, "", "  ")

	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleCompareHotspots handles the compare_hotspots tool.
func (h *toolHandler) handleCompareHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	cfg.Compare.BaseRef = request.GetString("base_ref", "")
	cfg.Compare.TargetRef = request.GetString("target_ref", "")
	lookbackStr := request.GetString("lookback", "")
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))

	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}

	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	if err := config.RevalidateCompare(cfg, lookbackStr); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
	}

	comparisonResult, _, err := core.GetHotspotCompareResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("comparison failed: %v", err)), nil
	}

	jsonData, _ := json.MarshalIndent(comparisonResult, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleGetTimeseries handles the get_timeseries tool.
func (h *toolHandler) handleGetTimeseries(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	cfg.Timeseries.Path = request.GetString("path", "")
	cfg.Timeseries.Points = request.GetInt("points", 0)
	intervalStr := request.GetString("interval", "")
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))

	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}

	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	// Re-validate specifically for timeseries interval parsing
	if err := config.RevalidateTimeseries(cfg, intervalStr); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid timeseries parameters: %v", err)), nil
	}

	result, _, err := core.GetHotspotTimeseriesResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("timeseries analysis failed: %v", err)), nil
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleReadResource handles the reading of registered resources.
func (h *toolHandler) handleReadResource(_ context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	var filePath string
	var mimeType string

	switch request.Params.URI {
	case "hotspot://config":
		filePath = ".hotspot.yml"
		mimeType = "application/x-yaml"
	case "hotspot://docs/agents":
		filePath = "AGENTS.md"
		mimeType = "text/markdown"
	case "hotspot://docs/user-guide":
		filePath = "USERGUIDE.md"
		mimeType = "text/markdown"
	default:
		return nil, fmt.Errorf("unknown resource: %s", request.Params.URI)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("resource file %s not found", filePath)
		}
		return nil, fmt.Errorf("failed to read resource: %v", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: mimeType,
			Text:     string(data),
		},
	}, nil
}

// handleGetPrompt handles the retrieval of analytical playbooks.
func (h *toolHandler) handleGetPrompt(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	var messages []mcp.PromptMessage

	switch request.Params.Name {
	case "repository-audit":
		messages = []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `Please perform a comprehensive audit of this repository using the following steps:
1. Run 'get_repo_shape' to identify the project scale and the recommended preset.
2. Use 'get_files_hotspots' with the recommended preset and mode='hot' to find the most active files.
3. Use 'get_files_hotspots' with mode='risk' to identify files with high ownership concentration (bus factor risk).
4. Summarize the overall health of the project, highlighting any files that appear in both the 'hot' and 'risk' top results.`,
				},
			},
		}
	case "refactor-prioritization":
		messages = []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `Help me prioritize refactoring targets by following this workflow:
1. Run 'get_repo_shape' to ensure we are using the correct preset for this codebase.
2. Use 'get_files_hotspots' with the recommended preset and mode='roi' to identify files with the highest return on refactoring investment.
3. For the top 3 files identified, use 'get_timeseries' to see if their risk profile is improving or worsening over the last 6 months.
4. Provide a prioritized list of refactoring candidates with justifications based on churn, complexity, and historical trends.`,
				},
			},
		}
	default:
		return nil, fmt.Errorf("unknown prompt: %s", request.Params.Name)
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Workflow for %s", request.Params.Name),
		Messages:    messages,
	}, nil
}
