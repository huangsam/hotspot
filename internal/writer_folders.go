package internal

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// writeJSONResultsForFolders marshals the schema.FolderResults slice to JSON and writes it.
func writeJSONResultsForFolders(w io.Writer, results []schema.FolderResult) error {
	// For JSON, we can write the raw structure directly, avoiding unnecessary formatting.
	encoder := json.NewEncoder(w)
	// Use indenting for cleaner output, especially when writing to a file
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// writeCSVResultsForFolders writes the schema.FolderResults data to a CSV writer.
func writeCSVResultsForFolders(w *csv.Writer, results []schema.FolderResult, fmtFloat func(float64) string, intFmt string) error {
	// 1. Write Header Row
	header := []string{
		"rank",
		"folder",
		"score",
		"label",
		"total_commits",
		"total_churn",
		"total_loc",
		"owner",
		"mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// 2. Write Data Rows
	for i, r := range results {
		row := []string{
			strconv.Itoa(i + 1),             // Rank
			r.Path,                          // Folder Path
			fmtFloat(r.Score),               // Score
			getPlainLabel(r.Score),          // Label
			fmt.Sprintf(intFmt, r.Commits),  // Total Commits
			fmt.Sprintf(intFmt, r.Churn),    // Total Churn
			fmt.Sprintf(intFmt, r.TotalLOC), // Total LOC
			strings.Join(r.Owners, ", "),    // Owners
			r.Mode,                          // Mode
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
