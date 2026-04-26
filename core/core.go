// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/huangsam/hotspot/core/algo"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/outwriter"
	"github.com/huangsam/hotspot/schema"
)

// ExecutorFunc defines the function signature for executing different analysis modes.
type ExecutorFunc func(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager, writer outwriter.FormatProvider) error

// ExecuteHotspotFiles runs the file-level analysis and prints results to stdout.
// It serves as the main entry point for the 'files' mode.
func ExecuteHotspotFiles(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager, writer outwriter.FormatProvider) error {
	ranked, duration, err := GetHotspotFilesResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteFiles(w, ranked, cfg.Output, cfg.Runtime, duration)
	}, "Wrote files table")
}

// GetHotspotFilesResults runs the file-level analysis and returns the ranked results.
func GetHotspotFilesResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) ([]schema.FileResult, time.Duration, error) {
	start := time.Now()
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
func ExecuteHotspotFolders(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager, writer outwriter.FormatProvider) error {
	ranked, duration, err := GetHotspotFoldersResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteFolders(w, ranked, cfg.Output, cfg.Runtime, duration)
	}, "Wrote folders table")
}

// GetHotspotFoldersResults runs the folder-level analysis and returns the ranked results.
func GetHotspotFoldersResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) ([]schema.FolderResult, time.Duration, error) {
	start := time.Now()
	output, err := runFolderAnalysisCore(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, cfg.Output, cfg.Compare, client, mgr)
	if err != nil {
		return nil, 0, err
	}
	ranked := algo.RankFolders(output.FolderResults, cfg.Output.ResultLimit)
	return ranked, time.Since(start), nil
}

// ExecuteHotspotCompare runs two file-level analyses (Base and Target)
// based on Git references and computes the delta results.
func ExecuteHotspotCompare(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager, writer outwriter.FormatProvider) error {
	internal.LogCompareHeader(cfg.Git, cfg.Scoring, cfg.Compare)

	comparisonResult, duration, err := GetHotspotCompareResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteComparison(w, comparisonResult, cfg.Output, cfg.Runtime, duration)
	}, "Wrote file comparison table")
}

// GetHotspotCompareResults runs the file-level comparison analysis and returns the results.
func GetHotspotCompareResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) (schema.ComparisonResult, time.Duration, error) {
	start := time.Now()

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
func ExecuteHotspotCompareFolders(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager, writer outwriter.FormatProvider) error {
	internal.LogCompareHeader(cfg.Git, cfg.Scoring, cfg.Compare)

	result, duration, err := GetHotspotCompareFoldersResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}

	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteComparison(w, result, cfg.Output, cfg.Runtime, duration)
	}, "Wrote folder comparison table")
}

// GetHotspotCompareFoldersResults runs the folder-level comparison analysis and returns the results.
func GetHotspotCompareFoldersResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) (schema.ComparisonResult, time.Duration, error) {
	start := time.Now()

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.Compare.BaseRef, mgr)
	if err != nil {
		return schema.ComparisonResult{}, 0, err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.Compare.TargetRef, mgr)
	if err != nil {
		return schema.ComparisonResult{}, 0, err
	}
	comparisonResult := compareFolderMetrics(baseOutput.FolderResults, targetOutput.FolderResults, cfg.Output.ResultLimit, string(cfg.Scoring.Mode))
	return comparisonResult, time.Since(start), nil
}

// ExecuteHotspotTimeseries runs multiple analyses over overlapping, dynamic-lookback time windows.
func ExecuteHotspotTimeseries(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager, writer outwriter.FormatProvider) error {
	result, duration, err := GetHotspotTimeseriesResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}

	internal.LogTimeseriesHeader(cfg.Git, cfg.Scoring, cfg.Timeseries)

	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteTimeseries(w, result, cfg.Output, cfg.Runtime, duration)
	}, "Wrote timeseries table")
}

