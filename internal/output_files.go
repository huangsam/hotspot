package internal

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/schema"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// PrintFileResults outputs the analysis results in a formatted table or exports them as CSV/JSON.
func PrintFileResults(files []schema.FileResult, cfg *Config, duration time.Duration) error {
	// helper format strings and closure for number formatting
	numFmt := "%.*f"
	intFmt := "%d"
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, cfg.Precision, v)
	}

	// Dispatcher: Handle different output formats
	switch cfg.Output {
	case schema.JSONOut:
		if err := printJSONResults(files, cfg); err != nil {
			return fmt.Errorf("error writing JSON output: %w", err)
		}
	case schema.CSVOut:
		if err := printCSVResults(files, cfg, fmtFloat, intFmt); err != nil {
			return fmt.Errorf("error writing CSV output: %w", err)
		}
	default:
		// Default to human-readable table
		if err := printTableResults(files, cfg, fmtFloat, intFmt, duration); err != nil {
			return fmt.Errorf("error writing table output: %w", err)
		}
	}
	return nil
}

// printJSONResults handles opening the file and calling the JSON writer.
func printJSONResults(files []schema.FileResult, cfg *Config) error {
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
func printCSVResults(files []schema.FileResult, cfg *Config, fmtFloat func(float64) string, intFmt string) error {
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
func printTableResults(files []schema.FileResult, cfg *Config, fmtFloat func(float64) string, intFmt string, duration time.Duration) error {
	table := tablewriter.NewWriter(os.Stdout)

	// 1. Define Headers
	headers := []string{"Rank", "Path", "Score", "Label"}
	if cfg.Detail {
		headers = append(headers, "Contrib", "Commits", "LOC", "Churn", "Age")
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
				fmt.Sprintf(intFmt, f.AgeDays),            // Age
			)
		}
		if cfg.Explain {
			topOnes := formatTopMetricBreakdown(&f)
			row = append(row, topOnes) // Breakdown explanation
		}
		if cfg.Owner {
			row = append(row, formatOwners(f.Owners)) // Top 2 owners
		}
		data = append(data, row)
	}

	// 4. Render the table
	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}
	// Compute summary stats
	numFiles := len(files)
	totalCommits := 0
	totalChurn := 0
	for _, f := range files {
		totalCommits += f.Commits
		totalChurn += f.Churn
	}
	fmt.Printf("Showing top %d files (total commits: %d, total churn: %d)\n", numFiles, totalCommits, totalChurn)
	fmt.Printf("Analysis completed in %v using %d workers.\n", duration, cfg.Workers)
	return nil
}

// formatOwners formats the top owners as "S. Huang, J. Doe"
func formatOwners(owners []string) string {
	var abbreviated []string
	for _, owner := range owners {
		abbreviated = append(abbreviated, abbreviateName(owner))
	}
	return strings.Join(abbreviated, ", ")
}

// abbreviateName formats "Samuel Huang" to "Samuel H"
func abbreviateName(name string) string {
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		return fmt.Sprintf("%s %s", parts[0], string(parts[1][0]))
	}
	return name
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

// formatTopMetricBreakdown computes the top 3 metric components that contribute to the final score.
func formatTopMetricBreakdown(f *schema.FileResult) string {
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
