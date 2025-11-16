package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// getDisplayNameForMode returns the display name with emoji for a given mode name.
func getDisplayNameForMode(modeName string) string {
	switch modeName {
	case "hot":
		return "🔥 HOT"
	case "risk":
		return "⚠️  RISK"
	case "complexity":
		return "🧩 COMPLEXITY"
	case "stale":
		return "🕰️  STALE"
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

// formatWeights formats weights for display in formulas.
func formatWeights(weights map[string]float64, factorKeys []string) string {
	var parts []string
	for _, key := range factorKeys {
		if weight, ok := weights[key]; ok && weight > 0 {
			factorName := strings.ToLower(strings.TrimPrefix(key, "breakdown_"))
			parts = append(parts, fmt.Sprintf("%.2f*%s", weight, factorName))
		}
	}
	return strings.Join(parts, "+")
}

// PrintMetricsDefinitions displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func PrintMetricsDefinitions(activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, cfg *contract.Config) error {
	// Build the complete render model with all processed data
	renderModel := buildMetricsRenderModel(activeWeights)

	switch cfg.Output {
	case schema.JSONOut:
		return printMetricsJSON(renderModel, cfg)
	case schema.CSVOut:
		return printMetricsCSV(renderModel, cfg)
	default:
		return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
			return printMetricsText(w, renderModel, cfg)
		}, "Wrote text")
	}
}

// printMetricsText displays metrics in human-readable text format.
func printMetricsText(w io.Writer, renderModel *schema.MetricsRenderModel, _ *contract.Config) error {
	if _, err := fmt.Fprintf(w, "🔥 Hotspot Scoring Modes\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "========================\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", renderModel.Description); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}

	for _, mode := range renderModel.Modes {
		// Add emoji prefix for display
		displayName := getDisplayNameForMode(mode.Name)
		if _, err := fmt.Fprintf(w, "%s: %s\n", displayName, mode.Purpose); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "   Factors: %s\n", strings.Join(mode.Factors, ", ")); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "   Formula: Score = %s\n", mode.Formula); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "\n"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "🔗 Special Relationship\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", renderModel.SpecialRelationship["description"]); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", renderModel.SpecialRelationship["note"]); err != nil {
		return err
	}

	return nil
}

// printMetricsJSON displays metrics in JSON format.
func printMetricsJSON(renderModel *schema.MetricsRenderModel, cfg *contract.Config) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		return writeJSONMetrics(w, renderModel)
	}, "Wrote JSON")
}

// printMetricsCSV displays metrics in CSV format.
func printMetricsCSV(renderModel *schema.MetricsRenderModel, cfg *contract.Config) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		writer := csv.NewWriter(w)
		defer writer.Flush()
		return writeCSVMetrics(writer, renderModel)
	}, "Wrote CSV")
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
