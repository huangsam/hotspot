// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// ExecutorFunc defines the function signature for executing different analysis modes.
type ExecutorFunc func(ctx context.Context, cfg *internal.Config) error

// ExecuteHotspotFiles runs the file-level analysis and prints results to stdout.
// It serves as the main entry point for the 'files' mode.
func ExecuteHotspotFiles(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()
	output, err := runSingleAnalysisCore(ctx, cfg, client)
	if err != nil {
		return err
	}
	resultsToRank := output.FileResults
	if cfg.Follow && len(resultsToRank) > 0 {
		rankedForFollow := rankFiles(resultsToRank, cfg.ResultLimit)
		resultsToRank = runFollowPass(ctx, cfg, client, rankedForFollow, output.AggregateOutput)
	}
	ranked := rankFiles(resultsToRank, cfg.ResultLimit)
	duration := time.Since(start)
	return internal.PrintFileResults(ranked, cfg, duration)
}

// ExecuteHotspotFolders runs the folder-level analysis and prints results to stdout.
// It serves as the main entry point for the 'folders' mode.
func ExecuteHotspotFolders(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()
	output, err := runSingleAnalysisCore(ctx, cfg, client)
	if err != nil {
		return err
	}
	folderResults := aggregateAndScoreFolders(cfg, output.FileResults)
	ranked := rankFolders(folderResults, cfg.ResultLimit)
	duration := time.Since(start)
	return internal.PrintFolderResults(ranked, cfg, duration)
}

// ExecuteHotspotCompare runs two file-level analyses (Base and Target)
// based on Git references and computes the delta results.
func ExecuteHotspotCompare(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()

	// Print single header for the comparison
	internal.LogCompareHeader(cfg)

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef)
	if err != nil {
		return err
	}
	comparisonResult := compareFileResults(baseOutput.FileResults, targetOutput.FileResults, cfg.ResultLimit, string(cfg.Mode))
	duration := time.Since(start)
	return internal.PrintComparisonResults(comparisonResult, cfg, duration)
}

// ExecuteHotspotCompareFolders runs two folder-level analyses (Base and Target)
// based on Git references and computes the delta results.
// It follows the same pattern as ExecuteHotspotCompare but aggregates to folders
// before performing the comparison.
func ExecuteHotspotCompareFolders(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()

	// Print single header for the comparison
	internal.LogCompareHeader(cfg)

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef)
	if err != nil {
		return err
	}
	comparisonResult := compareFolderMetrics(baseOutput.FolderResults, targetOutput.FolderResults, cfg.ResultLimit, string(cfg.Mode))
	duration := time.Since(start)
	return internal.PrintComparisonResults(comparisonResult, cfg, duration)
}

// ExecuteHotspotTimeseries runs multiple analyses over overlapping, dynamic-lookback time windows.
// This implements Strategy 2: Time-Boxed M_min Approximation. The Git search for M_min commits
// is capped by maxSearchDuration to prevent slow full-history traversal on large repos.
func ExecuteHotspotTimeseries(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()

	// Get timeseries-specific parameters from config
	path := cfg.TimeseriesPath
	interval := cfg.TimeseriesInterval
	numPoints := cfg.TimeseriesPoints

	if path == "" {
		return errors.New("--path is required")
	}
	if interval == 0 {
		return errors.New("--interval is required")
	}
	if numPoints < 1 {
		return errors.New("--points must be at least 1")
	}

	now := time.Now()
	client := internal.NewLocalGitClient()

	// Normalize and validate the path relative to repo root
	normalizedPath, err := internal.NormalizeTimeseriesPath(cfg.RepoPath, path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists and determine if it's a file or folder
	fullPath := filepath.Join(cfg.RepoPath, normalizedPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", normalizedPath)
	}
	isFolder := info.IsDir()

	// Define constraints
	const minCommits = 30
	const minLookback = 3 * 30 * 24 * time.Hour       // T_min: 3 months (temporal coverage constraint)
	const maxSearchDuration = 6 * 30 * 24 * time.Hour // T_Max: 6 months (performance constraint for Git search)

	// Print single header for the entire timeseries analysis
	internal.LogTimeseriesHeader(cfg, interval, numPoints)

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
		var score float64
		var owners []string
		suppressCtx := withSuppressHeader(ctx, true)
		output, err := runSingleAnalysisCore(suppressCtx, cfgWindow, client)

		if err != nil {
			// If no data in this window (e.g. no commits), score is 0
			score = 0
			owners = []string{}
		} else {
			// Extract score and owners from analysis output
			if isFolder {
				folderResults := aggregateAndScoreFolders(cfgWindow, output.FileResults)
				found := false
				for _, fr := range folderResults {
					if fr.Path == normalizedPath {
						score = fr.Score
						owners = fr.Owners
						found = true
						break
					}
				}
				if !found {
					owners = []string{}
				}
			} else {
				found := false
				for _, fr := range output.FileResults {
					if fr.Path == normalizedPath {
						score = fr.Score
						owners = fr.Owners
						found = true
						break
					}
				}
				if !found {
					owners = []string{}
				}
			}
		}
		// --- End Execute Analysis Core ---

		// 4. Generate period label
		var period string
		if i == 0 {
			period = "Current"
		} else {
			period = fmt.Sprintf("%dd Ago", int(interval.Hours()/24)*i)
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

	result := schema.TimeseriesResult{Points: timeseriesPoints}
	duration := time.Since(start)
	return internal.PrintTimeseriesResults(result, cfg, duration)
}

// ExecuteHotspotMetrics displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func ExecuteHotspotMetrics(_ context.Context, cfg *internal.Config) error {
	return internal.PrintMetricsDefinitions(cfg.CustomWeights, cfg)
}
