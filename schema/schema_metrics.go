package schema

import (
	"fmt"
	"strings"
)

// MetricsMode represents a scoring mode for display purposes.
type MetricsMode struct {
	Name       string             `json:"name"`
	Purpose    string             `json:"purpose"`
	Factors    []string           `json:"factors"`
	FactorKeys []string           `json:"factor_keys,omitempty"` // Only used for JSON output
	Weights    map[string]float64 `json:"weights,omitempty"`     // Only used for JSON output
	Formula    string             `json:"formula,omitempty"`     // Only used for JSON output
}

// MetricsRenderModel contains all processed data needed for displaying metrics definitions.
type MetricsRenderModel struct {
	Title               string                `json:"title"`
	Description         string                `json:"description"`
	Modes               []MetricsModeWithData `json:"modes"`
	SpecialRelationship map[string]string     `json:"special_relationship"`
}

// MetricsModeWithData extends MetricsMode with computed weights and formula.
type MetricsModeWithData struct {
	MetricsMode
	Weights map[string]float64 `json:"weights"`
	Formula string             `json:"formula"`
}

// BuildMetricsRenderModel constructs the complete render model with all processed data.
func BuildMetricsRenderModel(activeWeights map[ScoringMode]map[BreakdownKey]float64) *MetricsRenderModel {
	var modes []MetricsMode
	for _, mode := range AllScoringModes {
		purpose, factors, factorKeys := GetScoringModeMetadata(mode)
		modes = append(modes, MetricsMode{
			Name:       string(mode),
			Purpose:    purpose,
			Factors:    factors,
			FactorKeys: factorKeys,
		})
	}
	modesWithData := make([]MetricsModeWithData, len(modes))

	for i, mode := range modes {
		weights := GetDisplayWeightsForMode(ScoringMode(mode.Name), activeWeights)
		formula := FormatWeights(weights, mode.FactorKeys)

		modesWithData[i] = MetricsModeWithData{
			MetricsMode: mode,
			Weights:     weights,
			Formula:     formula,
		}
	}

	return &MetricsRenderModel{
		Title:       "Hotspot Scoring Modes",
		Description: "All scores = weighted sum of normalized factors",
		Modes:       modesWithData,
		SpecialRelationship: map[string]string{
			"description": "RISK Score = Weighted balance of Ownership concentration and Staleness",
			"note":        "(Focuses on Gini Index, Contributor diversity, and Knowledge decay)",
		},
	}
}

// GetDisplayNameForMode returns the display name for a given mode name.
func GetDisplayNameForMode(modeName string) string {
	switch modeName {
	case "hot":
		return "Hot"
	case "risk":
		return "Risk"
	case "complexity":
		return "Complexity"
	default:
		return strings.ToUpper(modeName)
	}
}

// GetDisplayWeightsForMode returns the weights to display for a given scoring mode.
// Uses active weights if available, otherwise falls back to defaults.
func GetDisplayWeightsForMode(mode ScoringMode, activeWeights map[ScoringMode]map[BreakdownKey]float64) map[string]float64 {
	// Start with default weights
	defaultWeights := GetDefaultWeights(mode)

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

// FormatWeights formats weights for display in formulas.
func FormatWeights(weights map[string]float64, factorKeys []string) string {
	var parts []string
	for _, key := range factorKeys {
		if weight, ok := weights[key]; ok && weight > 0 {
			factorName := strings.ToLower(strings.TrimPrefix(key, "breakdown_"))
			parts = append(parts, fmt.Sprintf("%.2f*%s", weight, factorName))
		}
	}
	return strings.Join(parts, "+")
}
