// Package internal has helpers that are exclusive to the app runtime.
package internal

import (
	"os"

	"github.com/huangsam/hotspot/internal/contract"
)

// getPlainLabel returns a plain text label indicating the criticality level
// based on the file's importance score. This is the core logic used for
// CSV, JSON, and table printing.
// - Critical (>=80)
// - High (>=60)
// - Moderate (>=40)
// - Low (<40)
func getPlainLabel(score float64) string {
	return contract.GetPlainLabel(score)
}

// getColorLabel returns a colored text label for console output (table).
// It uses getLabelText to determine the string, and then applies the appropriate color.
func getColorLabel(score float64) string {
	return contract.GetColorLabel(score)
}

// selectOutputFile returns the appropriate file handle for output, based on the provided
// file path and format type. It falls back to os.Stdout on error.
// This function replaces both selectCSVOutputFile and selectJSONOutputFile.
func selectOutputFile(filePath string) (*os.File, error) {
	return contract.SelectOutputFile(filePath)
}
