// Package provider provides cross-provider formatting and rendering utilities.
package provider

import (
	"github.com/fatih/color"
	"github.com/huangsam/hotspot/schema"
)

// color variables for console output.
var (
	criticalColor = color.New(color.FgRed, color.Bold)     // criticalColor represents standard danger.
	highColor     = color.New(color.FgMagenta, color.Bold) // highColor represents strong, distinct warning.
	moderateColor = color.New(color.FgYellow)              // moderateColor represents standard caution, not bold.
	lowColor      = color.New(color.FgCyan)                // lowColor represents informational / low-priority signal.
)

// GetColorLabel returns a colored text label for console output (table).
// It uses schema.GetPlainLabel to determine the string, and then applies the appropriate color.
func GetColorLabel(score float64) string {
	text := schema.GetPlainLabel(score)

	switch text {
	case schema.CriticalValue:
		return criticalColor.Sprint(text)
	case schema.HighValue:
		return highColor.Sprint(text)
	case schema.ModerateValue:
		return moderateColor.Sprint(text)
	default: // "Low"
		return lowColor.Sprint(text)
	}
}

// SetColorMode explicitly enables or disables color output by setting the global fatih/color.NoColor variable.
// This allows forcing colors in non-TTY environments (like VHS or CI) when explicitly requested.
func SetColorMode(useColors bool) {
	color.NoColor = !useColors
}
