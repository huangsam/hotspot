package outwriter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// writeJSONResultsForComparison marshals the schema.ComparisonResult to JSON and writes it.
func writeJSONResultsForComparison(w io.Writer, comparisonResult schema.ComparisonResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	// NOTE: The ComparisonResult struct fields are already public (uppercase),
	// so they will be correctly marshaled to JSON.
	return encoder.Encode(comparisonResult)
}

// writeCSVResultsForComparison writes the schema.ComparisonResult data to a CSV writer.
func writeCSVResultsForComparison(w *csv.Writer, comparisonResult schema.ComparisonResult, fmtFloat func(float64) string, intFmt string) error {
	// 1. Write Header Row
	header := []string{
		"rank",
		"path",
		"base_score",
		"comp_score",
		"delta_score",
		"delta_commits",
		"delta_churn",
		"before_owners",
		"after_owners",
		"mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// 2. Write Data Rows
	for i, r := range comparisonResult.Results {
		row := []string{
			strconv.Itoa(i + 1),                 // Rank
			r.Path,                              // Path
			fmtFloat(r.BeforeScore),             // Base Score
			fmtFloat(r.AfterScore),              // Current Score
			fmtFloat(r.Delta),                   // Delta Score (Current - Base)
			fmt.Sprintf(intFmt, r.DeltaCommits), // Delta Commits
			fmt.Sprintf(intFmt, r.DeltaChurn),   // Delta Churn
			strings.Join(r.BeforeOwners, "|"),   // Base Owners
			strings.Join(r.AfterOwners, "|"),    // Current Owners
			string(r.Mode),                      // Mode
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
