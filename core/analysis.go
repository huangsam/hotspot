package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/core/algo"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/schema"
)

// --- Orchestration helpers ---

// pipelineConfig captures operational options for the analysis pipeline.
type pipelineConfig struct {
	withFolderAggregation bool
	withTrackedAnalysis   bool
	discovery             Stage
}

// executePipeline constructs and runs a pipeline based on the provided configuration.
func executePipeline(ac *AnalysisContext, pCfg pipelineConfig) error {
	var stages []Stage
	if pCfg.withTrackedAnalysis {
		stages = append(stages, &preparationStage{})
	}
	if pCfg.discovery != nil {
		stages = append(stages, pCfg.discovery)
	}
	stages = append(stages, &aggregationStage{}, &filteringStage{}, &scoringStage{})
	if pCfg.withFolderAggregation {
		stages = append(stages, &folderAggregationStage{})
	}

	pipeline := NewPipeline(stages...)
	if pCfg.withTrackedAnalysis {
		pipeline = pipeline.WithDefer(&finalizationStage{})
	}

	return pipeline.Execute(ac)
}

// newAnalysisContext constructs an AnalysisContext for a standard HEAD analysis.
func newAnalysisContext(ctx context.Context, gitSettings config.GitSettings, scoringSettings config.ScoringSettings, runtimeSettings config.RuntimeSettings, outputSettings config.OutputSettings, compareSettings config.ComparisonSettings, client git.Client, mgr iocache.CacheManager) *AnalysisContext {
	return &AnalysisContext{
		Context: ctx, Git: gitSettings, Scoring: scoringSettings,
		Runtime: runtimeSettings, Output: outputSettings,
		Compare: compareSettings, Client: client, Mgr: mgr,
		TargetRef: "HEAD",
		RepoURN:   gitSettings.GetRepoURN(),
	}
}

// noFilesFoundError returns a descriptive error if no files remain after pipeline execution.
func noFilesFoundError(ac *AnalysisContext) error {
	if len(ac.Files) > 0 {
		return nil
	}
	var suggestion string
	switch {
	case ac.Git.GetPathFilter() != "":
		suggestion = fmt.Sprintf(" (try removing --filter '%s' or using --exclude differently)", ac.Git.GetPathFilter())
	case len(ac.Git.GetExcludes()) > 0:
		suggestion = fmt.Sprintf(" (try adjusting excludes: %v)", ac.Git.GetExcludes())
	default:
		suggestion = " (ensure your repository has tracked files in the analysis time range)"
	}
	return fmt.Errorf("no files found for analysis at %s%s", ac.Git.GetRepoPath(), suggestion)
}

// --- Orchestration entry points ---

// runSingleAnalysisCore performs the common Aggregation, Filtering, and Analysis steps.
func runSingleAnalysisCore(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	runtimeSettings config.RuntimeSettings,
	outputSettings config.OutputSettings,
	compareSettings config.ComparisonSettings,
	client git.Client,
	mgr iocache.CacheManager,
	currentFiles []string,
) (*schema.SingleAnalysisOutput, error) {
	ac := newAnalysisContext(ctx, gitSettings, scoringSettings, runtimeSettings, outputSettings, compareSettings, client, mgr)
	ac.Files = currentFiles

	pCfg := pipelineConfig{withTrackedAnalysis: true}
	if len(currentFiles) == 0 {
		pCfg.discovery = &fileDiscoveryStage{}
	}

	if err := executePipeline(ac, pCfg); err != nil {
		return nil, err
	}

	if err := noFilesFoundError(ac); err != nil {
		return nil, err
	}

	return &schema.SingleAnalysisOutput{
		FileResults:     ac.FileResults,
		AggregateOutput: ac.AggregateOutput,
	}, nil
}

