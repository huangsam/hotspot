package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintFolderResults outputs the analysis results, dispatching based on the output format configured.
func PrintFolderResults(results []schema.FolderResults, cfg *Config) {
	// helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// Dispatcher: Handle different output formats
	switch strings.ToLower(cfg.Output) {
	case "json":
		if err := printJSONResultsForFolders(results, cfg); err != nil {
			LogFatal("Error writing JSON output", err)
		}
	case "csv":
		if err := printCSVResultsForFolders(results, cfg, fmtFloat, intFmt); err != nil {
			LogFatal("Error writing CSV output", err)
		}
	default:
		// Default to human-readable table
		if err := printFolderTable(results, cfg, fmtFloat, intFmt); err != nil {
			LogFatal("Error writing table output", err)
		}
	}
}

// printJSONResultsForFolders handles opening the file and calling the JSON writer.
func printJSONResultsForFolders(results []schema.FolderResults, cfg *Config) error {
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
func printCSVResultsForFolders(results []schema.FolderResults, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
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
func printFolderTable(results []schema.FolderResults, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	table := tablewriter.NewWriter(os.Stdout)

	// 1. Define Headers (Folder Mode - Custom)
	headers := []string{"Rank", "Folder", "Score", "Label", "Commits", "Churn", "LOC"}
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
			strconv.Itoa(i + 1),                     // Rank
			truncatePath(r.Path, maxTablePathWidth), // Folder Path
			fmtFloat(r.Score),                       // Score
			getColorLabel(r.Score),                  // Label
			fmt.Sprintf(intFmt, r.Commits),          // Total Commits
			fmt.Sprintf(intFmt, r.Churn),            // Total Churn
			fmt.Sprintf(intFmt, r.TotalLOC),         // Total LOC
		}
		if cfg.Owner {
			row = append(row, r.Owner) // Owner
		}
		data = append(data, row)
	}

	// 4. Render the table
	if err := table.Bulk(data); err != nil {
		return err
	}
	return table.Render()
}
