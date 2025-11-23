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

	// Validate prerequisites and get files to analyze
	filesToAnalyze, err := validateCheckPrerequisites(ctx, client, cfg)
	if err != nil {
		return err
	}
	if filesToAnalyze == nil {
		return nil // Early success cases handled inside
	}

	// Prepare analysis configuration
	cfgTarget, err := prepareAnalysisConfig(ctx, client, cfg)
	if err != nil {
		return err
	}

	// Run analysis
	fileResults, err := runCheckAnalysis(ctx, cfgTarget, client, mgr, filesToAnalyze)
	if err != nil {
		return err
	}

	// Compute metrics and check against thresholds
	maxScores, failedFiles, maxScoreFiles := computeCheckMetrics(fileResults, cfg)

	// Build and print result
	result := buildCheckResult(filesToAnalyze, cfg, maxScores, failedFiles, maxScoreFiles)
	printCheckResult(result, time.Since(start))

	// Return error if check failed
	if !result.Passed {
		return fmt.Errorf("policy check failed: %d violation(s) found", len(result.FailedFiles))
	}

	return nil
}

// validateCheckPrerequisites validates config and returns files to analyze, or nil for early success.
func validateCheckPrerequisites(ctx context.Context, client contract.GitClient, cfg *contract.Config) ([]string, error) {
	// Validate that compare mode is enabled
	if !cfg.CompareMode {
		return nil, fmt.Errorf("check requires --base-ref and --target-ref flags")
	}

	// Get the list of changed files between base and target refs
	changedFiles, err := client.GetChangedFilesBetweenRefs(ctx, cfg.RepoPath, cfg.BaseRef, cfg.TargetRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		fmt.Println("No files changed between refs - check passed")
		return nil, nil
	}

	// Filter changed files to only include those we want to analyze
	filesToAnalyze := filterChangedFiles(changedFiles, cfg.Excludes)
	if len(filesToAnalyze) == 0 {
		fmt.Println("No relevant files to check (all excluded) - check passed")
		return nil, nil
	}

	return filesToAnalyze, nil
}

// prepareAnalysisConfig sets up the time window and returns the analysis config.
func prepareAnalysisConfig(ctx context.Context, client contract.GitClient, cfg *contract.Config) (*contract.Config, error) {
	// Get the time window for the target ref
	_, targetEndTime, err := getAnalysisWindowForRef(ctx, client, cfg.RepoPath, cfg.TargetRef, cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve time window for target ref '%s': %w", cfg.TargetRef, err)
	}

	// Create config for target ref analysis
	targetStartTime := targetEndTime.Add(-cfg.Lookback)
	return cfg.CloneWithTimeWindow(targetStartTime, targetEndTime), nil
}

// runCheckAnalysis performs the aggregation and file analysis.
func runCheckAnalysis(ctx context.Context, cfgTarget *contract.Config, client contract.GitClient, mgr contract.CacheManager, filesToAnalyze []string) ([]schema.FileResult, error) {
	// Run aggregation once (shared for all modes)
	output, err := agg.CachedAggregateActivity(ctx, cfgTarget, client, mgr)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate activity: %w", err)
	}

	// Analyze files once (all scores are computed upfront in FileResult)
	cfgDefault := cfgTarget.Clone()
	cfgDefault.Mode = schema.HotMode // Mode doesn't matter since all scores are computed
	return analyzeRepo(ctx, cfgDefault, client, output, filesToAnalyze), nil
}

// computeCheckMetrics calculates max scores and identifies failed files.
func computeCheckMetrics(fileResults []schema.FileResult, cfg *contract.Config) (map[schema.ScoringMode]float64, []schema.CheckFailedFile, map[schema.ScoringMode][]schema.CheckMaxScoreFile) {
	// Compute max scores for each mode
	maxScores := make(map[schema.ScoringMode]float64)
	maxScoreFiles := make(map[schema.ScoringMode][]schema.CheckMaxScoreFile)

	for _, mode := range schema.AllScoringModes {
		maxScore := 0.0
		var filesWithMax []schema.CheckMaxScoreFile

		for _, file := range fileResults {
			score := file.AllScores[mode]
			if score > maxScore {
				maxScore = score
				filesWithMax = []schema.CheckMaxScoreFile{{
					Path:   file.Path,
					Owners: file.Owners,
				}}
			} else if score == maxScore && maxScore > 0 {
				// Include files that tie for the max score
				filesWithMax = append(filesWithMax, schema.CheckMaxScoreFile{
					Path:   file.Path,
					Owners: file.Owners,
				})
			}
		}

		maxScores[mode] = maxScore
		maxScoreFiles[mode] = filesWithMax
	}

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

	return maxScores, failedFiles, maxScoreFiles
}

// buildCheckResult constructs the final CheckResult.
func buildCheckResult(filesToAnalyze []string, cfg *contract.Config, maxScores map[schema.ScoringMode]float64, failedFiles []schema.CheckFailedFile, maxScoreFiles map[schema.ScoringMode][]schema.CheckMaxScoreFile) schema.CheckResult {
	return schema.CheckResult{
		Passed:        len(failedFiles) == 0,
		FailedFiles:   failedFiles,
		TotalFiles:    len(filesToAnalyze),
		CheckedModes:  schema.AllScoringModes,
		BaseRef:       cfg.BaseRef,
		TargetRef:     cfg.TargetRef,
		Thresholds:    cfg.RiskThresholds,
		MaxScores:     maxScores,
		MaxScoreFiles: maxScoreFiles,
		Lookback:      cfg.Lookback,
	}
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
	fmt.Println("Policy Check Results:")

	// Define labels and values for dynamic padding
	labels := []string{"Base:", "Target:", "Lookback:", "Thresholds:"}
	values := []any{
		result.BaseRef,
		result.TargetRef,
		result.Lookback,
		fmt.Sprintf("hot=%.1f, risk=%.1f, complexity=%.1f, stale=%.1f",
			result.Thresholds[schema.HotMode],
			result.Thresholds[schema.RiskMode],
			result.Thresholds[schema.ComplexityMode],
			result.Thresholds[schema.StaleMode]),
	}

	// Find the longest label for consistent padding
	maxLabelLen := 0
	for _, label := range labels {
		if len(label) > maxLabelLen {
			maxLabelLen = len(label)
		}
	}

	// Print each label-value pair with consistent padding
	for i, label := range labels {
		fmt.Printf("  %-*s %v\n", maxLabelLen+1, label, values[i])
	}
	fmt.Println()

	fmt.Printf("Checked %d files in %v\n\n", result.TotalFiles, duration)

	if result.Passed {
		fmt.Printf("✅ All files passed policy checks\n\n")
		fmt.Println("Max scores observed:")

		for _, mode := range result.CheckedModes {
			score := result.MaxScores[mode]
			files := result.MaxScoreFiles[mode]

			if len(files) == 0 {
				fmt.Printf("  %s=%.1f\n", mode, score)
				continue
			}

			// Show the primary file that achieved max score (first one if tie)
			fileName := files[0].Path
			if len(files) > 1 {
				fileName += fmt.Sprintf(" (+%d more)", len(files)-1)
			}

			fmt.Printf("  %s=%.1f (%s)\n", mode, score, fileName)
		}
		return
	}

	// Print failed files grouped by mode
	fmt.Printf("❌ Policy check failed: %d violation(s) found\n\n", len(result.FailedFiles))

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
			fmt.Printf("  - %s (score: %.1f > threshold: %.1f)\n", f.Path, f.Score, f.Threshold)
		}
		fmt.Println()
	}
}
