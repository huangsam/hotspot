// Package internal has helpers that are exclusive to the app runtime.
package internal

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
)

// LogAnalysisHeader prints a concise, 2-line header for each analysis phase.
func LogAnalysisHeader(cfg *contract.Config) {
	repoName := filepath.Base(cfg.Git.RepoPath)
	if repoName == "" || repoName == "." {
		repoName = "current"
	}

	// Line 1: The analysis summary (Repo and Mode)
	fmt.Printf("Repo: %s (Mode: %s)\n", repoName, cfg.Scoring.Mode)

	// Line 2: The actual date range being analyzed
	fmt.Printf("Range: %s → %s\n", cfg.Git.StartTime.Format(contract.DateTimeFormat), cfg.Git.EndTime.Format(contract.DateTimeFormat))
}

// LogTimeseriesHeader prints a header for timeseries analysis.
func LogTimeseriesHeader(cfg *contract.Config, intervalDuration time.Duration, numPoints int) {
	repoName := filepath.Base(cfg.Git.RepoPath)
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Printf("Repo: %s (Mode: %s)\n", repoName, cfg.Scoring.Mode)
	fmt.Printf("Timeseries: %d data points (interval: %v)\n", numPoints, intervalDuration)
}

// LogCompareHeader prints a header for comparison analysis.
func LogCompareHeader(cfg *contract.Config) {
	repoName := filepath.Base(cfg.Git.RepoPath)
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Printf("Repo: %s (Mode: %s)\n", repoName, cfg.Scoring.Mode)
	fmt.Printf("Comparing: %s ↔ %s (lookback: %v)\n", cfg.Compare.BaseRef, cfg.Compare.TargetRef, cfg.Compare.Lookback)
}
