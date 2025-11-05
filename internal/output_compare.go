package internal

import (
	"fmt"
	"os"
	"strconv"

	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintComparisonResults outputs the analysis results in a formatted table.
func PrintComparisonResults(metrics []schema.ComparisonMetrics, cfg *Config) {
	// Helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// NOTE: For brevity, we'll only implement the default table view here.
	// JSON and CSV helpers would also be needed if supported.

	// Default to human-readable table
	if err := printComparisonTable(metrics, cfg, fmtFloat, intFmt); err != nil {
		LogFatal("Error writing comparison table output", err)
	}
}

// printComparisonTable prints the metrics in a custom comparison format.
func printComparisonTable(metrics []schema.ComparisonMetrics, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	// Use os.Stdout, consistent with existing table printing
	table := tablewriter.NewWriter(os.Stdout)

	// --- 1. Define Headers (Comparison Mode) ---
	// Note: Use clear headers for base, comparison, and the change (Delta)
	headers := []string{
		"Rank",
		"File",
		"Base Score",
		"Comp Score",
		"Delta",
	}
	if cfg.Detail {
		headers = append(headers,
			"Δ Commits",
			"Δ Churn",
		)
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
		}
		if cfg.Detail {
			row = append(row,
				fmt.Sprintf(intFmt, r.DeltaCommits),
				fmt.Sprintf(intFmt, r.DeltaChurn),
			)
		}
		data = append(data, row)
	}

	// --- 4. Render the table ---
	if err := table.Bulk(data); err != nil {
		return err
	}
	return table.Render()
}
