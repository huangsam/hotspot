// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"fmt"
	"sort"

	"github.com/huangsam/hotspot/internal"
)

// ExecuteHotspotFiles runs the file-level analysis and prints results to stdout.
// It serves as the main entry point for the 'files' mode.
func ExecuteHotspotFiles(cfg *internal.Config) {
	client := internal.NewLocalGitClient()
	ranked, err := AnalyzeFiles(cfg, client)
	if err != nil || len(ranked) == 0 {
		return
	}
	internal.PrintFileResults(ranked, cfg)
}

// ExecuteHotspotFolders runs the folder-level analysis and prints results to stdout.
// It serves as the main entry point for the 'folders' mode.
func ExecuteHotspotFolders(cfg *internal.Config) {
	client := internal.NewLocalGitClient()
	ranked, err := AnalyzeFolders(cfg, client)
	if err != nil || len(ranked) == 0 {
		return
	}
	internal.PrintFolderResults(ranked, cfg)
}

// ExecuteHotspotCompare runs two file-level analyses (Base and Target)
// based on Git references and computes the delta results.
func ExecuteHotspotCompare(cfg *internal.Config) {
	// 1. Instantiate the single GitClient for both runs (still a pointer)
	client := internal.NewLocalGitClient()

	// --- A. Prepare and Run the Base (Before) Analysis ---

	// Convert the BaseRef commit to a time window (e.g., 6 months before that commit)
	baseStartTime, baseEndTime, err := getAnalysisWindowForRef(client, cfg.RepoPath, cfg.BaseRef, cfg.Lookback)
	if err != nil {
		internal.LogFatal(fmt.Sprintf("Failed to resolve time window for BaseRef '%s'", cfg.BaseRef), err)
		return
	}

	// Create the isolated config for the Base run (with the correct time window)
	cfgBase := cfg.CloneWithTimeWindow(baseStartTime, baseEndTime)

	// Run the analysis for the Base state
	baseMetrics, err := AnalyzeFiles(cfgBase, client)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Base Analysis failed for ref %s", cfg.BaseRef))
		return
	}

	// --- B. Prepare and Run the Target (After) Analysis ---

	// Convert the TargetRef commit to a time window
	targetStartTime, targetEndTime, err := getAnalysisWindowForRef(client, cfg.RepoPath, cfg.TargetRef, cfg.Lookback)
	if err != nil {
		internal.LogFatal(fmt.Sprintf("Failed to resolve time window for TargetRef '%s'", cfg.TargetRef), err)
		return
	}

	// Create the isolated config for the Target run
	cfgTarget := cfg.CloneWithTimeWindow(targetStartTime, targetEndTime)

	// Run the analysis for the Target state
	targetMetrics, err := AnalyzeFiles(cfgTarget, client)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Target Analysis failed for ref %s", cfg.TargetRef))
		return
	}

	// --- C. Compute Delta and Output Results ---

	// Pass the results to the comparison function
	sort.Slice(baseMetrics, func(i, j int) bool {
		return baseMetrics[i].Path < baseMetrics[j].Path
	})
	sort.Slice(targetMetrics, func(i, j int) bool {
		return targetMetrics[i].Path < targetMetrics[j].Path
	})
	comparisonResults := compareFileMetrics(baseMetrics, targetMetrics)

	// Output the final comparison table
	internal.PrintComparisonResults(comparisonResults, cfg)
}
