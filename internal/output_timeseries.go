package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintTimeseriesResults outputs the timeseries results, dispatching based on the output format configured.
func PrintTimeseriesResults(result schema.TimeseriesResult, cfg *Config, duration time.Duration) error {
	// Helper format strings and closure for number formatting
	numFmt := "%.*f"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// Dispatcher: Handle different output formats
	switch strings.ToLower(cfg.Output) {
	case schema.JSONOut:
		if err := printJSONResultsForTimeseries(result, cfg); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		if err := printCSVResultsForTimeseries(result, cfg, fmtFloat); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		if err := printTimeseriesTable(result, cfg, fmtFloat, duration); err != nil {
			return fmt.Errorf("error writing timeseries table output: %w", err)
		}
	}
	return nil
}

// printJSONResultsForTimeseries handles opening the file and calling the JSON writer.
func printJSONResultsForTimeseries(result schema.TimeseriesResult, cfg *Config) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := writeJSONResultsForTimeseries(file, result); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote JSON timeseries results to %s\n", cfg.OutputFile)
	}
	return nil
}

// printCSVResultsForTimeseries handles opening the file and calling the CSV writer.
func printCSVResultsForTimeseries(result schema.TimeseriesResult, cfg *Config, fmtFloat func(float64) string) error {
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	w := csv.NewWriter(file)
	if err := writeCSVResultsForTimeseries(w, result, cfg, fmtFloat); err != nil {
		return err
	}
	w.Flush()

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote CSV timeseries results to %s\n", cfg.OutputFile)
	}
	return nil
}

// printTimeseriesTable prints the timeseries in a four-column table.
func printTimeseriesTable(result schema.TimeseriesResult, cfg *Config, fmtFloat func(float64) string, duration time.Duration) error {
	// Use os.Stdout, consistent with existing table printing
	table := tablewriter.NewWriter(os.Stdout)

	// --- 1. Define Headers ---
	headers := []string{"Path", "Period", "Score", "Mode", "Owners"}
	table.Header(headers)

	// 2. Configure Alignment
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	// --- 3. Prepare Data Rows ---
	var data [][]string
	for _, p := range result.Points {
		ownersStr := ""
		if len(p.Owners) > 0 {
			ownersStr = schema.FormatOwners(p.Owners)
		} else {
			ownersStr = "No owners"
		}
		row := []string{
			truncatePath(p.Path, maxTablePathWidth),
			p.Period,
			fmtFloat(p.Score),
			cfg.Mode,
			ownersStr,
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

	fmt.Printf("Timeseries analysis completed in %v using %d workers.\n", duration, cfg.Workers)
	return nil
}
