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

// WriteFolderResults outputs the analysis results, dispatching based on the output format configured.
func WriteFolderResults(results []schema.FolderResult, cfg *contract.Config, duration time.Duration) error {
	// Create formatters using helper
	fmtFloat, intFmt := createFormatters(cfg.Precision)

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case schema.JSONOut:
		if err := writeFolderJSONResults(results, cfg); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		if err := writeFolderCSVResults(results, cfg, fmtFloat, intFmt); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
			return writeFolderTable(results, cfg, fmtFloat, intFmt, duration, w)
		}, "Wrote table")
	}
	return nil
}

// writeFolderJSONResults handles opening the file and calling the JSON writer.
func writeFolderJSONResults(results []schema.FolderResult, cfg *contract.Config) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		return writeJSONResultsForFolders(w, results)
	}, "Wrote JSON")
}

// writeFolderCSVResults handles opening the file and calling the CSV writer.
func writeFolderCSVResults(results []schema.FolderResult, cfg *contract.Config, fmtFloat func(float64) string, intFmt string) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()
		return writeCSVResultsForFolders(csvWriter, results, fmtFloat, intFmt)
	}, "Wrote CSV")
}

// writeFolderTable writes the results in the custom folder-centric format,
// using the tablewriter API.
func writeFolderTable(results []schema.FolderResult, cfg *contract.Config, fmtFloat func(float64) string, intFmt string, duration time.Duration, writer io.Writer) error {
	table := tablewriter.NewWriter(writer)
	defer func() { _ = table.Close() }()

	// 1. Define Headers (Folder Mode - Custom)
	headers := []string{"Rank", "Path", "Score", "Label"}
	if cfg.Detail {
		headers = append(headers, "Commits", "Churn", "LOC")
	}
	if cfg.Owner {
		headers = append(headers, "Owner")
	}
	table.Header(headers)

	// 2. Configure Alignment
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	// 3. Prepare Data Rows
	var data [][]string
	for i, r := range results {
		// Prepare the row data as a slice of strings
		label := contract.GetPlainLabel(r.Score)
		if cfg.UseColors {
			label = contract.GetColorLabel(r.Score)
		}
		row := []string{
			strconv.Itoa(i + 1), // Rank
			contract.TruncatePath(r.Path, getMaxTablePathWidth(cfg)), // Folder Path
			fmtFloat(r.Score), // Score
			label,             // Label
		}
		if cfg.Detail {
			row = append(row,
				fmt.Sprintf(intFmt, r.Commits),  // Total Commits
				fmt.Sprintf(intFmt, r.Churn),    // Total Churn
				fmt.Sprintf(intFmt, r.TotalLOC), // Total LOC
			)
		}
		if cfg.Owner {
			row = append(row, schema.FormatOwners(r.Owners)) // Top 2 owners
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
	numFolders := len(results)
	totalCommits := 0
	totalChurn := 0
	totalLOC := 0
	for _, r := range results {
		totalCommits += r.Commits
		totalChurn += r.Churn
		totalLOC += r.TotalLOC
	}
	if _, err := fmt.Fprintf(writer, "Showing top %d folders (total commits: %d, total churn: %d, total LOC: %d)\n", numFolders, totalCommits, totalChurn, totalLOC); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Analysis completed in %v with %d workers. Cache backend: %s\n", duration, cfg.Workers, cfg.CacheBackend); err != nil {
		return err
	}
	return nil
}

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
