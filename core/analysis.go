package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/core/algo"
	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// Timeseries analysis constraints.
const (
	minCommits        = 30
	minLookback       = 3 * 30 * 24 * time.Hour // T_min: 3 months (temporal coverage constraint)
	maxSearchDuration = 6 * 30 * 24 * time.Hour // T_Max: 6 months (performance constraint for Git search)
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
	if ac.AnalysisStore != nil {
		configParams := map[string]any{
			"mode":         string(ac.Scoring.GetMode()),
			"lookback":     ac.Compare.GetLookback().String(),
			"repo_path":    ac.Git.GetRepoPath(),
			"workers":      ac.Runtime.GetWorkers(),
			"result_limit": ac.Output.GetResultLimit(),
		}
		id, err := ac.AnalysisStore.BeginAnalysis(time.Now(), configParams)
		if err != nil {
			contract.LogWarn("Analysis tracking initialization failed", err)
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
	ac.AggregateOutput, err = agg.CachedAggregateActivity(ac.Context, ac.Git, ac.Compare, ac.Client, ac.Mgr)
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
		if contract.ShouldIgnore(f, gitSettings.GetExcludes()) {
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
			contract.LogWarn("Failed to finalize analysis tracking", err)
		}
	}
	return nil
}

// --- Orchestration helpers ---

// newAnalysisContext constructs an AnalysisContext for a standard HEAD analysis.
func newAnalysisContext(ctx context.Context, gitSettings config.GitSettings, scoringSettings config.ScoringSettings, runtimeSettings config.RuntimeSettings, outputSettings config.OutputSettings, compareSettings config.ComparisonSettings, client contract.GitClient, mgr contract.CacheManager) *AnalysisContext {
	return &AnalysisContext{
		Context: ctx, Git: gitSettings, Scoring: scoringSettings,
		Runtime: runtimeSettings, Output: outputSettings,
		Compare: compareSettings, Client: client, Mgr: mgr,
		TargetRef: "HEAD",
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
	client contract.GitClient,
	mgr contract.CacheManager,
) (*schema.SingleAnalysisOutput, error) {
	ac := newAnalysisContext(ctx, gitSettings, scoringSettings, runtimeSettings, outputSettings, compareSettings, client, mgr)

	pipeline := NewPipeline(
		&preparationStage{},
		&aggregationStage{},
		&filteringStage{},
		&scoringStage{},
	).WithDefer(&finalizationStage{})

	if err := pipeline.Execute(ac); err != nil {
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
	client contract.GitClient,
	mgr contract.CacheManager,
) (*schema.SingleAnalysisOutput, error) {
	ac := newAnalysisContext(ctx, gitSettings, scoringSettings, runtimeSettings, outputSettings, compareSettings, client, mgr)

	pipeline := NewPipeline(
		&preparationStage{},
		&aggregationStage{},
		&filteringStage{},
		&scoringStage{},
		&folderAggregationStage{},
	).WithDefer(&finalizationStage{})

	if err := pipeline.Execute(ac); err != nil {
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
func runCompareAnalysisForRef(ctx context.Context, cfg *config.Config, client contract.GitClient, ref string, mgr contract.CacheManager) (*schema.CompareAnalysisOutput, error) {
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
	client contract.GitClient,
	ref string,
	mgr contract.CacheManager,
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

	// This pipeline discovers files at TargetRef, aggregates activity,
	// filters the discovered files, and scores them.
	pipeline := NewPipeline(
		&fileDiscoveryStage{},
		&aggregationStage{},
		&filteringStage{},
		&scoringStage{},
	)

	if err := pipeline.Execute(ac); err != nil {
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
	client contract.GitClient,
	ranked []schema.FileResult,
	output *schema.AggregateOutput,
) []schema.FileResult {
	// Determine the number of files to re-analyze
	n := min(outputSettings.GetResultLimit(), len(ranked))
	if n == 0 {
		return ranked // Nothing to do
	}

	if !shouldSuppressHeader(ctx) {
		fmt.Printf("Running --follow re-analysis for top %d files...\n", n)
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
	client contract.GitClient,
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
	client contract.GitClient,
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
	result := builder.Build()

	// 4. Record metrics and scores to database (if analysis tracking is enabled)
	if analysisID, ok := getAnalysisID(ctx); ok && analysisID > 0 {
		// Get the analysis store from the context via the cache manager
		recordFileAnalysis(ctx, scoringSettings, analysisID, path, &result)
	}

	return result
}

// recordFileAnalysis records file metrics and scores to the database.
func recordFileAnalysis(ctx context.Context, scoringSettings config.ScoringSettings, analysisID int64, path string, result *schema.FileResult) {
	// Get the cache manager from context
	mgr := cacheManagerFromContext(ctx)
	if mgr == nil {
		return
	}

	analysisStore := mgr.GetAnalysisStore()
	if analysisStore == nil {
		return
	}

	now := time.Now()

	// Record raw git metrics
	metrics := schema.FileMetrics{
		AnalysisTime:     now,
		TotalCommits:     result.Commits,
		TotalChurn:       result.Churn,
		ContributorCount: result.UniqueContributors,
		AgeDays:          float64(result.AgeDays), // Convert int to float64 for type compatibility with FileMetrics struct
		GiniCoefficient:  result.Gini,
		FileOwner:        getOwnerString(result.Owners),
	}

	// Compute all four scoring modes
	allScores := result.AllScores

	// Record final scores
	scores := schema.FileScores{
		AnalysisTime:    now,
		HotScore:        allScores[schema.HotMode],
		RiskScore:       allScores[schema.RiskMode],
		ComplexityScore: allScores[schema.ComplexityMode],
		StaleScore:      allScores[schema.StaleMode],
		ScoreLabel:      string(scoringSettings.GetMode()),
	}

	// Record both metrics and scores in one operation
	if err := analysisStore.RecordFileMetricsAndScores(analysisID, path, metrics, scores); err != nil {
		logTrackingError("RecordFileMetricsAndScores", path, err)
	}
}

// getOwnerString converts the owners slice to a string.
func getOwnerString(owners []string) string {
	if len(owners) == 0 {
		return ""
	}
	return owners[0] // Return the primary owner
}

// logTrackingError logs database tracking errors to stderr without disrupting analysis.
func logTrackingError(operation, path string, err error) {
	contract.LogWarn(fmt.Sprintf("Analysis tracking failed for %s on %s", operation, path), err)
}

// getAnalysisWindowForRef queries Git for the exact commit time of the given reference
// and sets the StartTime by looking back a fixed duration from that commit time.
func getAnalysisWindowForRef(ctx context.Context, client contract.GitClient, repoPath, ref string, lookback time.Duration) (startTime time.Time, endTime time.Time, err error) {
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

// runTimeseriesAnalysis performs the core timeseries analysis logic.
func runTimeseriesAnalysis(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	client contract.GitClient,
	normalizedPath string,
	isFolder bool,
	now time.Time,
	interval time.Duration,
	numPoints int,
	mgr contract.CacheManager,
) []schema.TimeseriesPoint {
	var timeseriesPoints []schema.TimeseriesPoint
	currentEnd := now

	// Process each time window in reverse chronological order
	for i := range numPoints {
		// 1. Establish the End Time for this point/window (fixed step-back)
		if i > 0 {
			currentEnd = currentEnd.Add(-interval)
		}

		// 2. Calculate T_M_min: time to get minCommits, limited by maxSearchDuration
		// The Git search will be confined to [currentEnd - maxSearchDuration, currentEnd]
		commitTime, err := client.GetOldestCommitDateForPath(
			ctx,
			gitSettings.GetRepoPath(),
			normalizedPath,
			currentEnd,
			minCommits,
			maxSearchDuration, // Limiting the Git traversal depth
		)

		var lookbackFromCommits time.Duration
		if err != nil || commitTime.IsZero() {
			// If the search fails or finds fewer than minCommits within T_Max,
			// assume the path is sparse and use the larger of T_min or T_Max.
			lookbackFromCommits = max(minLookback, maxSearchDuration)
		} else {
			lookbackFromCommits = currentEnd.Sub(commitTime)
		}

		// 3. T_L is the max of the two constraints (T_min and T_M_min)
		lookbackDuration := max(minLookback, lookbackFromCommits)
		startTime := currentEnd.Add(-lookbackDuration)

		// Create dynamic time window settings
		gitWin := config.GitConfig{
			RepoPath:  gitSettings.GetRepoPath(),
			StartTime: startTime,
			EndTime:   currentEnd,
		}

		// --- Execute Analysis Core ---
		score, owners := analyzeTimeseriesPoint(ctx, gitWin, scoringSettings, client, normalizedPath, isFolder, mgr)
		// --- End Execute Analysis Core ---

		// 4. Generate period label
		var period string
		intervalDays := int(interval.Hours() / 24)
		if i == 0 {
			period = fmt.Sprintf("0-%dd ago", intervalDays)
		} else {
			startDays := intervalDays * i
			endDays := startDays + intervalDays
			period = fmt.Sprintf("%d-%dd ago", startDays, endDays)
		}

		timeseriesPoints = append(timeseriesPoints, schema.TimeseriesPoint{
			Period:   period,
			Start:    startTime,
			End:      currentEnd,
			Score:    score,
			Path:     normalizedPath,
			Owners:   owners,
			Mode:     scoringSettings.GetMode(),
			Lookback: lookbackDuration,
		})
	}

	return timeseriesPoints
}

// analyzeTimeseriesPoint performs the analysis for a single timeseries point.
func analyzeTimeseriesPoint(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	client contract.GitClient,
	path string,
	isFolder bool,
	mgr contract.CacheManager,
) (float64, []string) {
	suppressCtx := WithSuppressHeader(ctx)
	// OutputSettings and RuntimeSettings/ComparisonSettings are needed for runSingleAnalysisCore.
	// We'll create defaults for those that aren't critical for a single point scoring.
	runtime := config.RuntimeConfig{Workers: 1}
	outputCfg := config.OutputConfig{ResultLimit: 10}
	compare := config.CompareConfig{}

	var output *schema.SingleAnalysisOutput
	var err error

	if isFolder {
		output, err = runFolderAnalysisCore(suppressCtx, gitSettings, scoringSettings, runtime, outputCfg, compare, client, mgr)
	} else {
		output, err = runSingleAnalysisCore(suppressCtx, gitSettings, scoringSettings, runtime, outputCfg, compare, client, mgr)
	}

	if err != nil {
		// If no data in this window (e.g. no commits), score is 0
		return 0, []string{}
	}

	// Extract score and owners from analysis output
	if isFolder {
		for _, fr := range output.FolderResults {
			if fr.Path == path {
				return fr.Score, fr.Owners
			}
		}
		return 0, []string{}
	}
	for _, fr := range output.FileResults {
		if fr.Path == path {
			return fr.ModeScore, fr.Owners
		}
	}

	return 0, []string{}
}
