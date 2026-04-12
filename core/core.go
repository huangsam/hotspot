// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/huangsam/hotspot/core/algo"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/outwriter"
	"github.com/huangsam/hotspot/schema"
)

// ExecutorFunc defines the function signature for executing different analysis modes.
type ExecutorFunc func(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) error

// ExecuteHotspotFiles runs the file-level analysis and prints results to stdout.
// It serves as the main entry point for the 'files' mode.
func ExecuteHotspotFiles(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) error {
	ranked, duration, err := GetHotspotFilesResults(ctx, cfg, mgr)
	if err != nil {
		return err
	}
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteFiles(w, ranked, cfg.Output, cfg.Runtime, duration)
	}, "Wrote files table")
}

// GetHotspotFilesResults runs the file-level analysis and returns the ranked results.
func GetHotspotFilesResults(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) ([]schema.FileResult, time.Duration, error) {
	start := time.Now()
	client := git.NewLocalGitClient()
	output, err := runSingleAnalysisCore(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, cfg.Output, cfg.Compare, client, mgr)
	if err != nil {
		return nil, 0, err
	}
	resultsToRank := output.FileResults
	if cfg.Git.Follow && len(resultsToRank) > 0 {
		rankedForFollow := algo.RankFiles(resultsToRank, cfg.Output.ResultLimit)
		resultsToRank = runFollowPass(ctx, cfg.Git, cfg.Scoring, cfg.Output, client, rankedForFollow, output.AggregateOutput)
	}
	ranked := algo.RankFiles(resultsToRank, cfg.Output.ResultLimit)
	return ranked, time.Since(start), nil
}

// ExecuteHotspotFolders runs the folder-level analysis and prints results to stdout.
// It serves as the main entry point for the 'folders' mode.
func ExecuteHotspotFolders(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) error {
	ranked, duration, err := GetHotspotFoldersResults(ctx, cfg, mgr)
	if err != nil {
		return err
	}
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteFolders(w, ranked, cfg.Output, cfg.Runtime, duration)
	}, "Wrote folders table")
}

// GetHotspotFoldersResults runs the folder-level analysis and returns the ranked results.
func GetHotspotFoldersResults(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) ([]schema.FolderResult, time.Duration, error) {
	start := time.Now()
	client := git.NewLocalGitClient()
	output, err := runFolderAnalysisCore(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, cfg.Output, cfg.Compare, client, mgr)
	if err != nil {
		return nil, 0, err
	}
	ranked := algo.RankFolders(output.FolderResults, cfg.Output.ResultLimit)
	return ranked, time.Since(start), nil
}

// ExecuteHotspotCompare runs two file-level analyses (Base and Target)
// based on Git references and computes the delta results.
func ExecuteHotspotCompare(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) error {
	// Print single header for the comparison only if output is text
	if cfg.Output.Format == schema.TextOut {
		internal.LogCompareHeader(cfg.Git, cfg.Scoring, cfg.Compare)
	}

	comparisonResult, duration, err := GetHotspotCompareResults(ctx, cfg, mgr)
	if err != nil {
		return err
	}
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteComparison(w, comparisonResult, cfg.Output, cfg.Runtime, duration)
	}, "Wrote file comparison table")
}

// GetHotspotCompareResults runs the file-level comparison analysis and returns the results.
func GetHotspotCompareResults(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) (schema.ComparisonResult, time.Duration, error) {
	start := time.Now()
	client := git.NewLocalGitClient()

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.Compare.BaseRef, mgr)
	if err != nil {
		return schema.ComparisonResult{}, 0, err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.Compare.TargetRef, mgr)
	if err != nil {
		return schema.ComparisonResult{}, 0, err
	}
	comparisonResult := compareFileResults(baseOutput.FileResults, targetOutput.FileResults, cfg.Output.ResultLimit, string(cfg.Scoring.Mode))
	return comparisonResult, time.Since(start), nil
}

// ExecuteHotspotCompareFolders runs two folder-level analyses (Base and Target)
// based on Git references and computes the delta results.
// It follows the same pattern as ExecuteHotspotCompare but aggregates to folders
// before performing the comparison.
func ExecuteHotspotCompareFolders(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) error {
	start := time.Now()
	client := git.NewLocalGitClient()

	// Print single header for the comparison only if output is text
	if cfg.Output.Format == schema.TextOut {
		internal.LogCompareHeader(cfg.Git, cfg.Scoring, cfg.Compare)
	}

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.Compare.BaseRef, mgr)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.Compare.TargetRef, mgr)
	if err != nil {
		return err
	}
	comparisonResult := compareFolderMetrics(baseOutput.FolderResults, targetOutput.FolderResults, cfg.Output.ResultLimit, string(cfg.Scoring.Mode))
	duration := time.Since(start)
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteComparison(w, comparisonResult, cfg.Output, cfg.Runtime, duration)
	}, "Wrote folder comparison table")
}

