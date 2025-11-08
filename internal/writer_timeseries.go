package internal

import (
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/huangsam/hotspot/schema"
)

// writeJSONResultsForTimeseries marshals the schema.TimeseriesResult to JSON and writes it.
func writeJSONResultsForTimeseries(w io.Writer, result schema.TimeseriesResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// writeCSVResultsForTimeseries writes the schema.TimeseriesResult data to a CSV writer.
func writeCSVResultsForTimeseries(w *csv.Writer, result schema.TimeseriesResult, _ *Config, fmtFloat func(float64) string) error {
	// 1. Write Header Row
	header := []string{
		"path",
		"period",
		"score",
		"mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// 2. Write Data Rows
	for _, p := range result.Points {
		row := []string{
			p.Path,
			p.Period,
			fmtFloat(p.Score),
			p.Mode,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
