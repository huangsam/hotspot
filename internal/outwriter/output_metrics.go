package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// WriteMetricsDefinitions displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func WriteMetricsDefinitions(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, cfg *contract.Config) error {
	// Build the complete render model with all processed data
	renderModel := buildMetricsRenderModel(activeWeights)

	switch cfg.Output {
	case schema.JSONOut:
		return writeJSONMetrics(w, renderModel)
	case schema.CSVOut:
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()
		return writeCSVMetrics(csvWriter, renderModel)
	default:
		return writeMetricsText(w, renderModel, cfg)
	}
}

// writeMetricsText displays metrics in human-readable text format.
func writeMetricsText(w io.Writer, renderModel *schema.MetricsRenderModel, _ *contract.Config) error {
	if _, err := fmt.Fprintf(w, "Hotspot Scoring Modes\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "=====================\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", renderModel.Description); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}

	for _, mode := range renderModel.Modes {
		// Add emoji prefix for display
		displayName := getDisplayNameForMode(mode.Name)
		if _, err := fmt.Fprintf(w, "%s: %s\n", displayName, mode.Purpose); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Factors: %s\n", strings.Join(mode.Factors, ", ")); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Formula: Score = %s\n", mode.Formula); err != nil {
			return err
		}
	}

	return nil
}

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
