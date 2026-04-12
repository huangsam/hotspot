package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/mark3labs/mcp-go/mcp"
)

// toolHandler holds common dependencies for MCP tool handlers.
type toolHandler struct {
	baseCfg *config.Config
	mgr     contract.CacheManager
	client  contract.GitClient
}

// handleGetFilesHotspots handles the get_files_hotspots tool.
func (h *toolHandler) handleGetFilesHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	repoPath := request.GetString("repo_path", "")
	if repoPath == "" && cfg.Git.RepoPath == "" {
		repoPath = "."
	}
	if repoPath != "" {
		if err := config.ResolveGitPathAndFilter(ctx, cfg, h.client, &config.RawInput{RepoPathStr: repoPath}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid repo path: %v", err)), nil
		}
	}
	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}
	if l := request.GetInt("limit", 0); l > 0 {
		cfg.Output.ResultLimit = l
	}
	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	ranked, _, err := core.GetHotspotFilesResults(core.WithSuppressHeader(ctx), cfg, h.mgr)
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
	repoPath := request.GetString("repo_path", "")
	if repoPath == "" && cfg.Git.RepoPath == "" {
		repoPath = "."
	}
	if repoPath != "" {
		if err := config.ResolveGitPathAndFilter(ctx, cfg, h.client, &config.RawInput{RepoPathStr: repoPath}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid repo path: %v", err)), nil
		}
	}
	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}
	if l := request.GetInt("limit", 0); l > 0 {
		cfg.Output.ResultLimit = l
	}
	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	ranked, _, err := core.GetHotspotFoldersResults(core.WithSuppressHeader(ctx), cfg, h.mgr)
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
	repoPath := request.GetString("repo_path", "")
	if repoPath == "" && cfg.Git.RepoPath == "" {
		repoPath = "."
	}
	if repoPath != "" {
		if err := config.ResolveGitPathAndFilter(ctx, cfg, h.client, &config.RawInput{RepoPathStr: repoPath}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid repo path: %v", err)), nil
		}
	}
	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}

	if err := config.RevalidateCompare(cfg, lookbackStr); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
	}

	comparisonResult, _, err := core.GetHotspotCompareResults(core.WithSuppressHeader(ctx), cfg, h.mgr)
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

	repoPath := request.GetString("repo_path", "")
	if repoPath == "" && cfg.Git.RepoPath == "" {
		repoPath = "."
	}
	if repoPath != "" {
		if err := config.ResolveGitPathAndFilter(ctx, cfg, h.client, &config.RawInput{RepoPathStr: repoPath}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid repo path: %v", err)), nil
		}
	}
	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}

	// Re-validate specifically for timeseries interval parsing
	if err := config.RevalidateTimeseries(cfg, intervalStr); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid timeseries parameters: %v", err)), nil
	}

	result, _, err := core.GetHotspotTimeseriesResults(core.WithSuppressHeader(ctx), cfg, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("timeseries analysis failed: %v", err)), nil
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}
