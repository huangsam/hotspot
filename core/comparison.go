package core

import (
	"math"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// compareFileResults matches metrics from the base run against the comparison run
// and computes the difference (delta) for key metrics like Score.
func compareFileResults(baseResults, targetResults []schema.FileResult, limit int) []schema.ComparisonResult {
	baseMap := make(map[string]schema.FileResult, len(baseResults))
	targetMap := make(map[string]schema.FileResult, len(targetResults))
	allPaths := make(map[string]struct{}) // Set to hold all unique paths

	// 1. Populate maps and the set of ALL paths
	for _, m := range baseResults {
		baseMap[m.Path] = m
		allPaths[m.Path] = struct{}{}
	}
	for _, m := range targetResults {
		targetMap[m.Path] = m
		allPaths[m.Path] = struct{}{}
	}

	comparisonResults := make([]schema.ComparisonResult, 0, len(allPaths))

	// 2. Iterate over ALL unique paths (Full Outer Join)
	for path := range allPaths {
		baseM, baseExists := baseMap[path]
		targetM, targetExists := targetMap[path]

		// Initialize default/zero metrics for non-existent files
		if !baseExists {
			baseM = schema.FileResult{} // Zero values (Score=0, Commits=0, Churn=0)
		}
		if !targetExists {
			targetM = schema.FileResult{} // Zero values
		}

		// 3. Calculate Delta and assemble the result
		deltaScore := targetM.Score - baseM.Score
		deltaCommits := targetM.Commits - baseM.Commits
		deltaChurn := targetM.Churn - baseM.Churn
		deltaLOC := targetM.LinesOfCode - baseM.LinesOfCode
		deltaContrib := targetM.UniqueContributors - baseM.UniqueContributors

		// Determine status based on existence in each analysis
		var status string
		switch {
		case !baseExists && targetExists:
			status = schema.NewStatus
		case baseExists && targetExists:
			status = schema.ActiveStatus
		case baseExists && !targetExists:
			status = schema.InactiveStatus
		default:
			status = schema.UnknownStatus
		}

		// Only track and report files where the score actually changed significantly
		if math.Abs(deltaScore) > 0.01 {
			// Crucially, use the *actual* path for files that only exist in one set
			// For DELETED files: BaseScore > 0, CompScore = 0, Delta < 0
			// For NEW files: BaseScore = 0, CompScore > 0, Delta > 0
			file := &schema.FileComparison{DeltaLOC: deltaLOC, DeltaContrib: deltaContrib}
			comparisonResults = append(comparisonResults, schema.ComparisonResult{
				Path:           path,
				BeforeScore:    baseM.Score,
				AfterScore:     targetM.Score,
				Delta:          deltaScore,
				DeltaCommits:   deltaCommits,
				DeltaChurn:     deltaChurn,
				FileComparison: file,
				Status:         status,
			})
		}
	}

	// 4. Implement a deterministic three-level sort on the filtered list.
	// This stabilizes the output when multiple files have the same absolute delta.
	sort.Slice(comparisonResults, func(i, j int) bool {
		a := comparisonResults[i]
		b := comparisonResults[j]

		// --- 1. Primary Key: Absolute Delta (Descending) ---
		absA := math.Abs(a.Delta)
		absB := math.Abs(b.Delta)

		if absA != absB {
			// Sort by the one with the largest absolute change first (Descending)
			return absA > absB
		}

		// --- 2. Secondary Key (Tie-breaker): Delta Sign (Positive over Negative) ---
		// This ensures increasing risk (+0.25) ranks higher than decreasing risk (-0.25).
		if a.Delta != b.Delta {
			return a.Delta > b.Delta // Sorts positive deltas before negative deltas
		}

		// --- 3. Tertiary Key (Final Tie-breaker): File Path (Ascending) ---
		// Guarantees an identical sort order for identical deltas and signs (e.g., two +0.25 deltas).
		return strings.Compare(a.Path, b.Path) < 0
	})

	if len(comparisonResults) > 0 && len(comparisonResults) > limit {
		return comparisonResults[:limit]
	}
	return comparisonResults
}

// compareFolderMetrics matches metrics from the base run against the target run
// and computes the difference (delta) for the Score metric.
func compareFolderMetrics(baseResults, targetResults []schema.FolderResult, limit int) []schema.ComparisonResult {
	baseMap := make(map[string]schema.FolderResult, len(baseResults))
	targetMap := make(map[string]schema.FolderResult, len(targetResults))
	allPaths := make(map[string]struct{}) // Set to hold all unique folder paths

	// 1. Populate maps and the set of ALL paths
	for _, m := range baseResults {
		baseMap[m.Path] = m
		allPaths[m.Path] = struct{}{}
	}
	for _, m := range targetResults {
		targetMap[m.Path] = m
		allPaths[m.Path] = struct{}{}
	}

	comparisonResults := make([]schema.ComparisonResult, 0, len(allPaths))

	// 2. Iterate over ALL unique paths (Full Outer Join)
	for path := range allPaths {
		baseM, baseExists := baseMap[path]
		targetM, targetExists := targetMap[path]

		// Determine status based on existence in each analysis
		var status string
		switch {
		case !baseExists && targetExists:
			status = schema.NewStatus
		case baseExists && targetExists:
			status = schema.ActiveStatus
		case baseExists && !targetExists:
			status = schema.InactiveStatus
		default:
			status = schema.UnknownStatus
		}

		baseScore := 0.0
		if baseExists {
			baseScore = baseM.Score
		}

		targetScore := 0.0
		if targetExists {
			targetScore = targetM.Score
		}

		// 3. Calculate Delta and assemble the result
		deltaScore := targetScore - baseScore
		deltaCommits := targetM.Commits - baseM.Commits
		deltaChurn := targetM.Churn - baseM.Churn

		// Only track and report folders where the score actually changed significantly.
		// Using a tolerance of 0.01 to match the file comparison logic.
		if math.Abs(deltaScore) > 0.01 {
			comparisonResults = append(comparisonResults, schema.ComparisonResult{
				Path:         path,
				BeforeScore:  baseScore,
				AfterScore:   targetScore,
				Delta:        deltaScore,
				Status:       status,
				DeltaCommits: deltaCommits,
				DeltaChurn:   deltaChurn,
			})
		}
	}

	// 4. Implement a deterministic three-level sort on the filtered list.
	sort.Slice(comparisonResults, func(i, j int) bool {
		a := comparisonResults[i]
		b := comparisonResults[j]

		// --- 1. Primary Key: Absolute Delta (Descending) ---
		absA := math.Abs(a.Delta)
		absB := math.Abs(b.Delta)

		if absA != absB {
			// Sort by the one with the largest absolute change first (Descending)
			return absA > absB
		}

		// --- 2. Secondary Key (Tie-breaker): Delta Sign (Positive over Negative) ---
		// Increasing risk (+DeltaScore) ranks higher than decreasing risk (-DeltaScore).
		if a.Delta != b.Delta {
			return a.Delta > b.Delta // Sorts positive deltas before negative deltas
		}

		// --- 3. Tertiary Key (Final Tie-breaker): Folder Path (Ascending) ---
		// Guarantees an identical sort order for identical deltas and signs.
		return a.Path < b.Path
	})

	if limit > 0 && len(comparisonResults) > limit {
		return comparisonResults[:limit]
	}
	return comparisonResults
}
