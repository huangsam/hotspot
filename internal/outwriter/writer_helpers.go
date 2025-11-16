package outwriter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/huangsam/hotspot/internal/contract"
)

// writeWithFile handles the common pattern of opening a file, writing to it, and cleaning up.
// It accepts a writer function that takes an io.Writer and returns an error.
func writeWithFile(outputFile string, writer func(io.Writer) error, successMsg string) error {
	file, err := contract.SelectOutputFile(outputFile)
	if err != nil {
		return err
	}
	// Only close if it's not stdout
	if file != os.Stdout {
		defer func() { _ = file.Close() }()
	}

	if err := writer(file); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 %s to %s\n", successMsg, outputFile)
	}
	return nil
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
