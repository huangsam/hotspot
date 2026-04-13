package outwriter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// truncatePath truncates a file path to a maximum width with ellipsis prefix.
// Requires maxWidth > 3 to ensure there's space for both the "..." prefix and at least one character of content.
func truncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth && maxWidth > 3 {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
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
