package internal

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/huangsam/hotspot/schema"
)

// selectCSVOutputFile returns the appropriate file handle for CSV output.
func selectCSVOutputFile(cfg *schema.Config) *os.File {
	if cfg.CSVFile != "" {
		if file, err := os.Create(cfg.CSVFile); err == nil {
			return file
		}
		fmt.Fprintf(os.Stderr, "warning: cannot open csv file %s: default to stdout\n", cfg.CSVFile)
	}
	return os.Stdout
}

// writeCSVResults writes the analysis results in CSV format.
func writeCSVResults(w *csv.Writer, files []schema.FileMetrics, fmtFloat func(float64) string, intFmt string) error {
	// CSV header
	header := []string{
		"rank", "file", "score", "label", "contributors", "commits",
		"size_kb", "age_days", "churn", "gini", "first_commit",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for i, f := range files {
		rec := []string{
			strconv.Itoa(i + 1),
			f.Path,
			fmtFloat(f.Score),
			getPlainTextLabel(f.Score),
			fmt.Sprintf(intFmt, f.UniqueContributors),
			fmt.Sprintf(intFmt, f.Commits),
			fmtFloat(float64(f.SizeBytes) / 1024.0),
			fmt.Sprintf(intFmt, f.AgeDays),
			fmt.Sprintf(intFmt, f.Churn),
			fmtFloat(f.Gini),
			f.FirstCommit.Format("2006-01-02"),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

// JSONOutput represents the structure of the JSON data to be printed.
type JSONOutput struct {
	Rank  int    `json:"rank"`
	Label string `json:"label"`
	schema.FileMetrics
}

// selectJSONOutputFile returns the appropriate file handle for JSON output.
func selectJSONOutputFile(cfg *schema.Config) *os.File {
	if cfg.JSONFile != "" {
		if file, err := os.Create(cfg.JSONFile); err == nil {
			return file
		}
		fmt.Fprintf(os.Stderr, "warning: cannot open json file %s: default to stdout\n", cfg.JSONFile)
	}
	return os.Stdout
}

// writeJSONResults writes the analysis results in JSON format.
func writeJSONResults(w io.Writer, files []schema.FileMetrics) error {
	// 1. Prepare the data structure for JSON
	output := make([]JSONOutput, len(files))
	for i, f := range files {
		output[i] = JSONOutput{
			Rank:        i + 1,
			Label:       getPlainTextLabel(f.Score),
			FileMetrics: f,
		}
	}

	// 2. Create a JSON encoder
	encoder := json.NewEncoder(w)
	// Optional: Use Indent for pretty-printing if writing to a file,
	// otherwise omit for smaller output.
	encoder.SetIndent("", "  ")

	// 3. Encode and write the data
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}
