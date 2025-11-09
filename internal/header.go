package internal

import (
	"fmt"
	"path/filepath"
	"time"
)

// LogAnalysisHeader prints a concise, 2-line header for each analysis phase.
func LogAnalysisHeader(cfg *Config) {
	repoName := filepath.Base(cfg.RepoPath)
	if repoName == "" || repoName == "." {
		repoName = "current"
	}

	// Line 1: The analysis summary (Repo and Mode)
	fmt.Printf("🔎 Repo: %s (Mode: %s)\n", repoName, cfg.Mode)

	// Line 2: The actual date range being analyzed
	fmt.Printf("📅 Range: %s → %s\n", cfg.StartTime.Format(DateTimeFormat), cfg.EndTime.Format(DateTimeFormat))
}

// LogTimeseriesHeader prints a header for timeseries analysis.
func LogTimeseriesHeader(cfg *Config, totalInterval time.Duration, numPoints int) {
	repoName := filepath.Base(cfg.RepoPath)
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Printf("🔎 Repo: %s (Mode: %s)\n", repoName, cfg.Mode)
	fmt.Printf("📅 Total Interval: %s → %s (%d points)\n",
		time.Now().Add(-totalInterval).Format(DateTimeFormat),
		time.Now().Format(DateTimeFormat),
		numPoints)
}

// LogCompareHeader prints a header for comparison analysis.
func LogCompareHeader(cfg *Config) {
	repoName := filepath.Base(cfg.RepoPath)
	if repoName == "" || repoName == "." {
		repoName = "current"
	}
	fmt.Printf("🔎 Repo: %s (Mode: %s)\n", repoName, cfg.Mode)
	fmt.Printf("📊 Comparing: %s ↔ %s (lookback: %v)\n", cfg.BaseRef, cfg.TargetRef, cfg.Lookback)
}
