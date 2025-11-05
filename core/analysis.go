package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// AnalyzeFolders performs a full folder-level hotspot analysis and returns
// the ranked folder results.
func AnalyzeFolders(cfg *internal.Config, client internal.GitClient) ([]schema.FolderResults, error) {
	// --- 1. Aggregation Phase ---
	fmt.Printf("🔎 Aggregating activity since %s\n", cfg.StartTime.Format(internal.DateTimeFormat))
	output, err := aggregateActivity(cfg, client)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
		return nil, err
	}

	// --- 2. File List Building and Filtering ---
	files := buildFilteredFileList(cfg, output)
	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return []schema.FolderResults{}, nil // Return empty, not an error
	}

	// Note: Folder-specific logic for building 'seenFolders' map
	// is removed as it wasn't used downstream. If it's needed
	// by 'aggregateAndScoreFolders', it can be rebuilt there
	// by iterating over the 'fileMetrics'.

	// --- 3. Core Analysis and Initial Ranking ---
	logAnalysisHeader(cfg)

	// We analyze the *files* to get metrics,
	fileMetrics := analyzeRepo(cfg, client, output, files)
	// ...then aggregate those metrics into folders.
	folderResults := aggregateAndScoreFolders(cfg, fileMetrics)

	// Rank the folder results
	ranked := rankFolders(folderResults, cfg.ResultLimit)

	// --- 4. Optional --follow Re-analysis and Re-ranking ---
	// (Skipped for folder analysis as in the original)

	// --- 5. Return Data ---
	return ranked, nil
}

// AnalyzeFiles performs a full file-level hotspot analysis and returns the
// ranked results. It encapsulates aggregation, filtering, analysis,
// and the optional --follow pass. This function is designed to be
// reusable for comparison logic, as it does not print to stdout.
func AnalyzeFiles(cfg *internal.Config, client internal.GitClient) ([]schema.FileMetrics, error) {
	// --- 1. Aggregation Phase ---
	fmt.Printf("🔎 Aggregating activity since %s\n", cfg.StartTime.Format(internal.DateTimeFormat))
	output, err := aggregateActivity(cfg, client)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
		return nil, err
	}

	// --- 2. File List Building and Filtering ---
	files := buildFilteredFileList(cfg, output)
	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return []schema.FileMetrics{}, nil // Return empty, not an error
	}

	// --- 3. Core Analysis and Initial Ranking ---
	logAnalysisHeader(cfg)
	results := analyzeRepo(cfg, client, output, files)
	ranked := rankFiles(results, cfg.ResultLimit)

	// --- 4. Optional --follow Re-analysis and Re-ranking ---
	if cfg.Follow && len(ranked) > 0 {
		ranked = runFollowPass(cfg, client, ranked, output)
	}

	// --- 5. Return Data ---
	return ranked, nil
}

// logAnalysisHeader prints the standard analysis startup message.
func logAnalysisHeader(cfg *internal.Config) {
	fmt.Printf("🧠 hotspot: Analyzing %s (Mode: %s)\n", cfg.RepoPath, cfg.Mode)
	fmt.Printf("📅 Range: %s → %s\n", cfg.StartTime.Format(internal.DateTimeFormat), cfg.EndTime.Format(internal.DateTimeFormat))
}

// runFollowPass re-analyzes the top N ranked files using 'git --follow'
// to account for renames, and then returns a new, re-ranked list.
func runFollowPass(cfg *internal.Config, client internal.GitClient, ranked []schema.FileMetrics, output *schema.AggregateOutput) []schema.FileMetrics {
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
func analyzeRepo(cfg *internal.Config, client internal.GitClient, output *schema.AggregateOutput, files []string) []schema.FileMetrics {
	// Initialize channels based on the final number of files to be processed.
	fileCh := make(chan string, len(files))
	resultCh := make(chan schema.FileMetrics, len(files))
	var wg sync.WaitGroup

	// Start worker pool
	for range cfg.Workers {
		// Add one to wait group for each worker
		wg.Go(func() {
			for f := range fileCh {
				// Analysis with useFollow=false for initial run
				metrics := analyzeFileCommon(cfg, client, f, output, false)
				resultCh <- metrics
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
	close(resultCh)

	// Aggregate results directly into the slice (removing the intermediate map)
	results := make([]schema.FileMetrics, 0, len(files))
	for r := range resultCh {
		results = append(results, r)
	}

	return results
}

// analyzeFileCommon computes all metrics for a single file in the repository.
// It gathers Git history data (commits, authors, dates), file size, and calculates
// derived metrics like churn and the Gini coefficient of author contributions.
// The analysis is constrained by the time range in cfg if specified.
// If useFollow is true, git --follow is used to track file renames.
func analyzeFileCommon(cfg *internal.Config, client internal.GitClient, path string, output *schema.AggregateOutput, useFollow bool) schema.FileMetrics {
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
// It requires a GitClient implementation to function.
func getAnalysisWindowForRef(client internal.GitClient, repoPath, ref string, lookback time.Duration) (startTime time.Time, endTime time.Time, err error) {
	// 1. Find the exact timestamp of the reference (which will be the EndTime)
	out, err := client.Run(repoPath, "log", "-1", "--pretty=format:%aI", ref)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get commit time for ref '%s': %w", ref, err)
	}
	dateStr := strings.TrimSpace(string(out))

	// 2. Parse the EndTime
	// We use time.RFC3339 which matches the %aI output (ISO 8601 strict).
	endTime, err = time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse Git timestamp '%s' for ref '%s': %w", dateStr, ref, err)
	}

	// 3. Calculate the StartTime (Look-Back Window)
	// StartTime is the commit time minus the look-back duration.
	startTime = endTime.Add(-lookback)

	return startTime, endTime, nil
}
