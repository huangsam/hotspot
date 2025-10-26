package internal

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/huangsam/hotspot/schema"
)

// selectOutputFile returns the appropriate file handle for output, based on the provided
// file path and format type. It falls back to os.Stdout on error.
// This function replaces both selectCSVOutputFile and selectJSONOutputFile.
func selectOutputFile(filePath string) (*os.File, error) {
	if filePath == "" {
		return nil, errors.New("no file specified")
	}
	return os.Create(filePath)
}

// writeCSVResults writes the analysis results in CSV format.
func writeCSVResults(w *csv.Writer, files []schema.FileMetrics, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	// CSV header
	header := []string{
		"rank", "file", "score", "label", "contributors", "commits",
		"size_kb", "age_days", "churn", "gini", "first_commit", "mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for i, f := range files {
		rec := []string{
			strconv.Itoa(i + 1),
			f.Path,
			fmtFloat(f.Score),
			getPlainLabel(f.Score),
			fmt.Sprintf(intFmt, f.UniqueContributors),
			fmt.Sprintf(intFmt, f.Commits),
			fmtFloat(float64(f.SizeBytes) / 1024.0),
			fmt.Sprintf(intFmt, f.AgeDays),
			fmt.Sprintf(intFmt, f.Churn),
			fmtFloat(f.Gini),
			f.FirstCommit.Format(DateFormat),
			cfg.Mode,
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

// JSONOutput represents the structure of the JSON data to be printed.
type JSONOutput struct {
	Rank               int    `json:"rank"`
	Label              string `json:"label"`
	Mode               string `json:"mode"`
	schema.FileMetrics        // Embeds Path, Score, etc.
}

// writeJSONResults writes the analysis results in JSON format.
func writeJSONResults(w io.Writer, files []schema.FileMetrics, cfg *Config) error {
	// 1. Prepare the data structure for JSON
	output := make([]JSONOutput, len(files))
	for i, f := range files {
		output[i] = JSONOutput{
			Rank:        i + 1,
			Label:       getPlainLabel(f.Score),
			Mode:        cfg.Mode,
			FileMetrics: f,
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