// GetHotspotTimeseriesResults runs the timeseries analysis and returns the results.
func GetHotspotTimeseriesResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) (schema.TimeseriesResult, time.Duration, error) {
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

	// Normalize and validate the path relative to repo root
	normalizedPath, err := schema.NormalizeTimeseriesPath(cfg.Git.RepoPath, path)
	if err != nil {
		return schema.TimeseriesResult{}, 0, fmt.Errorf("invalid path %q: %w. Path must be relative to the repository root (%s)", path, err, cfg.Git.RepoPath)
	}

	// Check if path exists and determine if it's a file or folder via Git
	// This is more robust than os.Stat as it works for non-checked-out refs
	// and allows for proper mocking in tests.
	ref := cfg.Compare.GetTargetRef()
	if ref == "" {
		ref = "HEAD"
	}
	files, err := client.ListFilesAtRef(ctx, cfg.Git.RepoPath, ref)
	if err != nil {
		return schema.TimeseriesResult{}, 0, fmt.Errorf("failed to verify path %q: %w", path, err)
	}

	exists := false
	isFolder := false
	for _, f := range files {
		if f == normalizedPath {
			exists = true
			break
		}
		// If normalizedPath is a prefix followed by a slash, it's a folder
		if len(f) > len(normalizedPath) && f[len(normalizedPath)] == '/' && f[:len(normalizedPath)] == normalizedPath {
			exists = true
			isFolder = true
			break
		}
	}

	if !exists {
		return schema.TimeseriesResult{}, 0, fmt.Errorf("path %q does not exist in repository at ref %q. Use 'hotspot files' to see available paths", path, ref)
	}

	// Execute the timeseries analysis
	anchor := cfg.Git.GetEndTime()
	timeseriesPoints := runTimeseriesAnalysis(ctx, cfg.Git, cfg.Scoring, client, normalizedPath, isFolder, anchor, interval, numPoints, mgr)

	result := schema.TimeseriesResult{Points: timeseriesPoints}
	return result, time.Since(start), nil
}

// ExecuteHotspotBlastRadius runs the blast radius analysis and prints results to stdout.
func ExecuteHotspotBlastRadius(ctx context.Context, cfg *config.Config, client git.Client, writer outwriter.FormatProvider) error {
	start := time.Now()
	result, err := GetHotspotBlastRadiusResults(ctx, cfg, client, cfg.Output.ResultLimit, 0.3) // Default threshold
	if err != nil {
		return err
	}
	duration := time.Since(start)

	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteBlastRadius(w, result, cfg.Output, cfg.Runtime, duration)
	}, "Wrote blast radius table")
}

// ExecuteHotspotMetrics displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func ExecuteHotspotMetrics(_ context.Context, cfg *config.Config, _ git.Client, _ iocache.CacheManager, writer outwriter.FormatProvider) error {
	return outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
		return writer.WriteMetrics(w, cfg.Scoring.CustomWeights, cfg.Output)
	}, "Wrote metrics info")
}

// ExecuteHotspotCheck runs the check command for CI/CD gating.
// It analyzes only files changed between base and target refs, checks them against thresholds,
// and returns a non-zero exit code if any files exceed the thresholds.
func ExecuteHotspotCheck(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) error {
	result, duration, err := GetHotspotCheckResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}

	if result != nil {
		printCheckResult(result, duration)

		// Return error if check failed
		if !result.Passed {
			fmt.Fprintf(os.Stderr, "%d violation(s) found\n", len(result.FailedFiles))
			os.Exit(1)
		}
	}
	return nil
}

// GetHotspotCheckResults runs the policy check and returns the result object.
func GetHotspotCheckResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) (*schema.CheckResult, time.Duration, error) {
	start := time.Now()

	builder := NewCheckResultBuilder(ctx, cfg.Git, cfg.Scoring, cfg.Compare, client, mgr)

	// Validate prerequisites
	_, err := builder.ValidatePrerequisites()
	if err != nil {
		return nil, 0, err
	}
	if result := builder.GetResult(); result != nil {
		// Early success case (e.g. no files changed)
		return result, time.Since(start), nil
	}

	// Prepare analysis config
	_, err = builder.PrepareAnalysisConfig()
	if err != nil {
		return nil, 0, err
	}

	// Run analysis
	_, err = builder.RunAnalysis()
	if err != nil {
		return nil, 0, err
	}

	// Compute metrics
	builder.ComputeMetrics()

	// Build result
	builder.BuildResult()

	return builder.GetResult(), time.Since(start), nil
}
