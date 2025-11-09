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
			Label:        getPlainLabel(r.Score),
			FolderResult: r,
		}
	}

	// 2. Create a JSON encoder
	encoder := json.NewEncoder(w)
	// Use indenting for cleaner output, especially when writing to a file
	encoder.SetIndent("", "  ")

	// 3. Encode and write the data
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
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
			strings.Join(r.Owners, "|"),     // Owners
			string(r.Mode),                  // Mode
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
