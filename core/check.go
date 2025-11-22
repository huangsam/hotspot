package core

import (
	"context"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// ExecuteHotspotCheck runs the check command for CI/CD gating.
// It analyzes only files changed between base and target refs, checks them against thresholds,
// and returns a non-zero exit code if any files exceed the thresholds.
func ExecuteHotspotCheck(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error {
	start := time.Now()
	client := contract.NewLocalGitClient()

	// Validate that compare mode is enabled
	if !cfg.CompareMode {
		return fmt.Errorf("check requires --base-ref and --target-ref flags")
	}

	// Get the list of changed files between base and target refs
	changedFiles, err := client.GetChangedFilesBetweenRefs(ctx, cfg.RepoPath, cfg.BaseRef, cfg.TargetRef)
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		fmt.Println("No files changed between refs - check passed")
		return nil
	}

	// Filter changed files to only include those we want to analyze
	filesToAnalyze := filterChangedFiles(changedFiles, cfg.Excludes)
	if len(filesToAnalyze) == 0 {
		fmt.Println("No relevant files to check (all excluded) - check passed")
		return nil
	}

	// Get the time window for the target ref
	_, targetEndTime, err := getAnalysisWindowForRef(ctx, client, cfg.RepoPath, cfg.TargetRef, cfg.Lookback)
	if err != nil {
		return fmt.Errorf("failed to resolve time window for target ref '%s': %w", cfg.TargetRef, err)
	}

	// Create config for target ref analysis
	targetStartTime := targetEndTime.Add(-cfg.Lookback)
	cfgTarget := cfg.CloneWithTimeWindow(targetStartTime, targetEndTime)

	// Run aggregation once (shared for all modes)
	output, err := agg.CachedAggregateActivity(ctx, cfgTarget, client, mgr)
	if err != nil {
		return fmt.Errorf("failed to aggregate activity: %w", err)
	}

	// Analyze files once (all scores are computed upfront in FileResult)
	cfgDefault := cfgTarget.Clone()
	cfgDefault.Mode = schema.HotMode // Mode doesn't matter since all scores are computed
	fileResults := analyzeRepo(ctx, cfgDefault, client, output, filesToAnalyze)

	// Check all files against thresholds for all modes
	failedFiles := []schema.CheckFailedFile{}
	for _, file := range fileResults {
		for _, mode := range schema.AllScoringModes {
			score := file.AllScores[mode]
			threshold := cfg.RiskThresholds[mode]
			if score > threshold {
				failedFiles = append(failedFiles, schema.CheckFailedFile{
					Path:      file.Path,
					Mode:      mode,
					Score:     score,
					Threshold: threshold,
				})
			}
		}
	}

	// Build result
	result := schema.CheckResult{
		Passed:       len(failedFiles) == 0,
		FailedFiles:  failedFiles,
		TotalFiles:   len(filesToAnalyze),
		CheckedModes: schema.AllScoringModes,
		BaseRef:      cfg.BaseRef,
		TargetRef:    cfg.TargetRef,
	}

	// Print results
	duration := time.Since(start)
	printCheckResult(result, duration)

	// Return error if check failed
	if !result.Passed {
		return fmt.Errorf("policy check failed: %d violation(s) found", len(result.FailedFiles))
	}

	return nil
}

// filterChangedFiles filters the list of changed files based on excludes.
func filterChangedFiles(files []string, excludes []string) []string {
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if !contract.ShouldIgnore(f, excludes) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// printCheckResult prints the check result in a concise format suitable for CI/CD.
func printCheckResult(result schema.CheckResult, duration time.Duration) {
	fmt.Printf("Policy Check Results:\n")
	fmt.Printf("  Base Ref:       %s\n", result.BaseRef)
	fmt.Printf("  Target Ref:     %s\n", result.TargetRef)
	fmt.Printf("  Checked %d files in %v\n\n", result.TotalFiles, duration)

	if result.Passed {
		fmt.Println("All files passed policy checks")
		return
	}

	// Print failed files grouped by mode
	fmt.Printf("Policy check failed: %d file(s) exceeded thresholds\n\n", len(result.FailedFiles))

	// Group by mode for better readability
	modeGroups := make(map[schema.ScoringMode][]schema.CheckFailedFile)
	for _, failed := range result.FailedFiles {
		modeGroups[failed.Mode] = append(modeGroups[failed.Mode], failed)
	}

	for _, mode := range result.CheckedModes {
		files := modeGroups[mode]
		if len(files) == 0 {
			continue
		}

		fmt.Printf("Mode: %s\n", mode)
		for _, f := range files {
			fmt.Printf("  - %s (score: %.1f, threshold: %.1f)\n", f.Path, f.Score, f.Threshold)
		}
		fmt.Println()
	}
}
