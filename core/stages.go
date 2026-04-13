package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/schema"
)

// --- Pipeline Stages ---

// preparationStage sets up analysis tracking and logging headers.
type preparationStage struct{}

func (s *preparationStage) Execute(ac *AnalysisContext) error {
	if !shouldSuppressHeader(ac.Context) {
		internal.LogAnalysisHeader(ac.Git, ac.Scoring, ac.Runtime, ac.Output)
	}

	ac.Context = contextWithCacheManager(ac.Context, ac.Mgr)
	ac.AnalysisStore = ac.Mgr.GetAnalysisStore()

	// Resolve Repository URN
	repoPath := ac.Git.GetRepoPath()
	var urn string
	if url, err := ac.Client.GetRemoteURL(ac.Context, repoPath); err == nil && url != "" {
		urn = "git:" + url
	} else {
		// Fallback to local path if no remote origin
		absPath, _ := ac.Client.GetRepoRoot(ac.Context, repoPath)
		if absPath == "" {
			absPath = repoPath
		}
		urn = "local:" + absPath
	}
	ac.RepoURN = urn

	if ac.AnalysisStore != nil {
		configParams := map[string]any{
			"mode":         string(ac.Scoring.GetMode()),
			"lookback":     ac.Compare.GetLookback().String(),
			"repo_path":    repoPath,
			"workers":      ac.Runtime.GetWorkers(),
			"result_limit": ac.Output.GetResultLimit(),
		}
		id, err := ac.AnalysisStore.BeginAnalysis(urn, time.Now(), configParams)
		if err != nil {
			logger.Warn("Analysis tracking initialization failed", err)
		} else if id > 0 {
			ac.AnalysisID = id
			ac.Context = withAnalysisID(ac.Context, id)
		}
	}
	return nil
}

// fileDiscoveryStage discovers files at the specified TargetRef.
type fileDiscoveryStage struct{}

func (s *fileDiscoveryStage) Execute(ac *AnalysisContext) error {
	ref := ac.TargetRef
	if ref == "" {
		ref = "HEAD"
	}
	files, err := ac.Client.ListFilesAtRef(ac.Context, ac.Git.GetRepoPath(), ref)
	if err != nil {
		return fmt.Errorf("failed to list files at ref %s: %w", ref, err)
	}
	ac.Files = files
	return nil
}

// aggregationStage executes the CachedAggregateActivity logic.
type aggregationStage struct{}

func (s *aggregationStage) Execute(ac *AnalysisContext) error {
	var err error
	ac.AggregateOutput, err = agg.CachedAggregateActivity(ac.Context, ac.Git, ac.Compare, ac.Client, ac.Mgr, ac.RepoURN)
	return err
}

// filteringStage combines discovered files with aggregated activity based on rules.
type filteringStage struct{}

// filterFiles applies basic path/exclude rules.
func filterFiles(gitSettings config.GitSettings, allFiles []string) []string {
	var filtered []string
	pathFilterSet := gitSettings.GetPathFilter() != ""
	for _, f := range allFiles {
		if pathFilterSet && !strings.HasPrefix(f, gitSettings.GetPathFilter()) {
			continue
		}
		if schema.ShouldIgnore(f, gitSettings.GetExcludes()) {
			continue
		}
		filtered = append(filtered, f)
	}
	return filtered
}

func (s *filteringStage) Execute(ac *AnalysisContext) error {
	if ac.TargetRef == "" || ac.TargetRef == "HEAD" {
		// For standard HEAD analysis, prioritize the map-based builder
		// which skips files with no activity.
		if ac.AggregateOutput != nil {
			ac.Files = agg.BuildFilteredFileList(ac.Git, ac.AggregateOutput)
		} else {
			ac.Files = filterFiles(ac.Git, ac.Files)
		}
	} else {
		// For compare modes with a specific ref, we only analyze files
		// actually present in that ref's tree that pass filters.
		ac.Files = filterFiles(ac.Git, ac.Files)
	}
	return nil
}

// scoringStage executes concurrent file scoring.
type scoringStage struct{}

func (s *scoringStage) Execute(ac *AnalysisContext) error {
	if len(ac.Files) == 0 {
		ac.FileResults = []schema.FileResult{}
		return nil
	}
	ac.FileResults = analyzeRepo(ac.Context, ac.Git, ac.Scoring, ac.Runtime, ac.Client, ac.AggregateOutput, ac.Files)
	return nil
}

// folderAggregationStage aggregates file results into folder results.
type folderAggregationStage struct{}

func (s *folderAggregationStage) Execute(ac *AnalysisContext) error {
	ac.FolderResults = agg.AggregateAndScoreFolders(ac.Git, ac.Scoring, ac.FileResults)
	return nil
}

// finalizationStage closes out analysis tracking.
type finalizationStage struct{}

func (s *finalizationStage) Execute(ac *AnalysisContext) error {
	if ac.AnalysisStore != nil && ac.AnalysisID > 0 {
		if err := ac.AnalysisStore.EndAnalysis(ac.AnalysisID, time.Now(), len(ac.FileResults)); err != nil {
			logger.Warn("Failed to finalize analysis tracking", err)
		}
	}
	return nil
}
