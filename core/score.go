package core

import (
	"math"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// computeScore calculates a file's importance score (0-100) based on its metrics.
// Supports multiple scoring modes:
// - hot: Activity hotspots (high commits, churn, contributors)
// - risk: Knowledge risk/bus factor (few contributors, high inequality)
// - complexity: Technical debt candidates (large, old, high churn)
// - stale: Maintenance debt (important but untouched)
// - onboarding: Files new developers should learn
// - ownership: Healthy ownership patterns
// - security: Security-critical file detection
func computeScore(m *schema.FileMetrics, mode string) float64 {
	// Tunable maxima to normalize metrics. These are conservative defaults
	// chosen to avoid a few outliers dominating the score. Consider making
	// these flags or schema.Config values in the future.
	const (
		maxContrib = 20.0   // contributors beyond this saturate
		maxCommits = 500.0  // commits beyond this saturate
		maxSizeKB  = 500.0  // file size in KB beyond this saturate
		maxAgeDays = 3650.0 // ~10 years
		maxChurn   = 5000.0 // total added+deleted lines
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

	// Normalize each metric into [0,1]
	nContrib := clamp01(float64(m.UniqueContributors) / maxContrib)
	nCommits := clamp01(float64(m.Commits) / maxCommits)
	nSize := clamp01((float64(m.SizeBytes) / 1024.0) / maxSizeKB)
	// Age is tricky: very old files shouldn't always be treated as critical.
	// We use a log-like scaling (but simple) to give diminishing returns.
	nAge := clamp01(math.Log1p(float64(m.AgeDays)) / math.Log1p(maxAgeDays))
	nChurn := clamp01(float64(m.Churn) / maxChurn)
	// Gini: lower is healthier; invert and clamp
	nGini := clamp01(1.0 - m.Gini)

	// For stale mode, we need inverse of recent activity
	nRecentCommits := clamp01(float64(m.RecentCommits) / 50.0) // assume 50 recent commits is high activity

	// Prepare breakdown map to return component contributions
	breakdown := make(map[string]float64)
	var raw float64

	switch strings.ToLower(mode) {
	case "risk":
		// Knowledge-risk focused scoring: prioritize concentration and bus-factor
		invContrib := clamp01(1.0 - (float64(m.UniqueContributors) / maxContrib))
		giniRaw := clamp01(m.Gini)

		const (
			wInvContrib = 0.32
			wGini       = 0.28
			wAgeRisk    = 0.18
			wSizeRisk   = 0.12
			wChurnRisk  = 0.06
			wCommRisk   = 0.04
		)
		breakdown["inv_contrib"] = wInvContrib * invContrib
		breakdown["gini"] = wGini * giniRaw
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
			wContribLow   = 0.05 // prefer fewer contributors (concentrated complexity)
		)
		// Invert recent activity - we want files that aren't being actively worked on
		invRecentCommits := clamp01(1.0 - nRecentCommits)
		breakdown["size"] = wSizeComplex * nSize
		breakdown["age"] = wAgeComplex * nAge
		breakdown["churn"] = wChurnComplex * nChurn
		breakdown["commits"] = wCommComplex * nCommits
		breakdown["low_recent"] = wContribLow * invRecentCommits
		raw = breakdown["size"] + breakdown["age"] + breakdown["churn"] + breakdown["commits"] + breakdown["low_recent"]

	case "stale":
		// Maintenance debt: important but haven't been touched recently
		invRecentCommits := clamp01(1.0 - nRecentCommits)
		const (
			wAgeStale       = 0.30
			wSizeStale      = 0.25
			wInvRecentStale = 0.25 // penalize recent activity
			wCommitsStale   = 0.15 // historically important
			wContribStale   = 0.05
		)
		breakdown["age"] = wAgeStale * nAge
		breakdown["size"] = wSizeStale * nSize
		breakdown["inv_recent"] = wInvRecentStale * invRecentCommits
		breakdown["commits"] = wCommitsStale * nCommits
		breakdown["contrib"] = wContribStale * nContrib
		raw = breakdown["age"] + breakdown["size"] + breakdown["inv_recent"] + breakdown["commits"] + breakdown["contrib"]

	case "onboarding":
		// Files new developers should learn: active, well-maintained, moderate complexity
		const (
			wContribOnboard = 0.30
			wCommitsOnboard = 0.25
			wSizeOnboard    = 0.20 // prefer moderate size (not too large)
			wAgeOnboard     = 0.15 // some maturity is good
			wGiniOnboard    = 0.10 // even distribution is healthy
		)
		// For size, prefer moderate - penalize both very small and very large
		moderateSize := 1.0 - math.Abs(nSize-0.4) // peak at 40% of max
		breakdown["contrib"] = wContribOnboard * nContrib
		breakdown["commits"] = wCommitsOnboard * nCommits
		breakdown["size"] = wSizeOnboard * clamp01(moderateSize)
		breakdown["age"] = wAgeOnboard * nAge
		breakdown["gini"] = wGiniOnboard * nGini
		raw = breakdown["contrib"] + breakdown["commits"] + breakdown["size"] + breakdown["age"] + breakdown["gini"]

	case "ownership":
		// Healthy ownership patterns: even distribution, steady activity
		const (
			wGiniOwnership    = 0.35 // reward even distribution
			wContribOwnership = 0.25 // moderate number of contributors
			wCommitsOwnership = 0.20
			wChurnOwnership   = 0.10 // steady but not chaotic
			wAgeOwnership     = 0.10
		)
		// Prefer moderate contributors (not too few, not too many)
		moderateContrib := 1.0 - math.Abs(nContrib-0.5)
		breakdown["gini"] = wGiniOwnership * nGini
		breakdown["contrib"] = wContribOwnership * clamp01(moderateContrib)
		breakdown["commits"] = wCommitsOwnership * nCommits
		breakdown["churn"] = wChurnOwnership * nChurn
		breakdown["age"] = wAgeOwnership * nAge
		raw = breakdown["gini"] + breakdown["contrib"] + breakdown["commits"] + breakdown["churn"] + breakdown["age"]

	case "security":
		// Security-critical file detection
		invContrib := clamp01(1.0 - (float64(m.UniqueContributors) / maxContrib))
		giniRaw := clamp01(m.Gini)

		// Detect security-related keywords in path
		securityBoost := 0.0
		lowerPath := strings.ToLower(m.Path)
		securityKeywords := []string{"auth", "password", "token", "secret", "crypto", "security", "login", "session", "oauth", "jwt", "credential", "permission", "acl", "rbac"}
		for _, keyword := range securityKeywords {
			if strings.Contains(lowerPath, keyword) {
				securityBoost = 1.0
				break
			}
		}

		const (
			wSecurityBoost = 0.30 // path-based detection
			wAgeSecurity   = 0.25 // old security code = more exposure
			wInvContribSec = 0.20 // fewer eyes = risk
			wGiniSecurity  = 0.15 // concentration risk
			wSizeSecurity  = 0.10
		)
		breakdown["sec_boost"] = wSecurityBoost * securityBoost
		breakdown["age"] = wAgeSecurity * nAge
		breakdown["inv_contrib"] = wInvContribSec * invContrib
		breakdown["gini"] = wGiniSecurity * giniRaw
		breakdown["size"] = wSizeSecurity * nSize
		raw = breakdown["sec_boost"] + breakdown["age"] + breakdown["inv_contrib"] + breakdown["gini"] + breakdown["size"]

	default:
		// Hotspot scoring (default): where activity and volatility are concentrated
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
		breakdown["gini"] = wGini * nGini
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
