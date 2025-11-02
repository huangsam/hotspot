// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"fmt"
	"strings"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// ExecuteHotspotFiles contains the application's core business logic for file-level analysis.
func ExecuteHotspotFiles(cfg *internal.Config) {
	var files []string

	// --- 1. Aggregation Phase ---
	fmt.Printf("🔎 Aggregating recent activity since %s\n", cfg.StartTime.Format(internal.DateTimeFormat))
	if err := aggregateRecent(cfg); err != nil {
		internal.LogWarning("Cannot aggregate recent activity")
	}

	// --- 2. File List Building and Filtering ---
	// Build file list from union of recent maps so we only analyze files touched since StartTime
	seen := make(map[string]bool)

	// Add files seen in recent commit activity
	for k := range schema.GetRecentCommitsMapGlobal() {
		seen[k] = true
	}
	// Add files seen in recent churn activity
	for k := range schema.GetRecentChurnMapGlobal() {
		seen[k] = true
	}
	// Add files seen in recent contributor activity
	for k := range schema.GetRecentContribMapGlobal() {
		seen[k] = true
	}

	for f := range seen {
		// apply path filter
		if cfg.PathFilter != "" && !strings.HasPrefix(f, cfg.PathFilter) {
			continue
		}

		// apply excludes filter
		if internal.ShouldIgnore(f, cfg.Excludes) {
			continue
		}
		files = append(files, f)
	}

	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return
	}

	// --- 3. Core Analysis and Initial Ranking ---
	fmt.Printf("🧠 hotspot: Analyzing %s (Mode: %s)\n", cfg.RepoPath, cfg.Mode)
	fmt.Printf("📅 Range: %s → %s\n", cfg.StartTime.Format(internal.DateTimeFormat), cfg.EndTime.Format(internal.DateTimeFormat))

	results := analyzeRepo(cfg, files)
	ranked := rankFiles(results, cfg.ResultLimit)

	// --- 4. Optional --follow Re-analysis and Re-ranking ---
	// If the user requested a follow-pass, re-analyze the top N files using
	// git --follow to account for renames/history and then re-rank.
	if cfg.Follow && len(ranked) > 0 {
		// Determine the number of files to re-analyze (min of limit or actual results)
		n := min(cfg.ResultLimit, len(ranked))

		fmt.Printf("🔁 Running --follow re-analysis for top %d files...\n", n)

		for i := range n {
			f := ranked[i]

			// re-analyze with follow enabled (passing 'true' for the follow flag)
			rean := analyzeFileCommon(cfg, f.Path, true)

			// preserve path but update metrics and score
			rean.Path = f.Path
			ranked[i] = rean
		}

		// re-rank after follow pass
		ranked = rankFiles(ranked, cfg.ResultLimit)
	}

	// --- 5. Output Results ---
	internal.PrintResults(ranked, cfg)
}
