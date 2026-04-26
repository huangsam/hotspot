package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
// setupConfig clones the base configuration and populates it with parameters from the MCP request.
// It handles repository resolution, presets, dynamic filters, scoring modes, and time ranges.
func (h *toolHandler) setupConfig(ctx context.Context, request mcp.CallToolRequest) (*config.Config, *mcp.CallToolResult) {
	cfg := h.baseCfg.Clone()
	urn := request.GetString("urn", "")
	repoPath := request.GetString("repo_path", "")

	// 1. Resolve repository and identity
	if urn != "" && repoPath == "" {
		repoPath = "."
	}
	if repoPath == "" && cfg.Git.RepoPath == "" {
		repoPath = "."
	}
	if repoPath != "" {
		if err := config.ResolveGitPathAndFilter(ctx, cfg, h.client, &config.RawInput{RepoPathStr: repoPath}); err != nil {
			return nil, mcp.NewToolResultError(fmt.Sprintf("invalid repository: %v", err))
		}
	}

	// 2. Apply preset if provided
	if p := request.GetString("preset", ""); p != "" {
		_ = config.ApplyPreset(cfg, schema.PresetName(p))
	}

	// 3. Apply dynamic path filters and excludes
	if filter := request.GetString("filter", ""); filter != "" {
		cfg.Git.PathFilter = filter
	}
	if exclude := request.GetString("exclude", ""); exclude != "" {
		var custom []string
		for p := range strings.SplitSeq(exclude, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				custom = append(custom, trimmed)
			}
		}
		cfg.Git.Excludes = custom
	}

	// 4. Apply common scoring and output params
	if m := request.GetString("mode", ""); m != "" {
		cfg.Scoring.Mode = schema.ScoringMode(m)
	}
	if l := request.GetInt("limit", 0); l > 0 {
		cfg.Output.ResultLimit = l
	}

	// 5. Apply comparison params if provided
	if base := request.GetString("base_ref", ""); base != "" {
		cfg.Compare.BaseRef = base
		cfg.Compare.Enabled = true
	}
	if target := request.GetString("target_ref", ""); target != "" {
		cfg.Compare.TargetRef = target
		cfg.Compare.Enabled = true
	}

	// 6. Apply timeseries params if provided
	if path := request.GetString("path", ""); path != "" {
		cfg.Timeseries.Path = path
	}
	if points := request.GetInt("points", 0); points > 0 {
		cfg.Timeseries.Points = points
	}

	// 7. Validate and apply time range
	if err := config.RevalidateTimeRange(cfg, request.GetString("start", ""), request.GetString("end", "")); err != nil {
		return nil, mcp.NewToolResultError(fmt.Sprintf("invalid time range: %v", err))
	}

	return cfg, nil
}

// jsonResponse marshals the data into a standardized indented JSON tool result.
func (h *toolHandler) jsonResponse(data any) (*mcp.CallToolResult, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

// handleGetRepoShape handles the get_repo_shape tool.
func (h *toolHandler) handleGetRepoShape(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	shape, duration, err := core.GetHotspotShapeResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("shape analysis failed: %v", err)), nil
	}

	return h.jsonResponse(schema.RepoShapeOutput{
		Results:  shape,
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleGetFilesHotspots handles the get_files_hotspots tool.
func (h *toolHandler) handleGetFilesHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	ranked, duration, err := core.GetHotspotFilesResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	return h.jsonResponse(schema.FileResultsOutput{
		Results:  schema.EnrichFiles(ranked),
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleGetHeatmap handles the get_heatmap tool.
func (h *toolHandler) handleGetHeatmap(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	analysisType := request.GetString("type", "files")
	var buf strings.Builder
	p := provider.NewHeatmapProvider()

	if analysisType == "folders" {
		ranked, duration, err := core.GetHotspotFoldersResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
		}
		if err := p.WriteFolders(&buf, ranked, cfg.Output, cfg.Runtime, duration); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("heatmap generation failed: %v", err)), nil
		}
	} else {
		ranked, duration, err := core.GetHotspotFilesResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
		}
		if err := p.WriteFiles(&buf, ranked, cfg.Output, cfg.Runtime, duration); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("heatmap generation failed: %v", err)), nil
		}
	}

	return mcp.NewToolResultText(buf.String()), nil
}

