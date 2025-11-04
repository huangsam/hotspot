// Package internal has helpers that are exclusive to the app runtime.
package internal

import (
	"os"

	"github.com/fatih/color"
)

// maxTablePathWidth is the max width of filepath when rendered inside a table.
const maxTablePathWidth = 60

const (
	criticalValue = "Critical" // Critical value
	highValue     = "High"     // High value
	moderateValue = "Moderate" // Moderate value
	lowValue      = "Low"      // Low value
)

var (
	criticalColor = color.New(color.FgRed, color.Bold)    // Critical Heat
	highColor     = color.New(color.FgYellow, color.Bold) // High Heat
	moderateColor = color.New(color.FgYellow)             // Moderate Heat
	lowColor      = color.New(color.FgHiBlack)            // Low Heat
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

// selectOutputFile returns the appropriate file handle for output, based on the provided
// file path and format type. It falls back to os.Stdout on error.
// This function replaces both selectCSVOutputFile and selectJSONOutputFile.
func selectOutputFile(filePath string) (*os.File, error) {
	if filePath == "" {
		return os.Stdout, nil
	}
	return os.Create(filePath)
}