// ExecuteHotspotTimeseries runs multiple analyses over overlapping, dynamic-lookback time windows.
func ExecuteHotspotTimeseries(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) error {
	result, duration, err := GetHotspotTimeseriesResults(ctx, cfg, mgr)
	if err != nil {
		return err
	}

	// Print single header for the entire timeseries analysis only if output is text
	if cfg.Output.Format == schema.TextOut {
		internal.LogTimeseriesHeader(cfg.Git, cfg.Scoring, cfg.Timeseries)
	}

	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteTimeseries(w, result, cfg.Output, cfg.Runtime, duration)
	}, "Wrote timeseries table")
}

// GetHotspotTimeseriesResults runs the timeseries analysis and returns the results.
func GetHotspotTimeseriesResults(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) (schema.TimeseriesResult, time.Duration, error) {
	start := time.Now()

	// Get timeseries-specific parameters from config
	path := cfg.Timeseries.Path
	interval := cfg.Timeseries.Interval
	numPoints := cfg.Timeseries.Points

	if path == "" {
		return schema.TimeseriesResult{}, 0, errors.New("--path is required for timeseries analysis (e.g., 'src/main.go' or 'pkg/mypackage')")
	}
	if interval == 0 {
		return schema.TimeseriesResult{}, 0, errors.New("--interval is required (e.g., --interval 6months or --interval 180d)")
	}
	if numPoints < 1 {
		return schema.TimeseriesResult{}, 0, fmt.Errorf("--points must be at least 1 (received %d). Use --points 3 to --points 10 for meaningful trends", numPoints)
	}

	client := git.NewLocalGitClient()

	// Normalize and validate the path relative to repo root
	normalizedPath, err := contract.NormalizeTimeseriesPath(cfg.Git.RepoPath, path)
	if err != nil {
		return schema.TimeseriesResult{}, 0, fmt.Errorf("invalid path %q: %w. Path must be relative to the repository root (%s)", path, err, cfg.Git.RepoPath)
	}

	// Check if path exists and determine if it's a file or folder
	fullPath := filepath.Join(cfg.Git.RepoPath, normalizedPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return schema.TimeseriesResult{}, 0, fmt.Errorf("path %q does not exist in repository. Use 'hotspot files' to see available paths", path)
	}
	isFolder := info.IsDir()

	// Execute the timeseries analysis
	anchor := cfg.Git.GetEndTime()
	timeseriesPoints := runTimeseriesAnalysis(ctx, cfg.Git, cfg.Scoring, client, normalizedPath, isFolder, anchor, interval, numPoints, mgr)

	result := schema.TimeseriesResult{Points: timeseriesPoints}
	return result, time.Since(start), nil
}

// ExecuteHotspotMetrics displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func ExecuteHotspotMetrics(_ context.Context, cfg *config.Config, _ iocache.CacheManager) error {
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteMetrics(w, cfg.Scoring.CustomWeights, cfg.Output)
	}, "Wrote metrics info")
}

// ExecuteHotspotCheck runs the check command for CI/CD gating.
// It analyzes only files changed between base and target refs, checks them against thresholds,
// and returns a non-zero exit code if any files exceed the thresholds.
func ExecuteHotspotCheck(ctx context.Context, cfg *config.Config, mgr iocache.CacheManager) error {
	start := time.Now()

	builder := NewCheckResultBuilder(ctx, cfg.Git, cfg.Scoring, cfg.Compare, mgr)

	// Validate prerequisites
	_, err := builder.ValidatePrerequisites()
	if err != nil {
		return err
	}
	if result := builder.GetResult(); result != nil {
		// Early success case
		printCheckResult(result, time.Since(start))
		return nil
	}

	// Prepare analysis config
	_, err = builder.PrepareAnalysisConfig()
	if err != nil {
		return err
	}

	// Run analysis
	_, err = builder.RunAnalysis()
	if err != nil {
		return err
	}

	// Compute metrics
	builder.ComputeMetrics()

	// Build result
	builder.BuildResult()

	if result := builder.GetResult(); result != nil {
		printCheckResult(result, time.Since(start))

		// Return error if check failed
		if !result.Passed {
			fmt.Printf("%d violation(s) found\n", len(result.FailedFiles))
			os.Exit(1)
		}
	}
	return nil
}
