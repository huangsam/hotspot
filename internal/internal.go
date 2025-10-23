// Package internal has helpers that are only useful within the hotspot runtime.
package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/huangsam/hotspot/schema"
)

var (
	criticalColor = color.New(color.FgRed, color.Bold)    // Critical: Red and Bold
	highColor     = color.New(color.FgYellow, color.Bold) // High: Yellow and Bold
	moderateColor = color.New(color.FgGreen)              // Moderate: Green
	lowColor      = color.New(color.FgHiBlack)            // Low: Dark Grey/HiBlack
)

// getTextLabel returns a text label indicating the criticality level
// based on the file's importance score, colored using fatih/color:
// - Critical (>=80)
// - High (>=60)
// - Moderate (>=40)
// - Low (<40)
func getTextLabel(score float64) string {
	switch {
	case score >= 80:
		return criticalColor.Sprint("Critical")
	case score >= 60:
		return highColor.Sprint("High")
	case score >= 40:
		return moderateColor.Sprint("Moderate")
	default:
		return lowColor.Sprint("Low")
	}
}

// metricBreakdown holds a key-value pair from the Breakdown map representing a metric's contribution.
type metricBreakdown struct {
	Name  string  // e.g., "commits", "churn", "size"
	Value float64 // The percentage contribution to the score
}

// formatTopMetricContributors computes the top 3 metric components that contribute to the final score.
func formatTopMetricContributors(f *schema.FileMetrics) string {
	var metrics []metricBreakdown

	// Define the known metric keys you want to include in the breakdown
	// (excluding inv_contrib if it's just the inverse of contrib).
	metricKeys := map[string]bool{
		"contrib": true,
		"commits": true,
		"size":    true,
		"age":     true,
		"churn":   true,
		"gini":    true,
		// Add other core metrics here, if any.
	}

	// 1. Filter and Convert Map to Slice
	for k, v := range f.Breakdown {
		// Only include known metrics, and skip those with zero contribution
		if metricKeys[k] && v != 0.0 {
			metrics = append(metrics, metricBreakdown{
				Name:  k,
				Value: v, // This is the percentage contribution
			})
		}
	}

	if len(metrics) == 0 {
		return "No significant metric contributors"
	}

	// 2. Sort the Slice by Value (Contribution %) in Descending Order
	// Metrics with the highest absolute percentage contribution come first.
	sort.Slice(metrics, func(i, j int) bool {
		// We compare the absolute value since some contributions might be negative
		// if the model is set up to penalize certain metrics.
		return abs(metrics[i].Value) > abs(metrics[j].Value)
	})

	// 3. Limit to Top 3 and Format the Output
	var parts []string
	limit := 3
	if len(metrics) < limit {
		limit = len(metrics)
	}

	for i := 0; i < limit; i++ {
		m := metrics[i]
		// Format as "metric (Percentage%)"
		parts = append(parts, fmt.Sprintf("%s (%.0f%%)", m.Name, m.Value))
	}

	return strings.Join(parts, ", ")
}

// abs helper function for absolute value comparison
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
