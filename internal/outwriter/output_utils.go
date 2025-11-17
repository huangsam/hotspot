package outwriter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"golang.org/x/term"
)

// writeWithFile handles the common pattern of opening a file, writing to it, and cleaning up.
// It accepts a writer function that takes an io.Writer and returns an error.
func writeWithFile(outputFile string, writer func(io.Writer) error, successMsg string) error {
	file, err := contract.SelectOutputFile(outputFile)
	if err != nil {
		return err
	}
	// Only close if it's not stdout
	if file != os.Stdout {
		defer func() { _ = file.Close() }()
	}

	if err := writer(file); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 %s to %s\n", successMsg, outputFile)
	}
	return nil
}

// writeJSON is a generic JSON encoder that handles indentation consistently.
func writeJSON(w io.Writer, data any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// writeCSVWithHeader handles the common pattern of creating a CSV writer,
// writing a header, and writing data rows.
func writeCSVWithHeader(w io.Writer, header []string, writeRows func(*csv.Writer) error) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	if err := writeRows(csvWriter); err != nil {
		return err
	}

	return nil
}

// createFormatters creates the common formatter closures used across multiple output types.
func createFormatters(precision int) (fmtFloat func(float64) string, intFmt string) {
	numFmt := "%.*f"
	intFmt = "%d"
	fmtFloat = func(v float64) string {
		return fmt.Sprintf(numFmt, precision, v)
	}
	return fmtFloat, intFmt
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
	for k, v := range f.Breakdown {
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
	// Metrics with the highest absolute percentage contribution come first.
	sort.Slice(metrics, func(i, j int) bool {
		// We compare the absolute value since some contributions might be negative
		// if the model is set up to penalize certain metrics.
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

// getMaxTablePathWidth calculates the maximum width for file paths in table output
// based on terminal width and table configuration.
func getMaxTablePathWidth(cfg *contract.Config) int {
	var termWidth int

	// Check for absolute width override from flag/env
	if cfg.Width > 0 {
		termWidth = cfg.Width
	}

	if termWidth == 0 { // Not set by override
		// Get terminal width
		detectedWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || detectedWidth <= 0 {
			// Fallback to conservative default if terminal size can't be detected
			termWidth = 80 // Conservative default for narrow terminals and CI
		} else {
			termWidth = detectedWidth
		}
	}

	// Reserve space for fixed columns with table formatting
	baseWidth := 25 // Rank + Score + Label with borders/padding

	// Add detail columns with formatting
	if cfg.Detail {
		baseWidth += 55 // All detail columns (Contrib + Commits + LOC + Churn + Age + Gini) with formatting
	}

	// Add explain column
	if cfg.Explain {
		baseWidth += 35 // Explain column with formatting
	}

	// Add owner column
	if cfg.Owner {
		baseWidth += 25 // Owner column with formatting
	}

	// Reserve generous space for table borders, separators, and padding
	baseWidth += 20

	// Calculate available space for path
	available := termWidth - baseWidth
	if available < 15 {
		// Minimum reasonable path width
		return 15
	}
	if available > 70 {
		// Maximum path width to prevent overly long paths
		return 70
	}
	return available
}

// formatOwnershipDiff computes and formats the ownership difference between before and after states.
func formatOwnershipDiff(r schema.ComparisonDetails) string {
	beforeOwners := r.BeforeOwners
	afterOwners := r.AfterOwners

	switch r.Status {
	case schema.NewStatus:
		// New file - show current owners
		if len(afterOwners) > 0 {
			return fmt.Sprintf("New: %s", schema.FormatOwners(afterOwners))
		}
		return "New"

	case schema.InactiveStatus:
		// Inactive file - show previous owners
		if len(beforeOwners) > 0 {
			return fmt.Sprintf("Removed: %s", schema.FormatOwners(beforeOwners))
		}
		return "Removed"

	default:
		// Active file - compare ownership stability
		if len(afterOwners) > 0 {
			if schema.OwnersEqual(beforeOwners, afterOwners) {
				return fmt.Sprintf("%s (stable)", schema.FormatOwners(afterOwners))
			}
			return schema.FormatOwners(afterOwners)
		}
		if len(beforeOwners) > 0 {
			return fmt.Sprintf("No owners (was: %s)", schema.FormatOwners(beforeOwners))
		}
		return "No owners"
	}
}
