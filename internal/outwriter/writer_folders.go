package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// writeJSONResultsForFolders marshals the schema.FolderResults slice to JSON and writes it.
func writeJSONResultsForFolders(w io.Writer, results []schema.FolderResult) error {
	// 1. Prepare the data structure for JSON with rank and label added
	type JSONFolderResult struct {
		Rank  int    `json:"rank"`
		Label string `json:"label"`
		schema.FolderResult
	}

	output := make([]JSONFolderResult, len(results))
	for i, r := range results {
		output[i] = JSONFolderResult{
			Rank:         i + 1,
			Label:        contract.GetPlainLabel(r.Score),
			FolderResult: r,
		}
	}

	// 2. Use the generic JSON writer
	return writeJSON(w, output)
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
			contract.GetPlainLabel(r.Score), // Label
			fmt.Sprintf(intFmt, r.Commits),  // Total Commits
			fmt.Sprintf(intFmt, r.Churn),    // Total Churn
			fmt.Sprintf(intFmt, r.TotalLOC), // Total LOC
			strings.Join(r.Owners, "|"),     // Owners
			string(r.Mode),                  // Mode
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
