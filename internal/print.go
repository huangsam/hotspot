package internal

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

const maxPathWidth = 40

// PrintResults outputs the analysis results in a formatted table.
// For each file it shows rank, path (truncated if needed), importance score,
// criticality label, and all individual metrics that contribute to the score.
func PrintResults(files []schema.FileMetrics, cfg *schema.Config) {
	explain := cfg.Explain
	precision := cfg.Precision
	outFmt := cfg.Output

	// helper format strings for numbers
	numFmt := "%.*f"
	intFmt := "%d"
	// closure to format floats with the configured precision
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, precision, v)
	}

	// If CSV output requested, skip printing the human-readable table
	if outFmt == "csv" {
		file := selectOutputFile(cfg)
		w := csv.NewWriter(file)
		writeCSVResults(w, files, fmtFloat, intFmt)
		w.Flush()
		if file != os.Stdout {
			_ = file.Close()
			fmt.Fprintf(os.Stderr, "wrote CSV to %s\n", cfg.CSVFile)
		}
		return
	}

	// Define columns and initial header names
	headers := []string{"Rank", "File", "Score", "Label", "Contrib", "Commits", "Size(KB)", "Age(d)", "Churn", "Gini", "First Commit"}

	// Compute column widths from headers and data
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for idx, f := range files {
		// Rank width
		rankStr := strconv.Itoa(idx + 1)
		if len(rankStr) > widths[0] {
			widths[0] = len(rankStr)
		}
		// File path width
		p := truncatePath(f.Path, maxPathWidth)
		if len(p) > widths[1] {
			widths[1] = len(p)
		}
		// Score
		s := fmt.Sprintf("%.1f", f.Score)
		if len(s) > widths[2] {
			widths[2] = len(s)
		}
		// Label
		lbl := getTextLabel(f.Score)
		if len(lbl) > widths[3] {
			widths[3] = len(lbl)
		}
		// Other numeric columns
		nums := []string{
			fmt.Sprintf("%d", f.UniqueContributors),
			fmt.Sprintf("%d", f.Commits),
			fmt.Sprintf("%.1f", float64(f.SizeBytes)/1024.0),
			fmt.Sprintf("%d", f.AgeDays),
			fmt.Sprintf("%d", f.Churn),
			fmt.Sprintf("%.2f", f.Gini),
			f.FirstCommit.Format("2006-01-02"),
		}
		for i, n := range nums {
			col := i + 4 // starts at Contrib column
			if len(n) > widths[col] {
				widths[col] = len(n)
			}
		}
	}

	// Build format string dynamically
	fmts := []string{
		fmt.Sprintf("%%%ds", widths[0]),
		fmt.Sprintf("%%-%ds", widths[1]),
		fmt.Sprintf("%%%ds", widths[2]),
		fmt.Sprintf("%%-%ds", widths[3]),
	}
	// Numeric right-aligned columns
	for i := 4; i < len(headers)-1; i++ {
		fmts = append(fmts, fmt.Sprintf("%%%ds", widths[i]))
	}
	// Last column (date) left-aligned
	fmts = append(fmts, fmt.Sprintf("%%-%ds", widths[len(headers)-1]))

	// Compose header line
	var headerParts []string
	for i, h := range headers {
		headerParts = append(headerParts, fmt.Sprintf(fmts[i], h))
	}

	// Compose separator line
	sepParts := make([]string, len(headers))
	for i := range headers {
		sepParts[i] = strings.Repeat("-", widths[i])
	}

	// Print human-readable header and separator
	fmt.Println(strings.Join(headerParts, "  "))
	fmt.Println(strings.Join(sepParts, "  "))

	// Print rows
	for i, f := range files {
		p := truncatePath(f.Path, maxPathWidth)
		rowVals := []any{
			strconv.Itoa(i + 1),
			p,
			fmtFloat(f.Score),
			getTextLabel(f.Score),
			fmt.Sprintf(intFmt, f.UniqueContributors),
			fmt.Sprintf(intFmt, f.Commits),
			fmtFloat(float64(f.SizeBytes) / 1024.0),
			fmt.Sprintf(intFmt, f.AgeDays),
			fmt.Sprintf(intFmt, f.Churn),
			fmtFloat(f.Gini),
			f.FirstCommit.Format("2006-01-02"),
		}

		// Build formatted row using fmts
		var parts []string
		for j, rv := range rowVals {
			parts = append(parts, fmt.Sprintf(fmts[j], fmt.Sprint(rv)))
		}
		fmt.Println(strings.Join(parts, "  "))

		// Explain breakdown if requested
		if explain && len(f.Breakdown) > 0 {
			fmt.Println()
			fmt.Print("      Breakdown:")
			// print key/value pairs sorted by keys for deterministic output
			keys := []string{"contrib", "commits", "size", "age", "churn", "gini", "inv_contrib"}
			for _, k := range keys {
				if v, ok := f.Breakdown[k]; ok {
					fmt.Printf(" %s=%.1f%%", k, v)
				}
			}
			fmt.Println()
			fmt.Println()
		}
	}
}

// truncatePath truncates a file path to a maximum width with ellipsis prefix.
func truncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}
