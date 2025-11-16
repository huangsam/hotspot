package internal

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// writeCSVResults writes the analysis results in CSV format.
func writeCSVResults(w *csv.Writer, files []schema.FileResult, fmtFloat func(float64) string, intFmt string) error {
	// CSV header
	header := []string{
		"rank",
		"file",
		"score",
		"label",
		"contributors",
		"commits",
		"size_kb",
		"age_days",
		"churn",
		"gini",
		"first_commit",
		"owner",
		"mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for i, f := range files {
		rec := []string{
			strconv.Itoa(i + 1),             // Rank
			f.Path,                          // File Path
			fmtFloat(f.Score),               // Score
			contract.GetPlainLabel(f.Score), // Label
			fmt.Sprintf(intFmt, f.UniqueContributors), // Contributors
			fmt.Sprintf(intFmt, f.Commits),            // Commits
			fmtFloat(float64(f.SizeBytes) / 1024.0),   // Size in KB
			fmt.Sprintf(intFmt, f.AgeDays),            // Age in Days
			fmt.Sprintf(intFmt, f.Churn),              // Churn
			fmtFloat(f.Gini),                          // Gini Coefficient
			f.FirstCommit.Format(DateTimeFormat),      // First Commit Date
			strings.Join(f.Owners, "|"),               // Owners
			string(f.Mode),                            // Mode
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

// writeJSONResults writes the analysis results in JSON format.
func writeJSONResults(w io.Writer, files []schema.FileResult) error {
	// 1. Prepare the data structure for JSON with rank and label added
	type JSONFileResult struct {
		Rank  int    `json:"rank"`
		Label string `json:"label"`
		schema.FileResult
	}

	output := make([]JSONFileResult, len(files))
	for i, f := range files {
		output[i] = JSONFileResult{
			Rank:       i + 1,
			Label:      contract.GetPlainLabel(f.Score),
			FileResult: f,
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
