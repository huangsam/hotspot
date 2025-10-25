// Package internal has helpers that are exclusive to the app runtime.
package internal

import "github.com/fatih/color"

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
