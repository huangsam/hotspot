package internal

import (
	"fmt"
	"os"
	"strconv"

	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintFolderResults outputs the analysis results in a formatted, human-readable table.
// It detects whether the results are for files or folders to adjust headers and metrics.
func PrintFolderResults(results []schema.FolderResults, cfg *Config) {
	// helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	if err := printFolderTable(results, fmtFloat, intFmt); err != nil {
		LogFatal("Error writing table output", err)
	}
}

// printFolderTable prints the results in the custom folder-centric format,
// using the tablewriter API.
func printFolderTable(results []schema.FolderResults, fmtFloat func(float64) string, intFmt string) error {
	table := tablewriter.NewWriter(os.Stdout)

	// 1. Define Headers (Folder Mode - Custom)
	headers := []string{"Rank", "Folder", "Score", "Label", "Total Commits", "Total Churn", "Total LOC"}
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
		data = append(data, row)
	}

	// 4. Render the table
	if err := table.Bulk(data); err != nil {
		return err
	}
	return table.Render()
}
