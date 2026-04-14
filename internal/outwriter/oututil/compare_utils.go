package oututil

import (
	"fmt"

	"github.com/fatih/color"
)

// FormatComparisonDelta returns a formatted string for a comparison delta, optionally with colors.
func FormatComparisonDelta(delta float64, precision int, useColors bool) string {
	var red, green, yellow func(...any) string
	if useColors {
		red = color.New(color.FgRed).SprintFunc()
		green = color.New(color.FgGreen).SprintFunc()
		yellow = color.New(color.FgYellow).SprintFunc()
	} else {
		red = fmt.Sprint
		green = fmt.Sprint
		yellow = fmt.Sprint
	}

	switch {
	case delta > 0:
		// Explicitly add + sign
		return red(fmt.Sprintf("+%.*f ▲", precision, delta))
	case delta < 0:
		// Keeps the - sign from the float
		return green(fmt.Sprintf("%.*f ▼", precision, delta))
	default:
		// For 0.0 deltas, format simply without an indicator
		return yellow(fmt.Sprintf("%.*f", precision, 0.0))
	}
}
