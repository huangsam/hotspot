package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// WriteTimeseriesResults outputs the timeseries results, dispatching based on the output format configured.
func WriteTimeseriesResults(result schema.TimeseriesResult, cfg *contract.Config, duration time.Duration) error {
	// Create formatters using helper
	fmtFloat, _ := createFormatters(cfg.Precision)

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case schema.JSONOut:
		if err := writeTimeseriesJSONResults(result, cfg); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		if err := writeTimeseriesCSVResults(result, cfg, fmtFloat); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
			return writeTimeseriesTable(result, cfg, fmtFloat, duration, w)
		}, "Wrote timeseries table")
	}
	return nil
}

// writeTimeseriesJSONResults handles opening the file and calling the JSON writer.
func writeTimeseriesJSONResults(result schema.TimeseriesResult, cfg *contract.Config) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		return writeJSONResultsForTimeseries(w, result)
	}, "Wrote JSON timeseries results")
}

// writeTimeseriesCSVResults handles opening the file and calling the CSV writer.
func writeTimeseriesCSVResults(result schema.TimeseriesResult, cfg *contract.Config, fmtFloat func(float64) string) error {
	return writeWithFile(cfg.OutputFile, func(w io.Writer) error {
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()
		return writeCSVResultsForTimeseries(csvWriter, result, fmtFloat)
	}, "Wrote CSV timeseries results")
}

// writeTimeseriesTable prints the timeseries in a four-column table.
func writeTimeseriesTable(result schema.TimeseriesResult, cfg *contract.Config, fmtFloat func(float64) string, duration time.Duration, writer io.Writer) error {
	// Use os.Stdout, consistent with existing table printing
	table := tablewriter.NewWriter(writer)

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

	if _, err := fmt.Fprintf(writer, "Timeseries analysis completed in %v with %d workers. Cache backend: %s\n", duration, cfg.Workers, cfg.CacheBackend); err != nil {
		return err
	}
	return nil
}

// writeJSONResultsForTimeseries marshals the schema.TimeseriesResult to JSON and writes it.
func writeJSONResultsForTimeseries(w io.Writer, result schema.TimeseriesResult) error {
	return writeJSON(w, result)
}

// writeCSVResultsForTimeseries writes the schema.TimeseriesResult data to a CSV writer.
func writeCSVResultsForTimeseries(w *csv.Writer, result schema.TimeseriesResult, fmtFloat func(float64) string) error {
	// 1. Write Header Row
	header := []string{
		"path",
		"period",
		"start",
		"end",
		"score",
		"owners",
		"mode",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	// 2. Write Data Rows
	for _, p := range result.Points {
		ownersStr := ""
		if len(p.Owners) > 0 {
			ownersStr = strings.Join(p.Owners, "|")
		} else {
			ownersStr = ""
		}
		row := []string{
			p.Path,
			p.Period,
			p.Start.Format(contract.DateTimeFormat),
			p.End.Format(contract.DateTimeFormat),
			fmtFloat(p.Score),
			ownersStr,
			string(p.Mode),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
