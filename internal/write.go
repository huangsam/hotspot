package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/huangsam/hotspot/schema"
)

// selectOutputFile returns the appropriate file handle for CSV output.
func selectOutputFile(cfg *schema.Config) *os.File {
	if cfg.CSVFile != "" {
		if file, err := os.Create(cfg.CSVFile); err == nil {
			return file
		}
		fmt.Fprintf(os.Stderr, "warning: cannot open csv file %s: falling back to stdout\n", cfg.CSVFile)
	}
	return os.Stdout
}

// writeCSVResults writes the analysis results in CSV format.
func writeCSVResults(w *csv.Writer, files []schema.FileMetrics, fmtFloat func(float64) string, intFmt string) {
	// CSV header
	_ = w.Write([]string{"rank", "file", "score", "label", "contributors", "commits", "size_kb", "age_days", "churn", "gini", "first_commit"})
	for i, f := range files {
		rec := []string{
			strconv.Itoa(i + 1),
			f.Path,
			fmtFloat(f.Score),
			getTextLabel(f.Score),
			fmt.Sprintf(intFmt, f.UniqueContributors),
			fmt.Sprintf(intFmt, f.Commits),
			fmtFloat(float64(f.SizeBytes) / 1024.0),
			fmt.Sprintf(intFmt, f.AgeDays),
			fmt.Sprintf(intFmt, f.Churn),
			fmtFloat(f.Gini),
			f.FirstCommit.Format("2006-01-02"),
		}
		_ = w.Write(rec)
	}
}
