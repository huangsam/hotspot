package core

import (
	"math"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// computeScore calculates a file's importance score (0-100) based on its metrics.
// Supports four core scoring modes:
// - hot: Activity hotspots (high commits, churn, contributors)
// - risk: Knowledge risk/bus factor (few contributors, high inequality)
// - complexity: Technical debt candidates (large, old, high total churn)
// - stale: Maintenance debt (important but untouched)
func computeScore(m *schema.FileMetrics, mode string) float64 {
	// DEFENSIVE CHECK: If the file has no content, its score should be 0.
	if m.SizeBytes == 0 {
		return 0.0
	}

	// Tunable maxima to normalize metrics.
	const (
		maxContrib = 20.0   // contributors beyond this saturate
		maxCommits = 500.0  // commits beyond this saturate
		maxSizeKB  = 500.0  // file size in KB beyond this saturate
		maxAgeDays = 3650.0 // ~10 years
		maxChurn   = 5000.0 // total added+deleted lines
		maxRecent  = 50.0   // 50 recent commits is high activity
	)

	clamp01 := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}

	// --- Normalized Metrics [0,1] ---
	nContrib := clamp01(float64(m.UniqueContributors) / maxContrib)
	nCommits := clamp01(float64(m.Commits) / maxCommits)
	nSize := clamp01((float64(m.SizeBytes) / 1024.0) / maxSizeKB)
	nAge := clamp01(math.Log1p(float64(m.AgeDays)) / math.Log1p(maxAgeDays))
	nChurn := clamp01(float64(m.Churn) / maxChurn)

	// Inverted Metrics
	nGiniRaw := clamp01(m.Gini)            // Gini (raw: high is bad)
	nInvContrib := clamp01(1.0 - nContrib) // Inverse Contributors (high is bad/risky)
	nRecentCommits := clamp01(float64(m.RecentCommits) / maxRecent)
	nInvRecentCommits := clamp01(1.0 - nRecentCommits) // Inverse Recent Activity (high is stale)

	// --------------------------------

	breakdown := make(map[string]float64)
	var raw float64

	switch strings.ToLower(mode) {
	case "risk":
		// Knowledge-risk focused scoring: prioritize concentration and bus-factor
		const (
			wInvContrib = 0.32 // Directly measures bus factor
			wGini       = 0.28 // Measures contribution inequality
			wAgeRisk    = 0.18
			wSizeRisk   = 0.12
			wChurnRisk  = 0.06
			wCommRisk   = 0.04
		)
		breakdown[schema.BreakdownInvContrib] = wInvContrib * nInvContrib
		breakdown[schema.BreakdownGini] = wGini * nGiniRaw
		breakdown[schema.BreakdownAge] = wAgeRisk * nAge
		breakdown[schema.BreakdownSize] = wSizeRisk * nSize
		breakdown[schema.BreakdownChurn] = wChurnRisk * nChurn
		breakdown[schema.BreakdownCommits] = wCommRisk * nCommits

	case "complexity":
		// Technical debt focus: large, old files with high total churn
		const (
			wSizeComplex  = 0.35 // The most fundamental measure of complexity
			wAgeComplex   = 0.25 // Older code often contains legacy complexity
			wChurnComplex = 0.25 // High churn suggests volatility and/or refactoring difficulty
			wCommComplex  = 0.10
			wContribLow   = 0.05
		)
		// Complexity should favor files that aren't being actively fixed (low recent commits)
		breakdown[schema.BreakdownSize] = wSizeComplex * nSize
		breakdown[schema.BreakdownAge] = wAgeComplex * nAge
		breakdown[schema.BreakdownChurn] = wChurnComplex * nChurn
		breakdown[schema.BreakdownCommits] = wCommComplex * nCommits
		breakdown[schema.BreakdownLowRecent] = wContribLow * nInvRecentCommits // low recent activity means complexity is "settled"

	case "stale":
		// Maintenance debt: important but haven't been touched recently
		const (
			wInvRecentStale = 0.35 // This is the definition of "stale" — a lack of recent commits
			wSizeStale      = 0.25 // A large file that goes untouched is a bigger debt than a small one.
			wAgeStale       = 0.20 // Older files have a higher chance of accumulating maintenance debt
			wCommitsStale   = 0.15
			wContribStale   = 0.05
		)
		breakdown[schema.BreakdownInvRecent] = wInvRecentStale * nInvRecentCommits
		breakdown[schema.BreakdownSize] = wSizeStale * nSize
		breakdown[schema.BreakdownAge] = wAgeStale * nAge
		breakdown[schema.BreakdownCommits] = wCommitsStale * nCommits
		breakdown[schema.BreakdownContrib] = wContribStale * nContrib

	default: // case "hot" (default)
		// Hotspot scoring: where activity and volatility are concentrated
		const (
			wCommits = 0.28 // Raw commit count is a great measure of activity
			wChurn   = 0.26 // Volatility (lines changed) is key to a "hotspot"
			wContrib = 0.18
			wSize    = 0.16
			wAge     = 0.08
			wGini    = 0.04
		)
		breakdown[schema.BreakdownCommits] = wCommits * nCommits
		breakdown[schema.BreakdownChurn] = wChurn * nChurn
		breakdown[schema.BreakdownContrib] = wContrib * nContrib
		breakdown[schema.BreakdownSize] = wSize * nSize
		breakdown[schema.BreakdownAge] = wAge * nAge
		// Note: nGini is 1.0 - m.Gini. Here we want low Gini to not contribute much.
		breakdown[schema.BreakdownGini] = wGini * (1.0 - nGiniRaw)
	}

	for _, value := range breakdown {
		raw += value
	}
	score := raw * 100.0

	// If risk mode and this looks like a test file, slightly reduce score since
	// tests often have narrow contributors and shouldn't be first-class risks.
	if strings.ToLower(mode) == "risk" {
		if strings.Contains(m.Path, "_test") || strings.HasSuffix(m.Path, "_test.go") {
			score *= 0.75
		}
	}

	// Save breakdown (scaled to percent contributions) in the metrics for explain mode.
	if m.Breakdown == nil {
		m.Breakdown = make(map[string]float64)
	}
	for k, v := range breakdown {
		m.Breakdown[k] = v * 100.0
	}

	return score
}

// gini calculates the Gini coefficient for a set of values.
// The Gini coefficient measures inequality in a distribution, ranging from 0 (perfect equality)
// to 1 (perfect inequality). It's used here to measure how evenly distributed commits are
// among contributors.
func gini(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)
	if mean == 0 {
		return 0
	}

	var diffSum float64
	for i := range n {
		for j := range n {
			diffSum += math.Abs(values[i] - values[j])
		}
	}

	g := diffSum / (2 * float64(n*n) * mean)
	return math.Min(math.Max(g, 0), 1) // clamp to [0,1]
}
