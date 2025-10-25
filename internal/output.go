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

// PrintResults outputs the analysis results in a formatted table or exports them as CSV/JSON.
func PrintResults(files []schema.FileMetrics, cfg *schema.Config) {
	// helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case "json":
		if err := printJSONResults(files, cfg); err != nil {
			FatalError("Error writing JSON output", err)
		}
	case "csv":
		if err := printCSVResults(files, cfg, fmtFloat, intFmt); err != nil {
			FatalError("Error writing CSV output", err)
		}
	default:
		// Default to human-readable table
		printTableResults(files, cfg, fmtFloat, intFmt)
	}
}

// printJSONResults handles opening the file and calling the JSON writer.
func printJSONResults(files []schema.FileMetrics, cfg *schema.Config) error {
	// Use the unified file selector defined in writers.go
	file, err := selectOutputFile(cfg.JSONFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := writeJSONResults(file, files); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "Wrote JSON to %s\n", cfg.JSONFile)
	}
	return nil
}

// printCSVResults handles opening the file and calling the CSV writer.
func printCSVResults(files []schema.FileMetrics, cfg *schema.Config, fmtFloat func(float64) string, intFmt string) error {
	// Use the unified file selector defined in writers.go
	file, err := selectOutputFile(cfg.CSVFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	w := csv.NewWriter(file)
	if err := writeCSVResults(w, files, fmtFloat, intFmt); err != nil {
		return err
	}
	w.Flush()

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "Wrote CSV to %s\n", cfg.CSVFile)
	}
	return nil
}

// printTableResults generates and prints the human-readable table.
func printTableResults(files []schema.FileMetrics, cfg *schema.Config, fmtFloat func(float64) string, intFmt string) {
	detail := cfg.Detail
	explain := cfg.Explain

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
			truncatePath(f.Path, maxPathWidth), // File
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
	_ = table.Render()
}
