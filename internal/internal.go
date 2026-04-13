// Package internal has helpers that are exclusive to the app runtime.
package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// LogAnalysisHeader prints a concise, 2-line header for each analysis phase.
func LogAnalysisHeader(git config.GitSettings, scoring config.ScoringSettings, _ config.RuntimeSettings, _ config.OutputSettings) {
	repoName := filepath.Base(git.GetRepoPath())
	if repoName == "" || repoName == "." {
		repoName = "current"
	}

	// Line 1: The analysis summary (Repo and Mode)
	fmt.Fprintf(os.Stderr, "Repo: %s (Mode: %s)\n", repoName, scoring.GetMode())

	// Line 2: The actual date range being analyzed
	fmt.Fprintf(os.Stderr, "Range: %s → %s\n", git.GetStartTime().Format(schema.DateTimeFormat), git.GetEndTime().Format(schema.DateTimeFormat))
}

// LogTimeseriesHeader prints a header for timeseries analysis.
func LogTimeseriesHeader(git config.GitSettings, scoring config.ScoringSettings, timeseries config.TimeseriesSettings) {
	repoName := filepath.Base(git.GetRepoPath())
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Fprintf(os.Stderr, "Repo: %s (Mode: %s)\n", repoName, scoring.GetMode())
	fmt.Fprintf(os.Stderr, "Timeseries: %d data points (interval: %v)\n", timeseries.GetPoints(), timeseries.GetInterval())
}

// LogCompareHeader prints a header for comparison analysis.
func LogCompareHeader(git config.GitSettings, scoring config.ScoringSettings, compare config.ComparisonSettings) {
	repoName := filepath.Base(git.GetRepoPath())
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Fprintf(os.Stderr, "Repo: %s (Mode: %s)\n", repoName, scoring.GetMode())
	fmt.Fprintf(os.Stderr, "Comparing: %s ↔ %s (lookback: %v)\n", compare.GetBaseRef(), compare.GetTargetRef(), compare.GetLookback())
}
