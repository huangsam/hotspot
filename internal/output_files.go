package internal

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/schema"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintFileResults outputs the analysis results in a formatted table or exports them as CSV/JSON.
func PrintFileResults(files []schema.FileMetrics, cfg *Config) {
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
			LogFatal("Error writing JSON output", err)
		}
	case "csv":
		if err := printCSVResults(files, cfg, fmtFloat, intFmt); err != nil {
			LogFatal("Error writing CSV output", err)
		}
	default:
		// Default to human-readable table
		if err := printTableResults(files, cfg, fmtFloat, intFmt); err != nil {
			LogFatal("Error writing table output", err)
		}
	}
}

// printJSONResults handles opening the file and calling the JSON writer.
func printJSONResults(files []schema.FileMetrics, cfg *Config) error {
	// Use the unified file selector defined in writers.go
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if err := writeJSONResults(file, files, cfg); err != nil {
		return err
	}

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote JSON to %s\n", cfg.OutputFile)
	}
	return nil
}

// printCSVResults handles opening the file and calling the CSV writer.
func printCSVResults(files []schema.FileMetrics, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	// Use the unified file selector defined in writers.go
	file, err := selectOutputFile(cfg.OutputFile)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	w := csv.NewWriter(file)
	if err := writeCSVResults(w, files, cfg, fmtFloat, intFmt); err != nil {
		return err
	}
	w.Flush()

	if file != os.Stdout {
		fmt.Fprintf(os.Stderr, "💾 Wrote CSV to %s\n", cfg.OutputFile)
	}
	return nil
}

// printTableResults generates and prints the human-readable table.
func printTableResults(files []schema.FileMetrics, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
	table := tablewriter.NewWriter(os.Stdout)

	// 1. Define Headers
	headers := []string{"Rank", "File", "Score", "Label"}
	if cfg.Detail {
		headers = append(headers, "Contrib", "Commits", "LOC", "Churn")
	}
	if cfg.Explain {
		headers = append(headers, "Explain")
	}
	if cfg.Owner {
		headers = append(headers, "Owner")
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
			strconv.Itoa(i + 1),                     // Rank
			truncatePath(f.Path, maxTablePathWidth), // File
			fmtFloat(f.Score),                       // Score
			getColorLabel(f.Score),                  // Label
		}
		if cfg.Detail {
			row = append(
				row,
				fmt.Sprintf(intFmt, f.UniqueContributors), // Contrib
				fmt.Sprintf(intFmt, f.Commits),            // Commits
				fmt.Sprintf(intFmt, f.LinesOfCode),        // LOC
				fmt.Sprintf(intFmt, f.Churn),              // Churn
			)
		}
		if cfg.Explain {
			topOnes := formatTopMetricContributors(&f)
			row = append(row, topOnes) // Breakdown explanation
		}
		if cfg.Owner {
			row = append(row, f.Owner) // File owner
		}
		data = append(data, row)
	}

	// 4. Render the table
	if err := table.Bulk(data); err != nil {
		return err
	}
	return table.Render()
}

// metricBreakdown holds a key-value pair from the Breakdown map representing a metric's contribution.
type metricBreakdown struct {
	Name  string  // e.g., "commits", "churn", "size"
	Value float64 // The percentage contribution to the score
}

const (
	topNMetrics          = 3
	metricContribMinimum = 0.5
)

// formatTopMetricContributors computes the top 3 metric components that contribute to the final score.
func formatTopMetricContributors(f *schema.FileMetrics) string {
	var metrics []metricBreakdown

	// 1. Filter and Convert Map to Slice
	for k, v := range f.Breakdown {
		// Only include meaningful metrics
		if v >= metricContribMinimum {
			metrics = append(metrics, metricBreakdown{
				Name:  k,
				Value: v, // This is the percentage contribution
			})
		}
	}

	if len(metrics) == 0 {
		return "Not applicable"
	}

	// 2. Sort the Slice by Value (Contribution %) in Descending Order
	// Metrics with the highest absolute percentage contribution come first.
	sort.Slice(metrics, func(i, j int) bool {
		// We compare the absolute value since some contributions might be negative
		// if the model is set up to penalize certain metrics.
		return math.Abs(metrics[i].Value) > math.Abs(metrics[j].Value)
	})

	// 3. Limit to Top 3 and Format the Output
	var parts []string
	limit := min(len(metrics), topNMetrics)

	for i := range limit {
		m := metrics[i]
		parts = append(parts, m.Name)
	}

	if len(parts) == 0 {
		return "No meaningful contributors"
	}
	return strings.Join(parts, " > ")
}