// handleGetFoldersHotspots handles the get_folders_hotspots tool.
func (h *toolHandler) handleGetFoldersHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	ranked, duration, err := core.GetHotspotFoldersResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	return h.jsonResponse(schema.FolderResultsOutput{
		Results:  schema.EnrichFolders(ranked),
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleCompareFileHotspots handles the compare_file_hotspots tool.
func (h *toolHandler) handleCompareFileHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	if err := config.RevalidateCompare(cfg, request.GetString("lookback", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
	}

	result, duration, err := core.GetHotspotCompareResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("comparison failed: %v", err)), nil
	}

	return h.jsonResponse(schema.ComparisonResultsOutput{
		Results:  result,
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleCompareFolderHotspots handles the compare_folder_hotspots tool.
func (h *toolHandler) handleCompareFolderHotspots(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	if err := config.RevalidateCompare(cfg, request.GetString("lookback", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
	}

	result, duration, err := core.GetHotspotCompareFoldersResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("comparison failed: %v", err)), nil
	}

	return h.jsonResponse(schema.ComparisonResultsOutput{
		Results:  result,
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleGetTimeseries handles the get_timeseries tool.
func (h *toolHandler) handleGetTimeseries(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	if err := config.RevalidateTimeseries(cfg, request.GetString("interval", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid timeseries parameters: %v", err)), nil
	}

	result, duration, err := core.GetHotspotTimeseriesResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("timeseries analysis failed: %v", err)), nil
	}

	return h.jsonResponse(schema.TimeseriesResultsOutput{
		Results:  result,
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleGetReleaseJourney handles the get_release_journey tool.
func (h *toolHandler) handleGetReleaseJourney(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	transitions := request.GetInt("transitions", 3)
	if transitions < 1 {
		transitions = 3
	}

	start := time.Now()
	result, err := core.GetHotspotJourneyResults(ctx, cfg, h.client, h.mgr, transitions)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("journey analysis failed: %v", err)), nil
	}
	duration := time.Since(start)

	return h.jsonResponse(schema.JourneyResultsOutput{
		Results:  result,
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleGetBlastRadius handles the get_blast_radius tool.
func (h *toolHandler) handleGetBlastRadius(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	threshold := request.GetFloat("threshold", 0.3)

	start := time.Now()
	result, err := core.GetHotspotBlastRadiusResults(ctx, cfg, h.client, cfg.Output.ResultLimit, threshold)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("blast radius analysis failed: %v", err)), nil
	}
	duration := time.Since(start)

	return h.jsonResponse(schema.BlastRadiusResultsOutput{
		Results:  result,
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleRunCheck handles the run_check tool.
func (h *toolHandler) handleRunCheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg, errRes := h.setupConfig(ctx, request)
	if errRes != nil {
		return errRes, nil
	}

	if err := config.RevalidateCompare(cfg, request.GetString("lookback", "")); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid comparison parameters: %v", err)), nil
	}

	result, duration, err := core.GetHotspotCheckResults(core.WithSuppressHeader(ctx), cfg, h.client, h.mgr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("check failed: %v", err)), nil
	}

	return h.jsonResponse(schema.CheckResultsOutput{
		Results:  result,
		Metadata: schema.BuildMetadata(cfg.Runtime, duration),
	})
}

// handleRunBatchAnalysis handles the run_batch_analysis tool.
func (h *toolHandler) handleRunBatchAnalysis(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	searchDir := request.GetString("path", ".")
	autoDiscovery := request.GetBool("auto", false)

	var repos []string
	if autoDiscovery {
		var err error
		repos, err = git.DiscoverRepositories(searchDir)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("repository discovery failed: %v", err)), nil
		}
	} else {
		repos = []string{searchDir}
	}

	if len(repos) == 0 {
		return h.jsonResponse(schema.BatchAnalysisResultsOutput{
			Results:  []schema.RepoShape{},
			Metadata: schema.BuildMetadata(h.baseCfg.Runtime, 0),
		})
	}

	start := time.Now()
	var shapes []schema.RepoShape

	for _, repoPath := range repos {
		// Clone base config for each repo
		repoCfg := h.baseCfg.Clone()
		repoCfg.Git.RepoPath = repoPath
		repoCfg.Output.Format = schema.NoneOut

		// Resolve git details
		if err := config.ResolveGitPathAndFilter(ctx, repoCfg, h.client, &config.RawInput{RepoPathStr: repoPath}); err != nil {
			logger.Error("Failed to resolve repo in batch", "path", repoPath, "error", err)
			continue
		}

		// Run analysis
		analysisCtx := core.WithSuppressHeader(ctx)
		shape, _, err := core.GetBatchAnalysisResults(analysisCtx, repoCfg, h.client, h.mgr)
		if err != nil {
			logger.Error("Analysis failed in batch", "path", repoPath, "error", err)
			continue
		}
		shapes = append(shapes, shape)
	}

	duration := time.Since(start)
	return h.jsonResponse(schema.BatchAnalysisResultsOutput{
		Results:  shapes,
		Metadata: schema.BuildMetadata(h.baseCfg.Runtime, duration),
	})
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
