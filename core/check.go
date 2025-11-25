package core

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// ExecuteHotspotCheck runs the check command for CI/CD gating.
// It analyzes only files changed between base and target refs, checks them against thresholds,
// and returns a non-zero exit code if any files exceed the thresholds.
func ExecuteHotspotCheck(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) error {
	start := time.Now()

	builder := NewCheckResultBuilder(ctx, cfg, mgr)

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
func printCheckResult(result *schema.CheckResult, duration time.Duration) {
	printCheckHeader(result, duration)

	if result.Passed {
		printCheckSuccess(result)
	} else {
		printCheckFailure(result)
	}
}

// printCheckHeader prints the common header information for check results.
func printCheckHeader(result *schema.CheckResult, duration time.Duration) {
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
}

// printCheckSuccess prints the success case output.
func printCheckSuccess(result *schema.CheckResult) {
	fmt.Printf("✅ All files passed policy checks\n\n")
	fmt.Println("Scores observed:")

	for _, mode := range result.CheckedModes {
		score := result.MaxScores[mode]
		files := result.MaxScoreFiles[mode]
		avgScore := result.AvgScores[mode]

		if len(files) == 0 {
			fmt.Printf("  %s: max=%.1f, avg=%.1f\n", mode, score, avgScore)
			continue
		}

		// Show the primary file that achieved max score (first one if tie)
		fileName := files[0].Path
		if len(files) > 1 {
			fileName += fmt.Sprintf(" (+%d more)", len(files)-1)
		}

		fmt.Printf("  %s: max=%.1f (%s), avg=%.1f\n", mode, score, fileName, avgScore)
	}
}

// printCheckFailure prints the failure case output.
func printCheckFailure(result *schema.CheckResult) {
	// Print failed files grouped by mode
	fmt.Printf("❌ Policy check failed: %d violation(s) found across %d files\n\n", len(result.FailedFiles), result.TotalFiles)

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

		// Sort by score descending
		sort.Slice(files, func(i, j int) bool {
			return files[i].Score > files[j].Score
		})

		fmt.Printf("Mode: %s (%d violations)\n", mode, len(files))

		// Show top 5 violations, with "+X more" if needed
		maxToShow := 5
		shown := 0
		for _, f := range files {
			if shown >= maxToShow {
				remaining := len(files) - shown
				if remaining > 0 {
					fmt.Printf("  ... and %d more\n", remaining)
				}
				break
			}
			fmt.Printf("  - %s (score: %.1f > threshold: %.1f)\n", f.Path, f.Score, f.Threshold)
			shown++
		}
		fmt.Println()
	}
}
