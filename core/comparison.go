package core

import (
	"math"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// compareFileMetrics matches metrics from the base run against the comparison run
// and computes the difference (delta) for key metrics like Score.
func compareFileMetrics(baseMetrics, compareMetrics []schema.FileMetrics, limit int) []schema.ComparisonMetrics {
	baseMap := make(map[string]schema.FileMetrics, len(baseMetrics))
	compareMap := make(map[string]schema.FileMetrics, len(compareMetrics))
	allPaths := make(map[string]struct{}) // Set to hold all unique paths

	// 1. Populate maps and the set of ALL paths
	for _, m := range baseMetrics {
		baseMap[m.Path] = m
		allPaths[m.Path] = struct{}{}
	}
	for _, m := range compareMetrics {
		compareMap[m.Path] = m
		allPaths[m.Path] = struct{}{}
	}

	comparisonResults := make([]schema.ComparisonMetrics, 0, len(allPaths))

	// 2. Iterate over ALL unique paths (Full Outer Join)
	for path := range allPaths {
		baseM, baseExists := baseMap[path]
		compM, compExists := compareMap[path]

		// Initialize default/zero metrics for non-existent files
		if !baseExists {
			baseM = schema.FileMetrics{} // Zero values (Score=0, Commits=0, Churn=0)
		}
		if !compExists {
			compM = schema.FileMetrics{} // Zero values
		}

		// 3. Calculate Delta and assemble the result
		deltaScore := compM.Score - baseM.Score
		deltaCommits := compM.Commits - baseM.Commits
		deltaChurn := compM.Churn - baseM.Churn

		// Only track and report files where the score actually changed significantly
		if math.Abs(deltaScore) > 0.01 {
			// Crucially, use the *actual* path for files that only exist in one set
			// For DELETED files: BaseScore > 0, CompScore = 0, Delta < 0
			// For NEW files: BaseScore = 0, CompScore > 0, Delta > 0
			comparisonResults = append(comparisonResults, schema.ComparisonMetrics{
				Path:         path,
				BaseScore:    baseM.Score,
				CompScore:    compM.Score,
				Delta:        deltaScore,
				DeltaCommits: deltaCommits,
				DeltaChurn:   deltaChurn,
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

	if len(comparisonResults) > 0 && limit > 0 {
		newLimit := min(len(comparisonResults), limit)
		return comparisonResults[:newLimit]
	}
	return comparisonResults
}
