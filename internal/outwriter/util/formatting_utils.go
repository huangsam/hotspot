package util

import (
	"encoding/csv"
	"fmt"
	"io"
)

// TruncatePath truncates a file path to a maximum width with ellipsis prefix.
// Requires maxWidth > 3 to ensure there's space for both the "..." prefix and at least one character of content.
func TruncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth && maxWidth > 3 {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}

// WriteCSVWithHeader handles the common pattern of creating a CSV writer,
// writing a header, and writing data rows.
func WriteCSVWithHeader(w io.Writer, header []string, writeRows func(*csv.Writer) error) error {
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

// CreateFormatters creates the common float formatter closures used across multiple output types.
func CreateFormatters(precision int) (fmtFloat func(float64) string) {
	numFmt := "%.*f"
	fmtFloat = func(v float64) string {
		return fmt.Sprintf(numFmt, precision, v)
	}
	return fmtFloat
}

// formatWeights was moved to schema.FormatWeights
