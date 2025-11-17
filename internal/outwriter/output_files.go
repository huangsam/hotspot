package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// WriteFileResults outputs the analysis results, dispatching based on the output format configured.
func WriteFileResults(files []schema.FileResult, cfg *contract.Config, duration time.Duration) error {
	// Create formatters using helper
	fmtFloat, intFmt := createFormatters(cfg.Precision)

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case schema.JSONOut:
		if err := writeFileJSONResults(files, cfg); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		if err := writeFileCSVResults(files, cfg, fmtFloat, intFmt); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
			return writeFileTable(files, cfg, fmtFloat, intFmt, duration, w)
		}, "Wrote table")
	}
	return nil
}

// writeFileJSONResults handles opening the file and calling the JSON writer.
func writeFileJSONResults(files []schema.FileResult, cfg *contract.Config) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		return writeJSONResultsForFiles(w, files)
	}, "Wrote JSON")
}

// writeFileCSVResults handles opening the file and calling the CSV writer.
func writeFileCSVResults(files []schema.FileResult, cfg *contract.Config, fmtFloat func(float64) string, intFmt string) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()
		return writeCSVResultsForFiles(csvWriter, files, fmtFloat, intFmt)
	}, "Wrote CSV")
}

// writeFileTable generates and writes the human-readable table.
func writeFileTable(files []schema.FileResult, cfg *contract.Config, fmtFloat func(float64) string, intFmt string, duration time.Duration, writer io.Writer) error {
	table := tablewriter.NewWriter(writer)

	// 1. Define Headers
	headers := []string{"Rank", "Path", "Score", "Label"}
	if cfg.Detail {
		headers = append(headers, "Contrib", "Commits", "LOC", "Churn", "Age", "Gini")
	}
	if cfg.Explain {
		headers = append(headers, "Explain")
	}
	if cfg.Owner {
		headers = append(headers, "Owner")
	}
	table.Header(headers)

	// 2. Configure Separators/Borders to match a minimal look
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	// 3. Populate Rows
	var data [][]string
	for i, f := range files {
		// Prepare the row data as a slice of strings
		row := []string{
			strconv.Itoa(i + 1), // Rank
			contract.TruncatePath(f.Path, getMaxTablePathWidth(cfg)), // File
			fmtFloat(f.Score),               // Score
			contract.GetColorLabel(f.Score), // Label
		}
		if cfg.Detail {
			row = append(
				row,
				fmt.Sprintf(intFmt, f.UniqueContributors), // Contrib
				fmt.Sprintf(intFmt, f.Commits),            // Commits
				fmt.Sprintf(intFmt, f.LinesOfCode),        // LOC
				fmt.Sprintf(intFmt, f.Churn),              // Churn
				fmt.Sprintf(intFmt, f.AgeDays),            // Age
				fmtFloat(f.Gini),                          // Gini
			)
		}
		if cfg.Explain {
			topOnes := formatTopMetricBreakdown(&f)
			row = append(row, topOnes) // Breakdown explanation
		}
		if cfg.Owner {
			row = append(row, schema.FormatOwners(f.Owners)) // Top 2 owners
		}
		data = append(data, row)
	}

	// 4. Render the table
	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}
	// Compute summary stats
	numFiles := len(files)
	totalCommits := 0
	totalChurn := 0
	for _, f := range files {
		totalCommits += f.Commits
		totalChurn += f.Churn
	}
	if _, err := fmt.Fprintf(writer, "Showing top %d files (total commits: %d, total churn: %d)\n", numFiles, totalCommits, totalChurn); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Analysis completed in %v with %d workers. Cache backend: %s\n", duration, cfg.Workers, cfg.CacheBackend); err != nil {
		return err
	}
	return nil
}

// writeCSVResultsForFiles writes the analysis results in CSV format.
func writeCSVResultsForFiles(w *csv.Writer, files []schema.FileResult, fmtFloat func(float64) string, intFmt string) error {
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
			fmt.Sprintf(intFmt, f.UniqueContributors),     // Contributors
			fmt.Sprintf(intFmt, f.Commits),                // Commits
			fmtFloat(float64(f.SizeBytes) / 1024.0),       // Size in KB
			fmt.Sprintf(intFmt, f.AgeDays),                // Age in Days
			fmt.Sprintf(intFmt, f.Churn),                  // Churn
			fmtFloat(f.Gini),                              // Gini Coefficient
			f.FirstCommit.Format(contract.DateTimeFormat), // First Commit Date
			strings.Join(f.Owners, "|"),                   // Owners
			string(f.Mode),                                // Mode
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

// writeJSONResultsForFiles writes the analysis results in JSON format.
func writeJSONResultsForFiles(w io.Writer, files []schema.FileResult) error {
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

	// 2. Use the generic JSON writer
	return writeJSON(w, output)
}
