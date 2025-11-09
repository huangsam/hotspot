package internal

import (
	"fmt"
	"maps"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// getDisplayWeightsForMode returns the weights to display for a given scoring mode.
// Uses active weights if available, otherwise falls back to defaults.
func getDisplayWeightsForMode(mode string, activeWeights map[string]map[string]float64) map[string]float64 {
	// Start with default weights
	scoringMode := schema.ScoringMode(mode)
	defaultWeights := schema.GetDefaultWeights(scoringMode)

	// Convert BreakdownKey map to string map for backward compatibility
	weights := make(map[string]float64)
	for k, v := range defaultWeights {
		weights[string(k)] = v
	}

	// Override with active weights if provided
	if activeWeights != nil {
		if modeWeights, ok := activeWeights[mode]; ok {
			// Only override weights that are actually customized
			maps.Copy(weights, modeWeights)
		}
	}

	return weights
}

// formatWeights formats weights for display in formulas
func formatWeights(weights map[string]float64, factorKeys []string) string {
	var parts []string
	for _, key := range factorKeys {
		if weight, ok := weights[key]; ok && weight > 0 {
			factorName := strings.ToLower(strings.TrimPrefix(key, "breakdown_"))
			parts = append(parts, fmt.Sprintf("%.2f*%s", weight, factorName))
		}
	}
	return strings.Join(parts, " + ")
}

// PrintMetricsDefinitions displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func PrintMetricsDefinitions(activeWeights map[string]map[string]float64) error {
	fmt.Println("🔥 Hotspot Scoring Modes")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("All scores = weighted sum of normalized factors")
	fmt.Println()

	modes := []struct {
		name       string
		purpose    string
		factors    []string
		factorKeys []string
	}{
		{
			name:       "🔥 HOT",
			purpose:    "Activity hotspots - high recent activity & volatility",
			factors:    []string{"Commits", "Churn", "Contributors", "Age", "Size"},
			factorKeys: []string{string(schema.BreakdownCommits), string(schema.BreakdownChurn), string(schema.BreakdownContrib), string(schema.BreakdownAge), string(schema.BreakdownSize)},
		},
		{
			name:       "⚠️  RISK",
			purpose:    "Knowledge risk/bus factor - concentrated ownership",
			factors:    []string{"InvContributors", "Gini", "Age", "Churn", "Commits", "LOC", "Size"},
			factorKeys: []string{string(schema.BreakdownInvContrib), string(schema.BreakdownGini), string(schema.BreakdownAge), string(schema.BreakdownChurn), string(schema.BreakdownCommits), string(schema.BreakdownLOC), string(schema.BreakdownSize)},
		},
		{
			name:       "🧩 COMPLEXITY",
			purpose:    "Technical debt - large, old files with high maintenance burden",
			factors:    []string{"Age", "Churn", "Commits", "LOC", "LowRecent", "Size"},
			factorKeys: []string{string(schema.BreakdownAge), string(schema.BreakdownChurn), string(schema.BreakdownCommits), string(schema.BreakdownLOC), string(schema.BreakdownLowRecent), string(schema.BreakdownSize)},
		},
		{
			name:       "🕰️  STALE",
			purpose:    "Maintenance debt - important files untouched recently",
			factors:    []string{"InvRecent", "Age", "Size", "Commits", "Contributors"},
			factorKeys: []string{string(schema.BreakdownInvRecent), string(schema.BreakdownAge), string(schema.BreakdownSize), string(schema.BreakdownCommits), string(schema.BreakdownContrib)},
		},
	}

	for _, mode := range modes {
		fmt.Printf("%s: %s\n", mode.name, mode.purpose)
		fmt.Printf("   Factors: %s\n", strings.Join(mode.factors, ", "))

		// Extract mode name from emoji prefix
		modeName := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(mode.name, "🔥 "), "⚠️  "), "🧩 "), "🕰️  ")))
		weights := getDisplayWeightsForMode(modeName, activeWeights)
		formula := formatWeights(weights, mode.factorKeys)
		fmt.Printf("   Formula: Score = %s\n", formula)
		fmt.Println()
	}

	fmt.Println("🔗 Special Relationship")
	fmt.Println("RISK Score = HOT Score / Ownership Diversity Factor")
	fmt.Println("(Factor ↓ when few contributors → RISK Score ↑)")

	return nil
}
