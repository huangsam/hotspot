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

	"github.com/huangsam/hotspot/core/agg"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/outwriter"
	"github.com/huangsam/hotspot/schema"
)

// ExecutorFunc defines the function signature for executing different analysis modes.
type ExecutorFunc func(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error

// ExecuteHotspotFiles runs the file-level analysis and prints results to stdout.
// It serves as the main entry point for the 'files' mode.
func ExecuteHotspotFiles(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error {
	start := time.Now()
	client := contract.NewLocalGitClient()
	
	// Set global cache manager for analysis tracking
	setGlobalCacheManager(mgr)
	
	output, err := runSingleAnalysisCore(ctx, cfg, client, mgr)
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
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writer.WriteFiles(w, ranked, cfg, duration)
	}, "Wrote files table")
}

// ExecuteHotspotFolders runs the folder-level analysis and prints results to stdout.
// It serves as the main entry point for the 'folders' mode.
func ExecuteHotspotFolders(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error {
	start := time.Now()
	client := contract.NewLocalGitClient()
	
	// Set global cache manager for analysis tracking
	setGlobalCacheManager(mgr)
	
	output, err := runSingleAnalysisCore(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}
	folderResults := agg.AggregateAndScoreFolders(cfg, output.FileResults)
	ranked := rankFolders(folderResults, cfg.ResultLimit)
	duration := time.Since(start)
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writer.WriteFolders(w, ranked, cfg, duration)
	}, "Wrote folders table")
}

// ExecuteHotspotCompare runs two file-level analyses (Base and Target)
// based on Git references and computes the delta results.
func ExecuteHotspotCompare(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error {
	start := time.Now()
	client := contract.NewLocalGitClient()

	// Set global cache manager for analysis tracking
	setGlobalCacheManager(mgr)

	// Print single header for the comparison
	internal.LogCompareHeader(cfg)

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef, mgr)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef, mgr)
	if err != nil {
		return err
	}
	comparisonResult := compareFileResults(baseOutput.FileResults, targetOutput.FileResults, cfg.ResultLimit, string(cfg.Mode))
	duration := time.Since(start)
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writer.WriteComparison(w, comparisonResult, cfg, duration)
	}, "Wrote file comparison table")
}

// ExecuteHotspotCompareFolders runs two folder-level analyses (Base and Target)
// based on Git references and computes the delta results.
// It follows the same pattern as ExecuteHotspotCompare but aggregates to folders
// before performing the comparison.
func ExecuteHotspotCompareFolders(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error {
	start := time.Now()
	client := contract.NewLocalGitClient()

	// Set global cache manager for analysis tracking
	setGlobalCacheManager(mgr)

	// Print single header for the comparison
	internal.LogCompareHeader(cfg)

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef, mgr)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef, mgr)
	if err != nil {
		return err
	}
	comparisonResult := compareFolderMetrics(baseOutput.FolderResults, targetOutput.FolderResults, cfg.ResultLimit, string(cfg.Mode))
	duration := time.Since(start)
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writer.WriteComparison(w, comparisonResult, cfg, duration)
	}, "Wrote folder comparison table")
}

// ExecuteHotspotTimeseries runs multiple analyses over overlapping, dynamic-lookback time windows.
// This implements Strategy 2: Time-Boxed M_min Approximation. The Git search for M_min commits
// is capped by maxSearchDuration to prevent slow full-history traversal on large repos.
func ExecuteHotspotTimeseries(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error {
	start := time.Now()

	// Set global cache manager for analysis tracking
	setGlobalCacheManager(mgr)

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
	client := contract.NewLocalGitClient()

	// Normalize and validate the path relative to repo root
	normalizedPath, err := contract.NormalizeTimeseriesPath(cfg.RepoPath, path)
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

	// Print single header for the entire timeseries analysis
	internal.LogTimeseriesHeader(cfg, interval, numPoints)

	// Execute the timeseries analysis
	timeseriesPoints := runTimeseriesAnalysis(ctx, cfg, client, normalizedPath, isFolder, now, interval, numPoints, mgr)

	result := schema.TimeseriesResult{Points: timeseriesPoints}
	duration := time.Since(start)
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writer.WriteTimeseries(w, result, cfg, duration)
	}, "Wrote timeseries table")
}

// ExecuteHotspotMetrics displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func ExecuteHotspotMetrics(_ context.Context, cfg *contract.Config, _ contract.CacheManager) error {
	writer := outwriter.NewOutWriter()
	return outwriter.WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writer.WriteMetrics(w, cfg.CustomWeights, cfg)
	}, "Wrote metrics info")
}
