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
	"github.com/spf13/viper"
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
	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef)
	if err != nil {
		return err
	}
	comparisonResult := compareFileResults(baseOutput.FileResults, targetOutput.FileResults, cfg.ResultLimit)
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
	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef)
	if err != nil {
		return err
	}
	comparisonResult := compareFolderMetrics(baseOutput.FolderResults, targetOutput.FolderResults, cfg.ResultLimit)
	duration := time.Since(start)
	return internal.PrintComparisonResults(comparisonResult, cfg, duration)
}

// ExecuteHotspotTimeseries runs multiple analyses over disjoint time windows for a specific path.
func ExecuteHotspotTimeseries(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()

	// Get timeseries-specific parameters from Viper
	path := viper.GetString("path")
	intervalStr := viper.GetString("interval")
	numPoints := viper.GetInt("points")

	if path == "" {
		return errors.New("--path is required")
	}
	if intervalStr == "" {
		return errors.New("--interval is required")
	}
	if numPoints < 2 {
		return errors.New("--points must be at least 2")
	}

	interval, err := internal.ParseLookbackDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	numWindows := numPoints
	windowSize := interval / time.Duration(numWindows)
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

	var timeseriesPoints []schema.TimeseriesPoint

	for i := range numWindows {
		end := now.Add(-time.Duration(i) * windowSize)
		startTime := end.Add(-windowSize)
		cfgWindow := cfg.CloneWithTimeWindow(startTime, end)

		var score float64
		if isFolder {
			output, err := runSingleAnalysisCore(ctx, cfgWindow, client)
			if err != nil {
				// If no data in this window, score is 0
				score = 0
			} else {
				folderResults := aggregateAndScoreFolders(cfgWindow, output.FileResults)
				for _, fr := range folderResults {
					if fr.Path == normalizedPath {
						score = fr.Score
						break
					}
				}
			}
		} else {
			output, err := runSingleAnalysisCore(ctx, cfgWindow, client)
			if err != nil {
				// If no data in this window, score is 0
				score = 0
			} else {
				for _, fr := range output.FileResults {
					if fr.Path == normalizedPath {
						score = fr.Score
						break
					}
				}
			}
		}

		// Generate period label
		windowDays := int(windowSize.Hours() / 24)
		var period string
		if i == 0 {
			period = fmt.Sprintf("Current (%dd)", windowDays)
		} else {
			startAgo := i * windowDays
			endAgo := (i + 1) * windowDays
			period = fmt.Sprintf("%dd to %dd Ago", startAgo, endAgo)
		}

		timeseriesPoints = append(timeseriesPoints, schema.TimeseriesPoint{
			Period: period,
			Score:  score,
			Mode:   cfg.Mode,
			Path:   normalizedPath,
		})
	}

	result := schema.TimeseriesResult{Points: timeseriesPoints}
	duration := time.Since(start)
	return internal.PrintTimeseriesResults(result, cfg, duration)
}

// ExecuteHotspotMetrics displays the formal definitions of all scoring modes.
// This is a static command that does not perform Git analysis.
func ExecuteHotspotMetrics() error {
	return internal.PrintMetricsDefinitions()
}
