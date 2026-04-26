package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/internal/outwriter/provider"
	"github.com/huangsam/hotspot/schema"
	"github.com/mark3labs/mcp-go/mcp"
)

// toolHandler holds common dependencies for MCP tool handlers.
type toolHandler struct {
	baseCfg   *config.Config
	mgr       iocache.CacheManager
	client    git.Client
	agentsDoc string
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

// applyDynamicFilters applies ad-hoc filter and exclude parameters from the MCP request.
// It overrides any preset or default exclusions/filters if explicitly provided.
func applyDynamicFilters(cfg *config.Config, request mcp.CallToolRequest) {
	if filter := request.GetString("filter", ""); filter != "" {
		cfg.Git.PathFilter = filter
	}
	if exclude := request.GetString("exclude", ""); exclude != "" {
		var custom []string
		parts := strings.SplitSeq(exclude, ",")
		for p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				custom = append(custom, trimmed)
			}
		}
		cfg.Git.Excludes = custom
	}
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
	applyDynamicFilters(cfg, request)

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

// handleGetHeatmap handles the get_heatmap tool.
func (h *toolHandler) handleGetHeatmap(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))
	applyDynamicFilters(cfg, request)

	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}
	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	analysisType := request.GetString("type", "files")
	var buf strings.Builder
	p := provider.NewHeatmapProvider()

	if analysisType == "folders" {
		ranked, _, err := core.GetHotspotFoldersResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
		}
		if err := p.WriteFolders(&buf, ranked, cfg.Output, cfg.Runtime, 0); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("heatmap generation failed: %v", err)), nil
		}
	} else {
		ranked, _, err := core.GetHotspotFilesResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
		}
		if err := p.WriteFiles(&buf, ranked, cfg.Output, cfg.Runtime, 0); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("heatmap generation failed: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(buf.String()), nil
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
	applyDynamicFilters(cfg, request)

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

// handleCompareFileHotspots handles the compare_file_hotspots tool.
func (h *toolHandler) handleCompareFileHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	cfg.Compare.BaseRef = request.GetString("base_ref", "")
	cfg.Compare.TargetRef = request.GetString("target_ref", "")
	cfg.Compare.Enabled = true
	lookbackStr := request.GetString("lookback", "")
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))
	applyDynamicFilters(cfg, request)

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

// handleCompareFolderHotspots handles the compare_folder_hotspots tool.
func (h *toolHandler) handleCompareFolderHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	cfg.Compare.BaseRef = request.GetString("base_ref", "")
	cfg.Compare.TargetRef = request.GetString("target_ref", "")
	cfg.Compare.Enabled = true
	lookbackStr := request.GetString("lookback", "")
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))
	applyDynamicFilters(cfg, request)

	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}

	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	if err := config.RevalidateCompare(cfg, lookbackStr); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
	}

	comparisonResult, _, err := core.GetHotspotCompareFoldersResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
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
	applyDynamicFilters(cfg, request)

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

// handleGetReleaseJourney handles the get_release_journey tool.
func (h *toolHandler) handleGetReleaseJourney(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyPresetToConfig(cfg, request.GetString("preset", ""))
	applyDynamicFilters(cfg, request)

	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}

	transitions := request.GetInt("transitions", 3)
	if transitions < 1 {
		transitions = 3
	}

	result, err := core.GetHotspotJourneyResults(ctx, cfg, h.client, h.mgr, transitions)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("journey analysis failed: %v", err)), nil
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleGetBlastRadius handles the get_blast_radius tool.
func (h *toolHandler) handleGetBlastRadius(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyDynamicFilters(cfg, request)

	limit := request.GetInt("limit", 10)
	threshold := request.GetFloat("threshold", 0.3)

	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err)), nil
	}

	result, err := core.GetHotspotBlastRadiusResults(ctx, cfg, h.client, limit, threshold)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("blast radius analysis failed: %v", err)), nil
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// handleRunCheck handles the run_check tool.
func (h *toolHandler) handleRunCheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := h.baseCfg.Clone()
	cfg.Compare.BaseRef = request.GetString("base_ref", "")
	cfg.Compare.TargetRef = request.GetString("target_ref", "")
	cfg.Compare.Enabled = true
	lookbackStr := request.GetString("lookback", "")
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	if err := h.resolveRepositoryPath(ctx, cfg, urn, repoPath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err)), nil
	}

	applyDynamicFilters(cfg, request)

	if err := config.RevalidateCompare(cfg, lookbackStr); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
	}

	result, _, err := core.GetHotspotCheckResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("check failed: %v", err)), nil
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// withRecovery is a decorator that adds panic recovery to a tool handler.
func withRecovery(handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic in tool handler", "err", r, "tool", request.Params.Name)
				result = mcp.NewToolResultError(fmt.Sprintf("internal server error: %v", r))
				err = nil
			}
		}()
		return handler(ctx, request)
	}
}

// handleReadResource handles the reading of registered resources.
func (h *toolHandler) handleReadResource(_ context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in handleReadResource", "err", r, "uri", request.Params.URI)
		}
	}()
	switch request.Params.URI {
	case "hotspot://docs/agents":
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/markdown",
				Text:     h.agentsDoc,
			},
		}, nil
	case "hotspot://docs/metrics":
		var buf strings.Builder
		p := provider.NewMarkdownProvider()
		if err := p.WriteMetrics(&buf, h.baseCfg.Scoring.CustomWeights, h.baseCfg.Output); err != nil {
			return nil, fmt.Errorf("failed to render metrics documentation: %w", err)
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/markdown",
				Text:     buf.String(),
			},
		}, nil
	}

	return nil, fmt.Errorf("unknown resource: %s", request.Params.URI)
}

// handleGetPrompt handles the retrieval of analytical playbooks.
func (h *toolHandler) handleGetPrompt(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	var messages []mcp.PromptMessage

	switch request.Params.Name {
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
	case "release-readiness":
		messages = []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `Assess whether this repository is ready for a release cut by following this workflow:
1. Run 'get_repo_shape' to identify the preset and recommended scoring mode.
2. Use 'compare_hotspots' with base_ref set to the most recent tag and target_ref='HEAD', mode='hot' to identify files that have spiked in activity since the last release.
3. Re-run 'compare_hotspots' with the same refs but mode='risk' to surface any newly-introduced knowledge silos or ownership concentration.
4. If any files appear in BOTH the hot and risk results, flag them as release blockers — they are simultaneously volatile and fragile.
5. Provide a clear go/no-go recommendation with a short list of specific files and the reason each one is a concern, or confirm the release looks clean.`,
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
