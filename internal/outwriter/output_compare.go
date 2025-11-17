package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// WriteComparisonResults outputs the analysis results, dispatching based on the output format configured.
func WriteComparisonResults(w io.Writer, comparisonResult schema.ComparisonResult, cfg *contract.Config, duration time.Duration) error {
	// Create formatters using helper
	fmtFloat, intFmt := createFormatters(cfg.Precision)

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case schema.JSONOut:
		if err := writeJSONResultsForComparison(w, comparisonResult); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()
		if err := writeCSVResultsForComparison(csvWriter, comparisonResult, fmtFloat, intFmt); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		return writeComparisonTable(comparisonResult, cfg, fmtFloat, intFmt, duration, w)
	}
	return nil
}

// writeComparisonTable writes the metrics in a custom comparison format.
func writeComparisonTable(comparisonResult schema.ComparisonResult, cfg *contract.Config, fmtFloat func(float64) string, intFmt string, duration time.Duration, writer io.Writer) error {
	table := tablewriter.NewWriter(writer)
	defer func() { _ = table.Close() }()

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
	var red, green, yellow func(...any) string
	if cfg.UseColors {
		red = color.New(color.FgRed).SprintFunc()
		green = color.New(color.FgGreen).SprintFunc()
		yellow = color.New(color.FgYellow).SprintFunc()
	} else {
		red = fmt.Sprint
		green = fmt.Sprint
		yellow = fmt.Sprint
	}
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
			strconv.Itoa(i + 1), // Rank
			contract.TruncatePath(r.Path, getMaxTablePathWidth(cfg)), // File Path
			fmtFloat(r.BeforeScore),                                  // Base Score
			fmtFloat(r.AfterScore),                                   // Comparison Score
			deltaStr,                                                 // Delta Score
			string(r.Status),                                         // Status
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
	if _, err := fmt.Fprintf(writer, "Showing top %d changes\n", numItems); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Net score delta: %.*f, Net churn delta: %d\n", cfg.Precision, comparisonResult.Summary.NetScoreDelta, comparisonResult.Summary.NetChurnDelta); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "New files: %d, Inactive files: %d, Modified files: %d, Ownership changes: %d\n", comparisonResult.Summary.TotalNewFiles, comparisonResult.Summary.TotalInactiveFiles, comparisonResult.Summary.TotalModifiedFiles, comparisonResult.Summary.TotalOwnershipChanges); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Analysis completed in %v with %d workers. Cache backend: %s\n", duration, cfg.Workers, cfg.CacheBackend); err != nil {
		return err
	}
	return nil
}

// writeJSONResultsForComparison marshals the schema.ComparisonResult to JSON and writes it.
func writeJSONResultsForComparison(w io.Writer, comparisonResult schema.ComparisonResult) error {
	return writeJSON(w, comparisonResult)
}

// writeCSVResultsForComparison writes the schema.ComparisonResult data to a CSV writer.
func writeCSVResultsForComparison(w *csv.Writer, comparisonResult schema.ComparisonResult, fmtFloat func(float64) string, intFmt string) error {
	// 1. Write Header Row
	header := []string{
		"rank",
		"path",
		"base_score",
		"comp_score",
		"delta_score",
		"delta_commits",
		"delta_churn",
		"before_owners",
		"after_owners",
		"mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// 2. Write Data Rows
	for i, r := range comparisonResult.Results {
		row := []string{
			strconv.Itoa(i + 1),                 // Rank
			r.Path,                              // Path
			fmtFloat(r.BeforeScore),             // Base Score
			fmtFloat(r.AfterScore),              // Current Score
			fmtFloat(r.Delta),                   // Delta Score (Current - Base)
			fmt.Sprintf(intFmt, r.DeltaCommits), // Delta Commits
			fmt.Sprintf(intFmt, r.DeltaChurn),   // Delta Churn
			strings.Join(r.BeforeOwners, "|"),   // Base Owners
			strings.Join(r.AfterOwners, "|"),    // Current Owners
			string(r.Mode),                      // Mode
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
