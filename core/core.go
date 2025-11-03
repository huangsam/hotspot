// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/huangsam/hotspot/internal"
)

// ExecuteHotspotFiles contains the application's core business logic for file-level analysis.
func ExecuteHotspotFiles(cfg *internal.Config) {
	var files []string

	// --- 1. Aggregation Phase ---
	fmt.Printf("🔎 Aggregating activity since %s\n", cfg.StartTime.Format(internal.DateTimeFormat))
	output, err := aggregateActivity(cfg)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
	}

	// --- 2. File List Building and Filtering ---
	// Build file list from union of recent maps so we only analyze files touched since StartTime
	seen := make(map[string]bool)

	// Add files seen in recent commit activity
	for k := range output.CommitMap {
		seen[k] = true
	}
	// Add files seen in recent churn activity
	for k := range output.ChurnMap {
		seen[k] = true
	}
	// Add files seen in recent contributor activity
	for k := range output.ContribMap {
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

	results := analyzeRepo(cfg, output, files)
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
			rean := analyzeFileCommon(cfg, f.Path, output, true)

			// preserve path but update metrics and score
			rean.Path = f.Path
			ranked[i] = rean
		}

		// re-rank after follow pass
		ranked = rankFiles(ranked, cfg.ResultLimit)
	}

	// --- 5. Output Results ---
	internal.PrintFileResults(ranked, cfg)
}

// ExecuteHotspotFolders contains the application's core business logic for folder-level analysis.
func ExecuteHotspotFolders(cfg *internal.Config) {
	var files []string

	// --- 1. Aggregation Phase (Identical to file analysis) ---
	// The aggregation is file-level, so this part is the same.
	fmt.Printf("🔎 Aggregating activity since %s\n", cfg.StartTime.Format(internal.DateTimeFormat))
	output, err := aggregateActivity(cfg)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
	}

	// --- 2. File List Building and Filtering (Slightly modified) ---
	// Build file list from union of recent maps so we only analyze files touched since StartTime
	seenFiles := make(map[string]bool)
	seenFolders := make(map[string]bool) // New map to track unique folders

	// Add files seen in recent activity
	for k := range output.CommitMap {
		seenFiles[k] = true
	}
	for k := range output.ChurnMap {
		seenFiles[k] = true
	}
	for k := range output.ContribMap {
		seenFiles[k] = true
	}

	for f := range seenFiles {
		// apply path filter
		if cfg.PathFilter != "" && !strings.HasPrefix(f, cfg.PathFilter) {
			continue
		}

		// apply excludes filter
		if internal.ShouldIgnore(f, cfg.Excludes) {
			continue
		}

		// Add the file to the list for later processing
		files = append(files, f)

		// Extract the directory (folder) from the file path
		folderPath := filepath.Dir(f)

		// Exclude the root folder when not explicitly requested by PathFilter
		// A common convention is to treat "." (current directory) as a special case
		if cfg.PathFilter == "" && folderPath == "." {
			continue
		}

		// Track unique folders
		if !seenFolders[folderPath] {
			seenFolders[folderPath] = true
		}
	}

	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return
	}

	// --- 3. Core Analysis and Initial Ranking ---
	// The analysis function needs to be adapted or replaced.
	// Since we're running in folder mode, we don't analyze individual files
	// but aggregate their metrics into folder results.
	fmt.Printf("🧠 hotspot: Analyzing %s (Mode: %s)\n", cfg.RepoPath, cfg.Mode)
	fmt.Printf("📅 Range: %s → %s\n", cfg.StartTime.Format(internal.DateTimeFormat), cfg.EndTime.Format(internal.DateTimeFormat))

	// New logic: Aggregate file results into folder results
	fileMetrics := analyzeRepo(cfg, output, files)
	folderResults := aggregateAndScoreFolders(cfg, fileMetrics)

	// The Folder-mode ranking will rank the folder results
	ranked := rankFolders(folderResults, cfg.ResultLimit)

	// --- 4. Optional --follow Re-analysis and Re-ranking ---
	// This phase is typically skipped for folder analysis as 'git --follow'
	// is file-centric. We will remove this section to simplify and align with
	// the folder-level goal. A re-analysis would require aggregating the
	// re-analyzed file metrics again.

	// --- 5. Output Results ---
	internal.PrintFolderResults(ranked, cfg)
}
