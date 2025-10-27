// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"sync"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// AnalyzeRepo processes all files in parallel using a worker pool.
// It spawns cfg.Workers number of goroutines to analyze files concurrently
// and aggregates their results into a single slice of schema.FileMetrics.
func AnalyzeRepo(cfg *internal.Config, files []string) []schema.FileMetrics {
	// Filter files according to excludes. This is required for consistency
	// since ListRepoFiles only applies the path filter, not excludes.
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if internal.ShouldIgnore(f, cfg.Excludes) {
			continue
		}
		// NOTE: Path filtering is mostly redundant here, as it's done in main/ListRepoFiles,
		// but excluding files is necessary.
		filtered = append(filtered, f)
	}

	// Initialize channels based on the final number of files to be processed.
	fileCh := make(chan string, len(filtered))
	resultCh := make(chan schema.FileMetrics, len(filtered)) // Use len(filtered) instead of len(files)
	var wg sync.WaitGroup

	// Start worker pool
	for range cfg.Workers {
		// Add one to wait group for each worker
		wg.Go(func() {
			for f := range fileCh {
				// Analysis with useFollow=false for initial run
				metrics := AnalyzeFileCommon(cfg, f, false)
				resultCh <- metrics
			}
		})
	}

	// Send files to worker channel
	for _, f := range filtered {
		fileCh <- f
	}
	close(fileCh)

	// Wait for all workers to finish processing
	wg.Wait()
	close(resultCh)

	// Aggregate results directly into the slice (removing the intermediate map)
	results := make([]schema.FileMetrics, 0, len(filtered))
	for r := range resultCh {
		results = append(results, r)
	}

	return results
}

// AnalyzeFileCommon computes all metrics for a single file in the repository.
// It gathers Git history data (commits, authors, dates), file size, and calculates
// derived metrics like churn and the Gini coefficient of author contributions.
// The analysis is constrained by the time range in cfg if specified.
// If useFollow is true, git --follow is used to track file renames.
func AnalyzeFileCommon(cfg *internal.Config, path string, useFollow bool) schema.FileMetrics {
	// 1. Initialize the builder
	builder := NewFileMetricsBuilder(cfg, path, useFollow)

	// 2. Execute the required steps in order (Method Chaining)
	builder.
		fetchCommitHistory().      // Gathers initial Git data
		fetchFileSize().           // Gets size
		fetchLinesOfCode().        // Gets lines of code
		calculateChurn().          // Computes lines added/deleted
		applyGlobalMaps().         // Adds recent metrics if global data exists
		calculateDerivedMetrics(). // Calculates AgeDays and Gini
		calculateScore()           // Computes the final composite score

	// 3. Return the final product
	return builder.Build()
}
