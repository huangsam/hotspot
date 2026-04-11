// Package internal has helpers that are exclusive to the app runtime.
package internal

import (
	"fmt"
	"path/filepath"

	"github.com/huangsam/hotspot/internal/contract"
)

// LogAnalysisHeader prints a concise, 2-line header for each analysis phase.
func LogAnalysisHeader(git contract.GitSettings, scoring contract.ScoringSettings, _ contract.RuntimeSettings, _ contract.OutputSettings) {
	repoName := filepath.Base(git.GetRepoPath())
	if repoName == "" || repoName == "." {
		repoName = "current"
	}

	// Line 1: The analysis summary (Repo and Mode)
	fmt.Printf("Repo: %s (Mode: %s)\n", repoName, scoring.GetMode())

	// Line 2: The actual date range being analyzed
	fmt.Printf("Range: %s → %s\n", git.GetStartTime().Format(contract.DateTimeFormat), git.GetEndTime().Format(contract.DateTimeFormat))
}

// LogTimeseriesHeader prints a header for timeseries analysis.
func LogTimeseriesHeader(git contract.GitSettings, scoring contract.ScoringSettings, timeseries contract.TimeseriesSettings) {
	repoName := filepath.Base(git.GetRepoPath())
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Printf("Repo: %s (Mode: %s)\n", repoName, scoring.GetMode())
	fmt.Printf("Timeseries: %d data points (interval: %v)\n", timeseries.GetPoints(), timeseries.GetInterval())
}

// LogCompareHeader prints a header for comparison analysis.
func LogCompareHeader(git contract.GitSettings, scoring contract.ScoringSettings, compare contract.ComparisonSettings) {
	repoName := filepath.Base(git.GetRepoPath())
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Printf("Repo: %s (Mode: %s)\n", repoName, scoring.GetMode())
	fmt.Printf("Comparing: %s ↔ %s (lookback: %v)\n", compare.GetBaseRef(), compare.GetTargetRef(), compare.GetLookback())
}
