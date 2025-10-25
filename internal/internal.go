// Package internal has helpers that are only useful within the hotspot runtime.
package internal

import (
	"math"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/huangsam/hotspot/schema"
)

const (
	criticalValue = "Critical"
	highValue     = "High"
	moderateValue = "Moderate"
	lowValue      = "Low"
)

var (
	criticalColor = color.New(color.FgRed, color.Bold)    // Critical: Red and Bold
	highColor     = color.New(color.FgYellow, color.Bold) // High: Yellow and Bold
	moderateColor = color.New(color.FgGreen)              // Moderate: Green
	lowColor      = color.New(color.FgHiBlack)            // Low: Dark Grey/HiBlack
)

// getPlainLabel returns a plain text label indicating the criticality level
// based on the file's importance score. This is the core logic used for
// CSV, JSON, and table printing.
// - Critical (>=80)
// - High (>=60)
// - Moderate (>=40)
// - Low (<40)
func getPlainLabel(score float64) string {
	switch {
	case score >= 80:
		return criticalValue
	case score >= 60:
		return highValue
	case score >= 40:
		return moderateValue
	default:
		return lowValue
	}
}

// getColorLabel returns a colored text label for console output (table).
// It uses getLabelText to determine the string, and then applies the appropriate color.
func getColorLabel(score float64) string {
	text := getPlainLabel(score)

	switch text {
	case criticalValue:
		return criticalColor.Sprint(text)
	case highValue:
		return highColor.Sprint(text)
	case moderateValue:
		return moderateColor.Sprint(text)
	default: // "Low"
		return lowColor.Sprint(text)
	}
}

// metricBreakdown holds a key-value pair from the Breakdown map representing a metric's contribution.
type metricBreakdown struct {
	Name  string  // e.g., "commits", "churn", "size"
	Value float64 // The percentage contribution to the score
}

const (
	topNMetrics          = 3
	metricContribMinimum = 0.5
)

// formatTopMetricContributors computes the top 3 metric components that contribute to the final score.
func formatTopMetricContributors(f *schema.FileMetrics) string {
	var metrics []metricBreakdown

	// 1. Filter and Convert Map to Slice
	for k, v := range f.Breakdown {
		// Only include meaningful metrics
		if v >= metricContribMinimum {
			metrics = append(metrics, metricBreakdown{
				Name:  k,
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

// truncatePath truncates a file path to a maximum width with ellipsis prefix.
func truncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}
