package core

import (
	"math"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// ComparableResult represents a comparable result with path, score, commits, and churn.
type ComparableResult interface {
	GetPath() string
	GetScore() float64
	GetCommits() int
	GetChurn() int
	GetOwners() []string
}

// DeltaExtractor extracts additional deltas for comparison details.
type DeltaExtractor[T ComparableResult] func(base, target T) (deltaLOC, deltaContrib int)

// compareResults is a generic function that compares two sets of results.
func compareResults[T ComparableResult](baseResults, targetResults []T, limit int, mode string, extractDeltas DeltaExtractor[T]) schema.ComparisonResult {
	baseMap := make(map[string]T, len(baseResults))
	targetMap := make(map[string]T, len(targetResults))
	allPaths := make(map[string]struct{})

	// 1. Populate maps and collect all paths
	for _, r := range baseResults {
		baseMap[r.GetPath()] = r
		allPaths[r.GetPath()] = struct{}{}
	}
	for _, r := range targetResults {
		targetMap[r.GetPath()] = r
		allPaths[r.GetPath()] = struct{}{}
	}

	comparisonResults := make([]schema.ComparisonDetails, 0, len(allPaths))

	// Initialize summary accumulators
	var netScoreDelta float64
	var netChurnDelta int
	var totalNewFiles, totalInactiveFiles, totalModifiedFiles int
	var totalOwnershipChanges int

	// 2. Compare all paths
	for path := range allPaths {
		baseR, baseExists := baseMap[path]
		targetR, targetExists := targetMap[path]

		// Get scores (default to 0 if not exists)
		baseScore := 0.0
		if baseExists {
			baseScore = baseR.GetScore()
		}
		targetScore := 0.0
		if targetExists {
			targetScore = targetR.GetScore()
		}

		// Calculate deltas
		deltaScore := targetScore - baseScore
		deltaCommits := 0
		deltaChurn := 0
		if baseExists && targetExists {
			deltaCommits = targetR.GetCommits() - baseR.GetCommits()
			deltaChurn = targetR.GetChurn() - baseR.GetChurn()
		}

		// Extract additional deltas
		deltaLOC, deltaContrib := 0, 0
		if extractDeltas != nil && baseExists && targetExists {
			deltaLOC, deltaContrib = extractDeltas(baseR, targetR)
		}

		// Accumulate summary
		netScoreDelta += deltaScore
		netChurnDelta += deltaChurn

		// Determine status
		status := determineStatus(baseExists, targetExists)
		switch status {
		case schema.NewStatus:
			totalNewFiles++
		case schema.ActiveStatus:
			totalModifiedFiles++
		case schema.InactiveStatus:
			totalInactiveFiles++
		}

		// Get owners (default to empty if not exists)
		var beforeOwners []string
		if baseExists {
			beforeOwners = baseR.GetOwners()
		}
		var afterOwners []string
		if targetExists {
			afterOwners = targetR.GetOwners()
		}

		// Check for ownership change (compare original unabbreviated owners)
		if baseExists && targetExists && !schema.OwnersEqual(baseR.GetOwners(), targetR.GetOwners()) {
			totalOwnershipChanges++
		}

		// Only include results with significant score changes
		if math.Abs(deltaScore) > 0.01 {
			details := schema.ComparisonDetails{
				Path:         path,
				BeforeScore:  baseScore,
				AfterScore:   targetScore,
				Delta:        deltaScore,
				DeltaCommits: deltaCommits,
				DeltaChurn:   deltaChurn,
				Status:       status,
				BeforeOwners: beforeOwners,
				AfterOwners:  afterOwners,
				Mode:         schema.ScoringMode(mode),
			}

			// Add file-specific deltas if applicable
			if deltaLOC != 0 || deltaContrib != 0 {
				details.FileComparison = &schema.FileComparison{
					DeltaLOC:     deltaLOC,
					DeltaContrib: deltaContrib,
				}
			}

			comparisonResults = append(comparisonResults, details)
		}
	}

	// Create summary
	summary := schema.ComparisonSummary{
		NetScoreDelta:         netScoreDelta,
		NetChurnDelta:         netChurnDelta,
		TotalNewFiles:         totalNewFiles,
		TotalInactiveFiles:    totalInactiveFiles,
		TotalModifiedFiles:    totalModifiedFiles,
		TotalOwnershipChanges: totalOwnershipChanges,
	}

	// Sort results
	sortComparisonResults(comparisonResults)

	// Apply limit
	if limit > 0 && len(comparisonResults) > limit {
		comparisonResults = comparisonResults[:limit]
	}

	return schema.ComparisonResult{Results: comparisonResults, Summary: summary}
}

// determineStatus returns the status based on existence in base and target.
func determineStatus(baseExists, targetExists bool) schema.Status {
	switch {
	case !baseExists && targetExists:
		return schema.NewStatus
	case baseExists && targetExists:
		return schema.ActiveStatus
	case baseExists: // Target does not exist in this case
		return schema.InactiveStatus
	default:
		return schema.UnknownStatus
	}
}

// sortComparisonResults sorts comparison results by absolute delta, then delta sign, then path.
func sortComparisonResults(results []schema.ComparisonDetails) {
	sort.Slice(results, func(i, j int) bool {
		a := results[i]
		b := results[j]

		// Primary: Absolute delta (descending)
		absA := math.Abs(a.Delta)
		absB := math.Abs(b.Delta)
		if absA != absB {
			return absA > absB
		}

		// Secondary: Delta sign (positive before negative)
		if a.Delta != b.Delta {
			return a.Delta > b.Delta
		}

		// Tertiary: Path (ascending)
		return strings.Compare(a.Path, b.Path) < 0
	})
}

// compareFileResults matches metrics from the base run against the comparison run
// and computes the difference (delta) for key metrics like Score.
func compareFileResults(baseResults, targetResults []schema.FileResult, limit int, mode string) schema.ComparisonResult {
	return compareResults(baseResults, targetResults, limit, mode, func(base, target schema.FileResult) (int, int) {
		deltaLOC := target.LinesOfCode - base.LinesOfCode
		deltaContrib := target.UniqueContributors - base.UniqueContributors
		return deltaLOC, deltaContrib
	})
}

// compareFolderMetrics matches metrics from the base run against the target run
// and computes the difference (delta) for the Score metric.
func compareFolderMetrics(baseResults, targetResults []schema.FolderResult, limit int, mode string) schema.ComparisonResult {
	return compareResults(baseResults, targetResults, limit, mode, nil) // Folders don't have LOC/contrib deltas
}
