package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// Timeseries analysis constraints
const (
	minCommits        = 30
	minLookback       = 3 * 30 * 24 * time.Hour // T_min: 3 months (temporal coverage constraint)
	maxSearchDuration = 6 * 30 * 24 * time.Hour // T_Max: 6 months (performance constraint for Git search)
)

// runSingleAnalysisCore performs the common Aggregation, Filtering, and Analysis steps.
func runSingleAnalysisCore(ctx context.Context, cfg *contract.Config, client contract.GitClient, mgr contract.CacheManager) (*schema.SingleAnalysisOutput, error) {
	if !shouldSuppressHeader(ctx) {
		internal.LogAnalysisHeader(cfg)
	}

	// --- 1. Aggregation Phase (with caching) ---
	output, err := agg.CachedAggregateActivity(ctx, cfg, client, mgr)
	if err != nil {
		return nil, err
	}

	// --- 2. File List Building and Filtering ---
	files := agg.BuildFilteredFileList(cfg, output)
	if len(files) == 0 {
		return nil, errors.New("no files found")
	}

	// --- 3. Core Analysis ---
	fileResults := analyzeRepo(ctx, cfg, client, output, files)

	return &schema.SingleAnalysisOutput{
		FileResults:     fileResults,
		AggregateOutput: output,
	}, nil
}

// runCompareAnalysisForRef runs the file analysis for a specific Git reference in compare mode.
// Headers are always suppressed in compare mode.
func runCompareAnalysisForRef(ctx context.Context, cfg *contract.Config, client contract.GitClient, ref string, mgr contract.CacheManager) (*schema.CompareAnalysisOutput, error) {
	// 1. Resolve the time window for the reference
	baseStartTime, baseEndTime, err := getAnalysisWindowForRef(ctx, client, cfg.RepoPath, ref, cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve time window for Ref '%s': %w", ref, err)
	}

	// 2. Create the isolated config for the run
	cfgRef := cfg.CloneWithTimeWindow(baseStartTime, baseEndTime)

	// 3. Run file analysis
	fileResults, err := analyzeAllFilesAtRef(ctx, cfgRef, client, ref, mgr)
	if err != nil {
		return nil, fmt.Errorf("analysis failed for ref %s", ref)
	}

	// 4. Aggregate folder metrics
	folderResults := agg.AggregateAndScoreFolders(cfgRef, fileResults)

	return &schema.CompareAnalysisOutput{
		FileResults:   fileResults,
		FolderResults: folderResults,
	}, nil
}

// analyzeAllFilesAtRef performs file analysis for all files that exist at a specific Git reference.
// Headers are always suppressed in compare mode.
func analyzeAllFilesAtRef(ctx context.Context, cfg *contract.Config, client contract.GitClient, ref string, mgr contract.CacheManager) ([]schema.FileResult, error) {
	// --- 1. Get all files at the reference ---
	files, err := client.ListFilesAtRef(ctx, cfg.RepoPath, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to list files at ref %s: %w", ref, err)
	}

	// Apply path filter and excludes
	filteredFiles := make([]string, 0, len(files))
	pathFilterSet := cfg.PathFilter != ""
	for _, f := range files {
		// Apply path filter check only if the filter is set
		if pathFilterSet && !strings.HasPrefix(f, cfg.PathFilter) {
			continue
		}

		// Apply excludes filter
		if contract.ShouldIgnore(f, cfg.Excludes) {
			continue
		}

		filteredFiles = append(filteredFiles, f)
	}

	if len(filteredFiles) == 0 {
		return []schema.FileResult{}, nil // Return empty, not an error
	}

	// --- 2. Aggregation Phase (with caching) ---
	output, err := agg.CachedAggregateActivity(ctx, cfg, client, mgr)
	if err != nil {
		return nil, err
	}

	// --- 3. Core Analysis ---
	results := analyzeRepo(ctx, cfg, client, output, filteredFiles)

	// --- 4. Return Data ---
	return results, nil
}

