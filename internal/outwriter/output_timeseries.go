package outwriter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// WriteTimeseriesResults outputs the timeseries results, dispatching based on the output format configured.
func WriteTimeseriesResults(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	// Create formatters using helper
	fmtFloat, _ := createFormatters(output.GetPrecision())

	// Dispatcher: Handle different output formats
	switch output.GetFormat() {
	case schema.CSVOut:
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()
		if err := writeCSVResultsForTimeseries(csvWriter, result, fmtFloat); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		return writeTimeseriesTable(result, output, runtime, fmtFloat, duration, w)
	}
	return nil
}

// writeTimeseriesTable writes the timeseries in a four-column table.
func writeTimeseriesTable(result schema.TimeseriesResult, output config.OutputSettings, runtime config.RuntimeSettings, fmtFloat func(float64) string, duration time.Duration, writer io.Writer) error {
	table := tablewriter.NewWriter(writer)
	defer func() { _ = table.Close() }()

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
			truncatePath(p.Path, getMaxTablePathWidth(output)),
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

	if _, err := fmt.Fprintf(writer, "Timeseries analysis completed in %v with %d workers. Cache backend: %s\n", duration, runtime.GetWorkers(), runtime.GetCacheBackend()); err != nil {
		return err
	}
	return nil
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
			p.Start.Format(schema.DateTimeFormat),
			p.End.Format(schema.DateTimeFormat),
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
