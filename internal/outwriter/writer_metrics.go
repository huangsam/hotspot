package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// writeJSONMetrics writes the metrics definitions in JSON format.
func writeJSONMetrics(w io.Writer, renderModel *schema.MetricsRenderModel) error {
	return writeJSON(w, renderModel)
}

// writeCSVMetrics writes the metrics definitions in CSV format.
func writeCSVMetrics(w *csv.Writer, renderModel *schema.MetricsRenderModel) error {
	// Write header
	header := []string{"Mode", "Purpose", "Factors", "Formula"}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write each mode
	for _, mode := range renderModel.Modes {
		record := []string{
			mode.Name,
			mode.Purpose,
			strings.Join(mode.Factors, "|"),
			mode.Formula,
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}
