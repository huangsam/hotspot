package core

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// runSingleAnalysisCore performs the common Aggregation, Filtering, and Analysis steps.
func runSingleAnalysisCore(cfg *internal.Config, client *internal.LocalGitClient) (*schema.SingleAnalysisOutput, error) {
	// --- 1. Aggregation Phase ---
	output, err := aggregateActivity(cfg, client)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
		return nil, err
	}

	// --- 2. File List Building and Filtering ---
	files := buildFilteredFileList(cfg, output)
	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return nil, fmt.Errorf("no files found")
	}

	// --- 3. Core Analysis ---
	logAnalysisHeader(cfg)
	fileResults := analyzeRepo(cfg, client, output, files)

	return &schema.SingleAnalysisOutput{
		FileResults:     fileResults,
		AggregateOutput: output,
	}, nil
}

// runCompareAnalysisCore runs the file analysis for a specific Git reference in compare mode.
// This extracts the logic repeated between Base and Target in both compare functions.
func runCompareAnalysisForRef(cfg *internal.Config, client *internal.LocalGitClient, ref string) (*schema.CompareAnalysisOutput, error) {
	// 1. Resolve the time window for the reference
	baseStartTime, baseEndTime, err := getAnalysisWindowForRef(client, cfg.RepoPath, ref, cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve time window for Ref '%s': %w", ref, err)
	}

	// 2. Create the isolated config for the run
	cfgRef := cfg.CloneWithTimeWindow(baseStartTime, baseEndTime)

	// 3. Run file analysis
	fileResults, err := analyzeAllFiles(cfgRef, client)
	if err != nil {
		return nil, fmt.Errorf("analysis failed for ref %s", ref)
	}

	// 4. Aggregate folder metrics
	folderReesults := aggregateAndScoreFolders(cfgRef, fileResults)

	return &schema.CompareAnalysisOutput{
		FileResults:   fileResults,
		FolderResults: folderReesults,
	}, nil
}

// analyzeAllFiles performs a full file-level hotspot analysis. This function is designed
// to be used for comparison logic, as it does not rank files preemptively.
func analyzeAllFiles(cfg *internal.Config, client internal.GitClient) ([]schema.FileResult, error) {
	// --- 1. Aggregation Phase ---
	output, err := aggregateActivity(cfg, client)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
		return nil, err
	}

	// --- 2. File List Building and Filtering ---
	files := buildFilteredFileList(cfg, output)
	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return []schema.FileResult{}, nil // Return empty, not an error
	}

	// --- 3. Core Analysis and Initial Ranking ---
	logAnalysisHeader(cfg)
	results := analyzeRepo(cfg, client, output, files)

	// --- 4. Return Data ---
	return results, nil
}

// logAnalysisHeader prints a concise, 2-line header for each analysis phase.
func logAnalysisHeader(cfg *internal.Config) {
	repoName := filepath.Base(cfg.RepoPath)
	if repoName == "" || repoName == "." {
		repoName = "current"
	}

	// Line 1: The analysis summary (Repo and Mode)
	fmt.Printf("🔎 Repo: %s (Mode: %s)\n", repoName, cfg.Mode)

	// Line 2: The actual date range being analyzed
	fmt.Printf("📅 Range: %s → %s\n", cfg.StartTime.Format(internal.DateTimeFormat), cfg.EndTime.Format(internal.DateTimeFormat))
}

// runFollowPass re-analyzes the top N ranked files using 'git --follow'
// to account for renames, and then returns a new, re-ranked list.
func runFollowPass(cfg *internal.Config, client internal.GitClient, ranked []schema.FileResult, output *schema.AggregateOutput) []schema.FileResult {
	// Determine the number of files to re-analyze
	n := min(cfg.ResultLimit, len(ranked))
	if n == 0 {
		return ranked // Nothing to do
	}

	fmt.Printf("🔁 Running --follow re-analysis for top %d files...\n", n)

	var wg sync.WaitGroup
	for i := range n {
		idx := i // Capture loop variable for goroutine
		wg.Go(func() {
			// Note: This modifies the 'ranked' slice concurrently,
			// but each goroutine writes to a *unique* index (ranked[idx]), which is safe.
			rankedFile := ranked[idx]
			rean := analyzeFileCommon(cfg, client, rankedFile.Path, output, true)
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
func analyzeRepo(cfg *internal.Config, client internal.GitClient, output *schema.AggregateOutput, files []string) []schema.FileResult {
	// Initialize channels based on the final number of files to be processed.
	fileCh := make(chan string, len(files))
	fileResultCh := make(chan schema.FileResult, len(files))
	var wg sync.WaitGroup

	// Start worker pool
	for range cfg.Workers {
		// Add one to wait group for each worker
		wg.Go(func() {
			for f := range fileCh {
				// Analysis with useFollow=false for initial run
				result := analyzeFileCommon(cfg, client, f, output, false)
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
// If useFollow is true, git --follow is used to track file renames.
func analyzeFileCommon(cfg *internal.Config, client internal.GitClient, path string, output *schema.AggregateOutput, useFollow bool) schema.FileResult {
	// 1. Initialize the builder
	builder := NewFileMetricsBuilder(cfg, client, path, output, useFollow)

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
func getAnalysisWindowForRef(client internal.GitClient, repoPath, ref string, lookback time.Duration) (startTime time.Time, endTime time.Time, err error) {
	// 1. Find the exact timestamp of the reference (which will be the EndTime)
	// The GitClient implementation now handles running the command and parsing the output.
	endTime, err = client.GetCommitTime(repoPath, ref)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get analysis window for ref '%s': %w", ref, err)
	}

	// 2. Calculate the StartTime (Look-Back Window)
	// StartTime is the commit time minus the look-back duration.
	startTime = endTime.Add(-lookback)

	return startTime, endTime, nil
}
