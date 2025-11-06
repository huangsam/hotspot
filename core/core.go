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

	// --- 1. Aggregation Phase ---
	fmt.Printf("🔎 Aggregating activity since %s\n", cfg.StartTime.Format(internal.DateTimeFormat))
	output, err := aggregateActivity(cfg, client)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
		return
	}

	// --- 2. File List Building and Filtering ---
	files := buildFilteredFileList(cfg, output)
	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return
	}

	// --- 3. Core Analysis ---
	logAnalysisHeader(cfg)
	results := analyzeRepo(cfg, client, output, files) // Full scored list

	// --- 4. Optional --follow Re-analysis (Limited for Efficiency) ---
	if cfg.Follow && len(results) > 0 {
		// A. Rank the full results to identify the expensive top-N files.
		// NOTE: We rely on rankFiles to return a *new* limited slice.
		rankedForFollow := rankFiles(results, cfg.ResultLimit)

		// B. Run the follow pass only on the limited list, which updates the scores
		// in the original 'results' list, or returns a merged set.
		results = runFollowPass(cfg, client, rankedForFollow, output)
	}

	// --- 5. Final Ranking, Limiting, and Presentation ---
	// The full result set ('results') is ranked, limited, and printed here.
	ranked := rankFiles(results, cfg.ResultLimit)

	internal.PrintFileResults(ranked, cfg)
}

// ExecuteHotspotFolders runs the folder-level analysis and prints results to stdout.
// It serves as the main entry point for the 'folders' mode.
func ExecuteHotspotFolders(cfg *internal.Config) {
	client := internal.NewLocalGitClient()
	// --- 1. Aggregation Phase ---
	fmt.Printf("🔎 Aggregating activity since %s\n", cfg.StartTime.Format(internal.DateTimeFormat))
	output, err := aggregateActivity(cfg, client)
	if err != nil {
		internal.LogWarning("Cannot aggregate activity")
		return
	}

	// --- 2. File List Building and Filtering ---
	files := buildFilteredFileList(cfg, output)
	if len(files) == 0 {
		internal.LogWarning("No files with activity found in the requested window")
		return
	}

	// --- 3. Core Analysis ---
	logAnalysisHeader(cfg)

	// We analyze the *files* to get metrics,
	fileResults := analyzeRepo(cfg, client, output, files)

	// Aggregate and score the folders
	folderResults := aggregateAndScoreFolders(cfg, fileResults)
	ranked := rankFolders(folderResults, cfg.ResultLimit)

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
	baseMetrics, err := analyzeAllFiles(cfgBase, client)
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
	targetMetrics, err := analyzeAllFiles(cfgTarget, client)
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
	comparisonResults := compareFileResults(baseMetrics, targetMetrics, cfg.ResultLimit)

	// Output the final comparison table
	internal.PrintComparisonResults(comparisonResults, cfg)
}

// ExecuteHotspotCompareFolders runs two folder-level analyses (Base and Target)
// based on Git references and computes the delta results.
// It follows the same pattern as ExecuteHotspotCompare but aggregates to folders
// before performing the comparison.
func ExecuteHotspotCompareFolders(cfg *internal.Config) {
	// 1. Instantiate the single GitClient for both runs
	client := internal.NewLocalGitClient()

	// --- A. Prepare and Run the Base (Before) Analysis ---

	// Resolve the time window for the Base reference
	baseStartTime, baseEndTime, err := getAnalysisWindowForRef(client, cfg.RepoPath, cfg.BaseRef, cfg.Lookback)
	if err != nil {
		internal.LogFatal(fmt.Sprintf("Failed to resolve time window for BaseRef '%s'", cfg.BaseRef), err)
		return
	}

	// Create the isolated config for the Base run
	cfgBase := cfg.CloneWithTimeWindow(baseStartTime, baseEndTime)

	// 1a. Run file analysis for the Base state
	baseFileMetrics, err := analyzeAllFiles(cfgBase, client)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Base File Analysis failed for ref %s", cfg.BaseRef))
		return
	}

	// 1b. Aggregate file metrics into folder results for the Base state
	baseFolderResults := aggregateAndScoreFolders(cfgBase, baseFileMetrics)

	// --- B. Prepare and Run the Target (After) Analysis ---

	// Resolve the time window for the Target reference
	targetStartTime, targetEndTime, err := getAnalysisWindowForRef(client, cfg.RepoPath, cfg.TargetRef, cfg.Lookback)
	if err != nil {
		internal.LogFatal(fmt.Sprintf("Failed to resolve time window for TargetRef '%s'", cfg.TargetRef), err)
		return
	}

	// Create the isolated config for the Target run
	cfgTarget := cfg.CloneWithTimeWindow(targetStartTime, targetEndTime)

	// 2a. Run file analysis for the Target state
	targetFileMetrics, err := analyzeAllFiles(cfgTarget, client)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Target File Analysis failed for ref %s", cfg.TargetRef))
		return
	}

	// 2b. Aggregate file metrics into folder results for the Target state
	targetFolderResults := aggregateAndScoreFolders(cfgTarget, targetFileMetrics)

	// --- C. Compute Delta and Output Results ---

	// Sort both sets of folder results by path for stable comparison
	sort.Slice(baseFolderResults, func(i, j int) bool {
		return baseFolderResults[i].Path < baseFolderResults[j].Path
	})
	sort.Slice(targetFolderResults, func(i, j int) bool {
		return targetFolderResults[i].Path < targetFolderResults[j].Path
	})

	// Compute the delta comparison for folders
	comparisonResults := compareFolderMetrics(baseFolderResults, targetFolderResults, cfg.ResultLimit)

	// Output the final comparison table
	internal.PrintComparisonResults(comparisonResults, cfg)
}
