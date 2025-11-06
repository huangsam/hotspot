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

// PrintComparisonResults outputs the analysis results, dispatching based on the output format configured.
func PrintComparisonResults(metrics []schema.ComparisonMetrics, cfg *Config) {
	// Helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// Dispatcher: Handle different output formats
	switch strings.ToLower(cfg.Output) {
	case "json":
		if err := printJSONResultsForComparison(metrics, cfg); err != nil {
			LogFatal("Error writing JSON output", err)
		}
	case "csv":
		if err := printCSVResultsForComparison(metrics, cfg, fmtFloat, intFmt); err != nil {
			LogFatal("Error writing CSV output", err)
		}
	default:
		// Default to human-readable table
		if err := printComparisonTable(metrics, cfg, fmtFloat, intFmt); err != nil {
			LogFatal("Error writing comparison table output", err)
		}
	}
}

// printJSONResultsForComparison handles opening the file and calling the JSON writer.
func printJSONResultsForComparison(metrics []schema.ComparisonMetrics, cfg *Config) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := writeJSONResultsForComparison(file, metrics); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote JSON comparison results to %s\n", cfg.OutputFile)
	}
	return nil
}

// printCSVResultsForComparison handles opening the file and calling the CSV writer.
func printCSVResultsForComparison(metrics []schema.ComparisonMetrics, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	w := csv.NewWriter(file)
	if err := writeCSVResultsForComparison(w, metrics, cfg, fmtFloat, intFmt); err != nil {
		return err
	}
	w.Flush()

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote CSV comparison results to %s\n", cfg.OutputFile)
	}
	return nil
}

// printComparisonTable prints the metrics in a custom comparison format.
func printComparisonTable(metrics []schema.ComparisonMetrics, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
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
	table.Header(headers)

	// 2. Configure Alignment
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	// --- 3. Prepare Data Rows ---
	var data [][]string
	for i, r := range metrics {
		var deltaStr string
		deltaValue := r.Delta
		switch {
		case deltaValue > 0:
			// Explicitly add + sign
			deltaStr = fmt.Sprintf("+%.*f ▲", cfg.Precision, deltaValue)
		case deltaValue < 0:
			// Keeps the - sign from the float
			deltaStr = fmt.Sprintf("%.*f ▼", cfg.Precision, deltaValue)
		default:
			// For 0.0 deltas, format simply without an indicator
			deltaStr = fmt.Sprintf("%.*f", cfg.Precision, 0.0)
		}

		// Prepare the row data as a slice of strings
		row := []string{
			strconv.Itoa(i + 1),                     // Rank
			truncatePath(r.Path, maxTablePathWidth), // File Path
			fmtFloat(r.BaseScore),                   // Base Score
			fmtFloat(r.CompScore),                   // Comparison Score
			deltaStr,                                // Delta Score
			r.Status,                                // Status
		}
		if cfg.Detail {
			row = append(row, fmt.Sprintf(intFmt, r.DeltaChurn))
		}
		data = append(data, row)
	}

	// --- 4. Render the table ---
	if err := table.Bulk(data); err != nil {
		return err
	}
	return table.Render()
}
