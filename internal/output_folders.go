package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintFolderResults outputs the analysis results, dispatching based on the output format configured.
func PrintFolderResults(results []schema.FolderResult, cfg *Config, duration time.Duration) error {
	// helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case schema.JSONOut:
		if err := printJSONResultsForFolders(results, cfg); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		if err := printCSVResultsForFolders(results, cfg, fmtFloat, intFmt); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		if err := printFolderTable(results, cfg, fmtFloat, intFmt, duration); err != nil {
			return fmt.Errorf("error writing table output: %w", err)
		}
	}
	return nil
}

// printJSONResultsForFolders handles opening the file and calling the JSON writer.
func printJSONResultsForFolders(results []schema.FolderResult, cfg *Config) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := writeJSONResultsForFolders(file, results); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote JSON to %s\n", cfg.OutputFile)
	}
	return nil
}

// printCSVResultsForFolders handles opening the file and calling the CSV writer.
func printCSVResultsForFolders(results []schema.FolderResult, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	w := csv.NewWriter(file)
	if err := writeCSVResultsForFolders(w, results, fmtFloat, intFmt); err != nil {
		return err
	}
	w.Flush()

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote CSV to %s\n", cfg.OutputFile)
	}
	return nil
}

// printFolderTable prints the results in the custom folder-centric format,
// using the tablewriter API.
func printFolderTable(results []schema.FolderResult, cfg *Config, fmtFloat func(float64) string, intFmt string, duration time.Duration) error {
	table := tablewriter.NewWriter(os.Stdout)

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
		row := []string{
			strconv.Itoa(i + 1),                             // Rank
			truncatePath(r.Path, GetMaxTablePathWidth(cfg)), // Folder Path
			fmtFloat(r.Score),                               // Score
			getColorLabel(r.Score),                          // Label
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
	fmt.Printf("Showing top %d folders (total commits: %d, total churn: %d, total LOC: %d)\n", numFolders, totalCommits, totalChurn, totalLOC)
	fmt.Printf("Analysis completed in %v with %d workers. Cache backend: %s\n", duration, cfg.Workers, cfg.CacheBackend)
	return nil
}