// runFolderAnalysisCore performs analysis and aggregates results into folder metrics.
func runFolderAnalysisCore(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	runtimeSettings config.RuntimeSettings,
	outputSettings config.OutputSettings,
	compareSettings config.ComparisonSettings,
	client git.Client,
	mgr iocache.CacheManager,
	currentFiles []string,
) (*schema.SingleAnalysisOutput, error) {
	ac := newAnalysisContext(ctx, gitSettings, scoringSettings, runtimeSettings, outputSettings, compareSettings, client, mgr)
	ac.Files = currentFiles

	pCfg := pipelineConfig{
		withTrackedAnalysis:   true,
		withFolderAggregation: true,
	}
	if len(currentFiles) == 0 {
		pCfg.discovery = &fileDiscoveryStage{}
	}

	if err := executePipeline(ac, pCfg); err != nil {
		return nil, err
	}

	if err := noFilesFoundError(ac); err != nil {
		return nil, err
	}

	return &schema.SingleAnalysisOutput{
		FileResults:     ac.FileResults,
		FolderResults:   ac.FolderResults,
		AggregateOutput: ac.AggregateOutput,
	}, nil
}

// runCompareAnalysisForRef runs the file analysis for a specific Git reference in compare mode.
func runCompareAnalysisForRef(ctx context.Context, cfg *config.Config, client git.Client, ref string, mgr iocache.CacheManager) (*schema.CompareAnalysisOutput, error) {
	baseStartTime, baseEndTime, err := getAnalysisWindowForRef(ctx, client, cfg.Git.RepoPath, ref, cfg.Compare.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve time window for Ref '%s': %w", ref, err)
	}

	cfgRef := cfg.CloneWithTimeWindow(baseStartTime, baseEndTime)

	fileResults, err := analyzeAllFilesAtRef(ctx, cfgRef.Git, cfgRef.Scoring, cfgRef.Runtime, client, ref, mgr)
	if err != nil {
		return nil, fmt.Errorf("analysis failed for ref %s: %w", ref, err)
	}

	folderResults := agg.AggregateAndScoreFolders(cfgRef.Git, cfgRef.Scoring, fileResults)

	return &schema.CompareAnalysisOutput{
		FileResults:   fileResults,
		FolderResults: folderResults,
	}, nil
}

// analyzeAllFilesAtRef performs file analysis for all files that exist at a specific Git reference.
func analyzeAllFilesAtRef(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	runtimeSettings config.RuntimeSettings,
	client git.Client,
	ref string,
	mgr iocache.CacheManager,
) ([]schema.FileResult, error) {
	ac := &AnalysisContext{
		Context: ctx, Git: gitSettings, Scoring: scoringSettings,
		Runtime: runtimeSettings, Client: client, Mgr: mgr,
		TargetRef: ref,
		// Compare must be a concrete zero-value (not nil interface) so downstream
		// cache key generation can safely call GetLookback(). Compare features are
		// intentionally disabled here since analyzeAllFilesAtRef uses a fixed time
		// window derived from the ref's commit time.
		// Preparation and finalization are intentionally omitted: compare-mode sub-analyses
		// are internal helpers and should not be tracked independently.
		Compare: config.CompareConfig{},
	}

	pCfg := pipelineConfig{discovery: &fileDiscoveryStage{}}
	if err := executePipeline(ac, pCfg); err != nil {
		return nil, err
	}

	return ac.FileResults, nil
}

// runFollowPass re-analyzes the top N ranked files using 'git --follow'
// to account for renames, and then returns a new, re-ranked list.
func runFollowPass(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	outputSettings config.OutputSettings,
	client git.Client,
	ranked []schema.FileResult,
	output *schema.AggregateOutput,
) []schema.FileResult {
	// Determine the number of files to re-analyze
	n := min(outputSettings.GetResultLimit(), len(ranked))
	if n == 0 {
		return ranked // Nothing to do
	}

	if !shouldSuppressHeader(ctx) {
		logger.Info(fmt.Sprintf("Running --follow re-analysis for top %d files...", n))
	}

	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			// Note: This modifies the 'ranked' slice concurrently,
			// but each goroutine writes to a *unique* index (ranked[i]), which is safe.
			rankedFile := ranked[i]
			followCtx := withUseFollow(ctx, true)
			rean := analyzeFileCommon(followCtx, gitSettings, scoringSettings, client, rankedFile.Path, output)
			ranked[i] = rean
		})
	}
	wg.Wait()

	// re-rank after follow pass
	return algo.RankFiles(ranked, outputSettings.GetResultLimit())
}

