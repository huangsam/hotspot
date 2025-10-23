package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/huangsam/hotspot/schema"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

const maxPathWidth = 40

// PrintResults outputs the analysis results in a formatted table.
// For each file it shows rank, path (truncated if needed), importance score,
// criticality label, and all individual metrics that contribute to the score.
func PrintResults(files []schema.FileMetrics, cfg *schema.Config) {
	detail := cfg.Detail
	explain := cfg.Explain

	precision := cfg.Precision
	outFmt := cfg.Output

	// helper format strings for numbers
	numFmt := "%.*f"
	intFmt := "%d"
	// closure to format floats with the configured precision
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, precision, v)
	}

	// If CSV output requested, skip printing the human-readable table
	if outFmt == "csv" {
		file := selectOutputFile(cfg)
		w := csv.NewWriter(file)
		writeCSVResults(w, files, fmtFloat, intFmt)
		w.Flush()
		if file != os.Stdout {
			_ = file.Close()
			fmt.Fprintf(os.Stderr, "wrote CSV to %s\n", cfg.CSVFile)
		}
		return
	}

	// --- Tablewriter Implementation Starts Here ---

	// Initialize tablewriter and set output to os.Stdout
	table := tablewriter.NewWriter(os.Stdout)

	// 1. Define Headers
	headers := []string{"Rank", "File", "Score", "Label"}
	if detail {
		headers = append(headers, "Contrib", "Commits", "Size(kb)", "Age(d)", "Churn", "Gini", "First Commit")
	}
	if explain {
		headers = append(headers, "Explain")
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
			strconv.Itoa(i + 1),                // Rank
			truncatePath(f.Path, maxPathWidth), // File (Path truncation still needed)
			fmtFloat(f.Score),                  // Score
			getTextLabel(f.Score),              // Label
		}
		if detail {
			row = append(
				row,
				fmt.Sprintf(intFmt, f.UniqueContributors), // Contrib
				fmt.Sprintf(intFmt, f.Commits),            // Commits
				fmtFloat(float64(f.SizeBytes)/1024.0),     // Size(KB)
				fmt.Sprintf(intFmt, f.AgeDays),            // Age(d)
				fmt.Sprintf(intFmt, f.Churn),              // Churn
				fmtFloat(f.Gini),                          // Gini
				f.FirstCommit.Format("2006-01-02"),        // First Commit
			)
		}
		if explain {
			topOnes := formatTopMetricContributors(&f)
			row = append(row, topOnes)
		}
		data = append(data, row)
	}

	// 4. Render the table
	_ = table.Bulk(data)
	_ = table.Render() // This is where the magic happens: widths are calculated and output is printed.
}

// truncatePath truncates a file path to a maximum width with ellipsis prefix.
func truncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}
