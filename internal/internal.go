// Package internal has helpers that are only useful within the hotspot runtime.
package internal

import "github.com/fatih/color"

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
