package outwriter

import (
	"encoding/csv"
	"io"
	"strings"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// writeJSONResultsForTimeseries marshals the schema.TimeseriesResult to JSON and writes it.
func writeJSONResultsForTimeseries(w io.Writer, result schema.TimeseriesResult) error {
	return writeJSON(w, result)
}

// writeCSVResultsForTimeseries writes the schema.TimeseriesResult data to a CSV writer.
func writeCSVResultsForTimeseries(w *csv.Writer, result schema.TimeseriesResult, fmtFloat func(float64) string) error {
	// 1. Write Header Row
	header := []string{
		"path",
		"period",
		"start",
		"end",
		"score",
		"owners",
		"mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// 2. Write Data Rows
	for _, p := range result.Points {
		ownersStr := ""
		if len(p.Owners) > 0 {
			ownersStr = strings.Join(p.Owners, "|")
		} else {
			ownersStr = ""
		}
		row := []string{
			p.Path,
			p.Period,
			p.Start.Format(contract.DateTimeFormat),
			p.End.Format(contract.DateTimeFormat),
			fmtFloat(p.Score),
			ownersStr,
			string(p.Mode),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
