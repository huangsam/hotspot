package outwriter

import (
	"math"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// getDisplayNameForMode returns the display name for a given mode name.
func getDisplayNameForMode(modeName string) string {
	return schema.GetDisplayNameForMode(modeName)
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
	return schema.BuildMetricsRenderModel(activeWeights)
}
