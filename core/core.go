// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"context"
	"time"

	"github.com/huangsam/hotspot/internal"
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
	comparisonOutput := compareFileResults(baseOutput.FileResults, targetOutput.FileResults, cfg.ResultLimit)
	duration := time.Since(start)
	return internal.PrintComparisonResults(comparisonOutput, cfg, duration)
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
	comparisonOutput := compareFolderMetrics(baseOutput.FolderResults, targetOutput.FolderResults, cfg.ResultLimit)
	duration := time.Since(start)
	return internal.PrintComparisonResults(comparisonOutput, cfg, duration)
}
