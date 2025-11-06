package internal

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/huangsam/hotspot/schema"
)

// writeJSONResultsForComparison marshals the schema.ComparisonMetrics slice to JSON and writes it.
func writeJSONResultsForComparison(w io.Writer, metrics []schema.ComparisonResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	// NOTE: The ComparisonMetrics struct fields are already public (uppercase),
	// so they will be correctly marshaled to JSON.
	return encoder.Encode(metrics)
}

// writeCSVResultsForComparison writes the schema.ComparisonMetrics data to a CSV writer.
func writeCSVResultsForComparison(w *csv.Writer, metrics []schema.ComparisonResult, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	// 1. Write Header Row
	header := []string{
		"rank",
		"path",
		"base_score",
		"comp_score",
		"delta_score",
	}
	if cfg.Detail {
		header = append(header,
			"delta_commits",
			"delta_churn",
		)
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// 2. Write Data Rows
	for i, r := range metrics {
		row := []string{
			strconv.Itoa(i + 1), // Rank
			r.Path,              // Path
			fmtFloat(r.BeforeScore),
			fmtFloat(r.AfterScore),
			fmtFloat(r.Delta), // Delta Score (Comp - Base)
		}
		if cfg.Detail {
			row = append(row,
				fmt.Sprintf(intFmt, r.DeltaCommits),
				fmt.Sprintf(intFmt, r.DeltaChurn),
			)
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
