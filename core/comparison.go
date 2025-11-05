package core

import (
	"math"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// compareFileMetrics matches metrics from the base run against the comparison run
// and computes the difference (delta) for key metrics like Score.
func compareFileMetrics(baseMetrics, compareMetrics []schema.FileMetrics) []schema.ComparisonMetrics {
	// 1. Create a quick-lookup map for the base metrics, keyed by file path.
	baseMap := make(map[string]schema.FileMetrics, len(baseMetrics))
	for _, m := range baseMetrics {
		baseMap[m.Path] = m
	}

	comparisonResults := make([]schema.ComparisonMetrics, 0)

	// 2. Iterate over the Comparison (New) metrics
	for _, compM := range compareMetrics {
		baseM, ok := baseMap[compM.Path]
		if !ok {
			continue
		}

		// 3. Calculate Delta and assemble the result
		deltaScore := compM.Score - baseM.Score
		deltaCommits := compM.Commits - baseM.Commits
		deltaChurn := compM.Churn - baseM.Churn

		// Only track and report files where the score actually changed significantly
		// A small epsilon check (e.g., math.Abs(deltaScore) > 0.01) is ideal,
		// but for now, checking against zero works if your scores have limited precision.
		if math.Abs(deltaScore) > 0.01 {
			comparisonResults = append(comparisonResults, schema.ComparisonMetrics{
				Path:         compM.Path,
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

	return comparisonResults
}
