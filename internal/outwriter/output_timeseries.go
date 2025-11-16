package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintTimeseriesResults outputs the timeseries results, dispatching based on the output format configured.
func PrintTimeseriesResults(result schema.TimeseriesResult, cfg *contract.Config, duration time.Duration) error {
	// Create formatters using helper
	fmtFloat, _ := createFormatters(cfg.Precision)

	// Dispatcher: Handle different output formats
	switch cfg.Output {
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
func printJSONResultsForTimeseries(result schema.TimeseriesResult, cfg *contract.Config) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		return writeJSONResultsForTimeseries(w, result)
	}, "Wrote JSON timeseries results")
}

// printCSVResultsForTimeseries handles opening the file and calling the CSV writer.
func printCSVResultsForTimeseries(result schema.TimeseriesResult, cfg *contract.Config, fmtFloat func(float64) string) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()
		return writeCSVResultsForTimeseries(csvWriter, result, fmtFloat)
	}, "Wrote CSV timeseries results")
}

// printTimeseriesTable prints the timeseries in a four-column table.
func printTimeseriesTable(result schema.TimeseriesResult, cfg *contract.Config, fmtFloat func(float64) string, duration time.Duration) error {
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
			contract.TruncatePath(p.Path, GetMaxTablePathWidth(cfg)),
			p.Period,
			fmtFloat(p.Score),
			string(p.Mode),
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

	fmt.Printf("Timeseries analysis completed in %v with %d workers. Cache backend: %s\n", duration, cfg.Workers, cfg.CacheBackend)
	return nil
}
