package outwriter

import (
	"math"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// getDisplayNameForMode returns the display name for a given mode name.
func getDisplayNameForMode(modeName string) string {
	switch modeName {
	case "hot":
		return "Hot"
	case "risk":
		return "Risk"
	case "complexity":
		return "Complexity"
	case "stale":
		return "Stale"
	default:
		return strings.ToUpper(modeName)
	}
}

// getDisplayWeightsForMode returns the weights to display for a given scoring mode.
// Uses active weights if available, otherwise falls back to defaults.
func getDisplayWeightsForMode(mode schema.ScoringMode, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64) map[string]float64 {
	// Start with default weights
	defaultWeights := schema.GetDefaultWeights(mode)

	// Convert BreakdownKey map to string map for backward compatibility
	weights := make(map[string]float64)
	for k, v := range defaultWeights {
		weights[string(k)] = v
	}

	// Override with active weights if provided
	if activeWeights != nil {
		if modeWeights, ok := activeWeights[mode]; ok {
			// Only override weights that are actually customized
			for k, v := range modeWeights {
				weights[string(k)] = v
			}
		}
	}

	return weights
}

// metricBreakdown holds a key-value pair from the Breakdown map representing a metric's contribution.
type metricBreakdown struct {
	Name  string  // e.g., "commits", "churn", "size"
	Value float64 // The percentage contribution to the score
}

const (
	metricContribMinimum = 0.5
	topNMetrics          = 3
)

// formatTopMetricBreakdown computes the top 3 metric components that contribute to the final score.
func formatTopMetricBreakdown(f *schema.FileResult) string {
	var metrics []metricBreakdown

	// 1. Filter and Convert Map to Slice
	for k, v := range f.ModeBreakdown {
		// Only include meaningful metrics
		if v >= metricContribMinimum {
			metrics = append(metrics, metricBreakdown{
				Name:  string(k),
				Value: v, // This is the percentage contribution
			})
		}
	}

	if len(metrics) == 0 {
		return "Not applicable"
	}

	// 2. Sort the Slice by Value (Contribution %) in Descending Order
	sort.Slice(metrics, func(i, j int) bool {
		return math.Abs(metrics[i].Value) > math.Abs(metrics[j].Value)
	})

	// 3. Limit to Top 3 and Format the Output
	var parts []string
	limit := min(len(metrics), topNMetrics)

	for i := range limit {
		m := metrics[i]
		parts = append(parts, m.Name)
	}

	if len(parts) == 0 {
		return "No meaningful contributors"
	}
	return strings.Join(parts, " > ")
}

// buildMetricsRenderModel constructs the complete render model with all processed data.
func buildMetricsRenderModel(activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64) *schema.MetricsRenderModel {
	modes := []schema.MetricsMode{
		{
			Name:       "hot",
			Purpose:    "Activity hotspots - high recent activity & volatility",
			Factors:    []string{"Commits", "Churn", "Contributors", "Age", "Size"},
			FactorKeys: []string{string(schema.BreakdownCommits), string(schema.BreakdownChurn), string(schema.BreakdownContrib), string(schema.BreakdownAge), string(schema.BreakdownSize)},
		},
		{
			Name:       "risk",
			Purpose:    "Knowledge risk/bus factor - concentrated ownership",
			Factors:    []string{"InvContributors", "Gini", "Age", "Churn", "Commits", "LOC", "Size"},
			FactorKeys: []string{string(schema.BreakdownInvContrib), string(schema.BreakdownGini), string(schema.BreakdownAge), string(schema.BreakdownChurn), string(schema.BreakdownCommits), string(schema.BreakdownLOC), string(schema.BreakdownSize)},
		},
		{
			Name:       "complexity",
			Purpose:    "Technical debt - large, old files with high maintenance burden",
			Factors:    []string{"Age", "Churn", "Commits", "LOC", "LowRecent", "Size"},
			FactorKeys: []string{string(schema.BreakdownAge), string(schema.BreakdownChurn), string(schema.BreakdownCommits), string(schema.BreakdownLOC), string(schema.BreakdownLowRecent), string(schema.BreakdownSize)},
		},
		{
			Name:       "stale",
			Purpose:    "Maintenance debt - important files untouched recently",
			Factors:    []string{"InvRecent", "Age", "Size", "Commits", "Contributors"},
			FactorKeys: []string{string(schema.BreakdownInvRecent), string(schema.BreakdownAge), string(schema.BreakdownSize), string(schema.BreakdownCommits), string(schema.BreakdownContrib)},
		},
		{
			Name:       "roi",
			Purpose:    "ROI - priority for refactoring effort based on technical impact",
			Factors:    []string{"Churn", "LOC", "Gini", "Age"},
			FactorKeys: []string{string(schema.BreakdownChurn), string(schema.BreakdownLOC), string(schema.BreakdownGini), string(schema.BreakdownAge)},
		},
	}
	modesWithData := make([]schema.MetricsModeWithData, len(modes))

	for i, mode := range modes {
		weights := getDisplayWeightsForMode(schema.ScoringMode(mode.Name), activeWeights)
		formula := formatWeights(weights, mode.FactorKeys)

		modesWithData[i] = schema.MetricsModeWithData{
			MetricsMode: mode,
			Weights:     weights,
			Formula:     formula,
		}
	}

	return &schema.MetricsRenderModel{
		Title:       "Hotspot Scoring Modes",
		Description: "All scores = weighted sum of normalized factors",
		Modes:       modesWithData,
		SpecialRelationship: map[string]string{
			"description": "RISK Score = HOT Score / Ownership Diversity Factor",
			"note":        "(Factor ↓ when few contributors → RISK Score ↑)",
		},
	}
}