// analyzeRepo processes all files in parallel using a worker pool.
// It spawns runtimeSettings.GetWorkers() number of goroutines to analyze files concurrently
// and aggregates their results into a single slice of schema.FileMetrics.
func analyzeRepo(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	runtimeSettings config.RuntimeSettings,
	client git.Client,
	output *schema.AggregateOutput,
	files []string,
) []schema.FileResult {
	// Initialize channels based on the final number of files to be processed.
	fileCh := make(chan string, len(files))
	fileResultCh := make(chan schema.FileResult, len(files))
	var wg sync.WaitGroup

	// Start worker pool
	for range runtimeSettings.GetWorkers() {
		// Add one to wait group for each worker
		wg.Go(func() {
			for f := range fileCh {
				// Analysis with useFollow=false for initial run (default context behavior)
				result := analyzeFileCommon(ctx, gitSettings, scoringSettings, client, f, output)
				fileResultCh <- result
			}
		})
	}

	// Send files to worker channel
	for _, f := range files {
		fileCh <- f
	}
	close(fileCh)

	// Wait for all workers to finish processing
	wg.Wait()
	close(fileResultCh)

	// Aggregate results directly into the slice (removing the intermediate map)
	results := make([]schema.FileResult, 0, len(files))
	for r := range fileResultCh {
		results = append(results, r)
	}

	// 5. Record metrics and scores to database (if analysis tracking is enabled)
	if analysisID, ok := getAnalysisID(ctx); ok && analysisID > 0 {
		BatchRecordFileAnalysis(ctx, scoringSettings, analysisID, results)
	}

	return results
}

// analyzeFileCommon computes all metrics for a single file in the repository.
// It gathers Git history data (commits, authors, dates), file size, and calculates
// derived metrics like churn and the Gini coefficient of author contributions.
// The analysis is constrained by the time range in gitSettings if specified.
// Git follow behavior is controlled by the context.
func analyzeFileCommon(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	client git.Client,
	path string,
	output *schema.AggregateOutput,
) schema.FileResult {
	// 1. Initialize the builder
	builder := NewFileMetricsBuilder(ctx, gitSettings, scoringSettings, client, path, output)

	// 2. Execute the required steps in order (Method Chaining)
	builder.
		FetchAllGitMetrics().      // Gathers all Git data
		FetchFileStats().          // Gets file stats
		FetchRecentInfo().         // Adds recent metrics if it exists
		CalculateDerivedMetrics(). // Calculates AgeDays and Gini
		CalculateOwner().          // Calculates file owner
		CalculateScore()           // Computes the final composite score

	// 3. Build the final result
	return builder.Build()
}

// getAnalysisWindowForRef queries Git for the exact commit time of the given reference
// and sets the StartTime by looking back a fixed duration from that commit time.
func getAnalysisWindowForRef(ctx context.Context, client git.Client, repoPath, ref string, lookback time.Duration) (startTime time.Time, endTime time.Time, err error) {
	// 1. Find the exact timestamp of the reference (which will be the EndTime)
	// The GitClient implementation now handles running the command and parsing the output.
	endTime, err = client.GetCommitTime(ctx, repoPath, ref)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get analysis window for ref '%s': %w", ref, err)
	}

	// 2. Calculate the StartTime (Look-Back Window)
	// StartTime is the commit time minus the look-back duration.
	startTime = endTime.Add(-lookback)

	return startTime, endTime, nil
}
