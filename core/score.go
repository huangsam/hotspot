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
			wInvContrib = 0.32
			wGini       = 0.28
			wAgeRisk    = 0.18
			wSizeRisk   = 0.12
			wChurnRisk  = 0.06
			wCommRisk   = 0.04
		)
		breakdown["inv_contrib"] = wInvContrib * nInvContrib
		breakdown["gini"] = wGini * nGiniRaw
		breakdown["age"] = wAgeRisk * nAge
		breakdown["size"] = wSizeRisk * nSize
		breakdown["churn"] = wChurnRisk * nChurn
		breakdown["commits"] = wCommRisk * nCommits
		raw = breakdown["inv_contrib"] + breakdown["gini"] + breakdown["age"] + breakdown["size"] + breakdown["churn"] + breakdown["commits"]

	case "complexity":
		// Technical debt focus: large, old files with high total churn
		const (
			wSizeComplex  = 0.35
			wAgeComplex   = 0.25
			wChurnComplex = 0.25
			wCommComplex  = 0.10
			wContribLow   = 0.05
		)
		// Complexity should favor files that aren't being actively fixed (low recent commits)
		breakdown["size"] = wSizeComplex * nSize
		breakdown["age"] = wAgeComplex * nAge
		breakdown["churn"] = wChurnComplex * nChurn
		breakdown["commits"] = wCommComplex * nCommits
		breakdown["low_recent"] = wContribLow * nInvRecentCommits // low recent activity means complexity is "settled"
		raw = breakdown["size"] + breakdown["age"] + breakdown["churn"] + breakdown["commits"] + breakdown["low_recent"]

	case "stale":
		// Maintenance debt: important but haven't been touched recently
		const (
			wAgeStale       = 0.30
			wSizeStale      = 0.25
			wInvRecentStale = 0.25 // primary driver: lack of recent activity
			wCommitsStale   = 0.15 // historically important
			wContribStale   = 0.05
		)
		breakdown["age"] = wAgeStale * nAge
		breakdown["size"] = wSizeStale * nSize
		breakdown["inv_recent"] = wInvRecentStale * nInvRecentCommits
		breakdown["commits"] = wCommitsStale * nCommits
		breakdown["contrib"] = wContribStale * nContrib
		raw = breakdown["age"] + breakdown["size"] + breakdown["inv_recent"] + breakdown["commits"] + breakdown["contrib"]

	default: // case "hot" (default)
		// Hotspot scoring: where activity and volatility are concentrated
		const (
			wContrib = 0.18
			wCommits = 0.28 // many code changes
			wSize    = 0.16
			wAge     = 0.08
			wChurn   = 0.26 // plenty of churn in the code
			wGini    = 0.04
		)
		breakdown["contrib"] = wContrib * nContrib
		breakdown["commits"] = wCommits * nCommits
		breakdown["size"] = wSize * nSize
		breakdown["age"] = wAge * nAge
		breakdown["churn"] = wChurn * nChurn
		// Note: nGini is 1.0 - m.Gini. Here we want low Gini to not contribute much.
		breakdown["gini"] = wGini * (1.0 - nGiniRaw)
		raw = breakdown["contrib"] + breakdown["commits"] + breakdown["size"] + breakdown["age"] + breakdown["churn"] + breakdown["gini"]
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