// runFollowPass re-analyzes the top N ranked files using 'git --follow'
// to account for renames, and then returns a new, re-ranked list.
func runFollowPass(ctx context.Context, cfg *contract.Config, client contract.GitClient, ranked []schema.FileResult, output *schema.AggregateOutput) []schema.FileResult {
	// Determine the number of files to re-analyze
	n := min(cfg.ResultLimit, len(ranked))
	if n == 0 {
		return ranked // Nothing to do
	}

	if !shouldSuppressHeader(ctx) {
		fmt.Printf("🔁 Running --follow re-analysis for top %d files...\n", n)
	}

	var wg sync.WaitGroup
	for i := range n {
		idx := i // Capture loop variable for goroutine
		wg.Go(func() {
			// Note: This modifies the 'ranked' slice concurrently,
			// but each goroutine writes to a *unique* index (ranked[idx]), which is safe.
			rankedFile := ranked[idx]
			followCtx := withUseFollow(ctx, true)
			rean := analyzeFileCommon(followCtx, cfg, client, rankedFile.Path, output)
			ranked[idx] = rean
		})
	}
	wg.Wait()

	// re-rank after follow pass
	return rankFiles(ranked, cfg.ResultLimit)
}

// analyzeRepo processes all files in parallel using a worker pool.
// It spawns cfg.Workers number of goroutines to analyze files concurrently
// and aggregates their results into a single slice of schema.FileMetrics.
func analyzeRepo(ctx context.Context, cfg *contract.Config, client contract.GitClient, output *schema.AggregateOutput, files []string) []schema.FileResult {
	// Initialize channels based on the final number of files to be processed.
	fileCh := make(chan string, len(files))
	fileResultCh := make(chan schema.FileResult, len(files))
	var wg sync.WaitGroup

	// Start worker pool
	for range cfg.Workers {
		// Add one to wait group for each worker
		wg.Go(func() {
			for f := range fileCh {
				// Analysis with useFollow=false for initial run (default context behavior)
				result := analyzeFileCommon(ctx, cfg, client, f, output)
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
// The analysis is constrained by the time range in cfg if specified.
// Git follow behavior is controlled by the context.
func analyzeFileCommon(ctx context.Context, cfg *contract.Config, client contract.GitClient, path string, output *schema.AggregateOutput) schema.FileResult {
	// 1. Initialize the builder
	builder := NewFileMetricsBuilder(ctx, cfg, client, path, output)

	// 2. Execute the required steps in order (Method Chaining)
	builder.
		FetchAllGitMetrics().      // Gathers all Git data
		FetchFileStats().          // Gets file stats
		FetchRecentInfo().         // Adds recent metrics if it exists
		CalculateDerivedMetrics(). // Calculates AgeDays and Gini
		CalculateOwner().          // Calculates file owner
		CalculateScore()           // Computes the final composite score

	// 3. Return the final product
	return builder.Build()
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
	cfg *contract.Config,
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
			cfg.RepoPath,
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

		cfgWindow := cfg.CloneWithTimeWindow(startTime, currentEnd)

		// --- Execute Analysis Core ---
		score, owners := analyzeTimeseriesPoint(ctx, cfgWindow, client, normalizedPath, isFolder, mgr)
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
			Mode:     cfg.Mode,
			Lookback: lookbackDuration,
		})
	}

	return timeseriesPoints
}

// analyzeTimeseriesPoint performs the analysis for a single timeseries point.
func analyzeTimeseriesPoint(
	ctx context.Context,
	cfg *contract.Config,
	client contract.GitClient,
	path string,
	isFolder bool,
	mgr contract.CacheManager,
) (float64, []string) {
	suppressCtx := withSuppressHeader(ctx)
	output, err := runSingleAnalysisCore(suppressCtx, cfg, client, mgr)
	if err != nil {
		// If no data in this window (e.g. no commits), score is 0
		return 0, []string{}
	}

	// Extract score and owners from analysis output
	if isFolder {
		folderResults := agg.AggregateAndScoreFolders(cfg, output.FileResults)
		for _, fr := range folderResults {
			if fr.Path == path {
				return fr.Score, fr.Owners
			}
		}
		return 0, []string{}
	}
	for _, fr := range output.FileResults {
		if fr.Path == path {
			return fr.Score, fr.Owners
		}
	}
	return 0, []string{}
}
