package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintComparisonResults outputs the analysis results, dispatching based on the output format configured.
func PrintComparisonResults(comparisonResult schema.ComparisonResult, cfg *Config, duration time.Duration) error {
	// Helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case schema.JSONOut:
		if err := printJSONResultsForComparison(comparisonResult, cfg); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		if err := printCSVResultsForComparison(comparisonResult, cfg, fmtFloat, intFmt); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		if err := printComparisonTable(comparisonResult, cfg, fmtFloat, intFmt, duration); err != nil {
			return fmt.Errorf("error writing comparison table output: %w", err)
		}
	}
	return nil
}

// printJSONResultsForComparison handles opening the file and calling the JSON writer.
func printJSONResultsForComparison(comparisonResult schema.ComparisonResult, cfg *Config) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := writeJSONResultsForComparison(file, comparisonResult); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote JSON comparison results to %s\n", cfg.OutputFile)
	}
	return nil
}

// printCSVResultsForComparison handles opening the file and calling the CSV writer.
func printCSVResultsForComparison(comparisonResult schema.ComparisonResult, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	w := csv.NewWriter(file)
	if err := writeCSVResultsForComparison(w, comparisonResult, fmtFloat, intFmt); err != nil {
		return err
	}
	w.Flush()

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote CSV comparison results to %s\n", cfg.OutputFile)
	}
	return nil
}

// printComparisonTable prints the metrics in a custom comparison format.
func printComparisonTable(comparisonResult schema.ComparisonResult, cfg *Config, fmtFloat func(float64) string, intFmt string, duration time.Duration) error {
	// Use os.Stdout, consistent with existing table printing
	table := tablewriter.NewWriter(os.Stdout)

	// --- 1. Define Headers (Comparison Mode) ---
	// Note: Use clear headers for base, comparison, and the change (Delta)
	headers := []string{
		"Rank",
		"Path",
		"Before",
		"After",
		"Delta",
		"Status",
	}
	if cfg.Detail {
		headers = append(headers, "Δ Churn")
	}
	if cfg.Owner {
		headers = append(headers, "Ownership")
	}
	table.Header(headers)

	// 2. Configure Alignment
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	// --- 3. Prepare Data Rows ---
	var data [][]string
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	for i, r := range comparisonResult.Results {
		var deltaStr string
		deltaValue := r.Delta
		switch {
		case deltaValue > 0:
			// Explicitly add + sign
			deltaStr = red(fmt.Sprintf("+%.*f ▲", cfg.Precision, deltaValue))
		case deltaValue < 0:
			// Keeps the - sign from the float
			deltaStr = green(fmt.Sprintf("%.*f ▼", cfg.Precision, deltaValue))
		default:
			// For 0.0 deltas, format simply without an indicator
			deltaStr = yellow(fmt.Sprintf("%.*f", cfg.Precision, 0.0))
		}

		// Prepare the row data as a slice of strings
		row := []string{
			strconv.Itoa(i + 1),                             // Rank
			truncatePath(r.Path, GetMaxTablePathWidth(cfg)), // File Path
			fmtFloat(r.BeforeScore),                         // Base Score
			fmtFloat(r.AfterScore),                          // Comparison Score
			deltaStr,                                        // Delta Score
			string(r.Status),                                // Status
		}
		if cfg.Detail {
			row = append(row, fmt.Sprintf(intFmt, r.DeltaChurn))
		}
		if cfg.Owner {
			row = append(row, formatOwnershipDiff(r))
		}
		data = append(data, row)
	}

	// --- 4. Render the table ---
	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}
	// Compute summary stats
	numItems := len(comparisonResult.Results)
	fmt.Printf("Showing top %d changes\n", numItems)
	fmt.Printf("Net score delta: %.*f, Net churn delta: %d\n", cfg.Precision, comparisonResult.Summary.NetScoreDelta, comparisonResult.Summary.NetChurnDelta)
	fmt.Printf("New files: %d, Inactive files: %d, Modified files: %d, Ownership changes: %d\n", comparisonResult.Summary.TotalNewFiles, comparisonResult.Summary.TotalInactiveFiles, comparisonResult.Summary.TotalModifiedFiles, comparisonResult.Summary.TotalOwnershipChanges)
	fmt.Printf("Analysis completed in %v using %d workers. Cache backend: %s\n", duration, cfg.Workers, cfg.CacheBackend)
	return nil
}
