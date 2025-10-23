package schema

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	maxLimitDefault  = 200
	defaultWorkers   = 8
	defaultLimit     = 10
	defaultPrecision = 1
)

// Config holds the runtime configuration for the analysis.
// It includes repository location, time range filters, and execution parameters.
type Config struct {
	RepoPath    string    // Absolute path to the Git repository
	StartTime   time.Time // Start of time range for commit analysis (zero = no limit)
	EndTime     time.Time // End of time range for commit analysis (zero = no limit)
	PathFilter  string    // Optional path prefix filter for files
	ResultLimit int       // Maximum number of files to show in results
	Workers     int       // Number of concurrent workers for analysis
	Mode        string    // Scoring mode: "hot" or "risk"
	Excludes    []string  // Path prefixes/suffixes to ignore
	Detail      bool      // If true, print per-file metadata
	Explain     bool      // If true, print per-file breakdown
	Precision   int       // Decimal precision for numeric columns (1 or 2)
	Output      string    // Output format: "text" (default) or "csv"
	CSVFile     string    // Optional path to write CSV output directly
	Follow      bool      // If true, re-run per-file analysis with `--follow` for the top -limit files
}

// ParseFlags processes command line arguments and returns a Config struct.
// It uses the standard flag package to handle options for controlling the analysis.
// Returns an error if required arguments are missing or invalid.
func ParseFlags() (*Config, error) {
	cfg := &Config{Workers: defaultWorkers, EndTime: time.Now()}

	// Define flags
	limit := flag.Int("limit", defaultLimit, fmt.Sprintf("Number of files to display (default: %d, max: %d)", defaultLimit, maxLimitDefault))
	filter := flag.String("filter", "", "Filter files by path prefix")
	startDate := flag.String("start", "", "Start date in ISO8601 format (e.g., 2023-01-01T00:00:00Z)")
	endDate := flag.String("end", "", "End date in ISO8601 format (defaults to current time)")
	workers := flag.Int("workers", defaultWorkers, fmt.Sprintf("Number of concurrent workers (default: %d)", defaultWorkers))
	mode := flag.String("mode", "hot", "Scoring mode: hot, risk, complexity, stale")
	exclude := flag.String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore (e.g. go.sum)")
	detail := flag.Bool("detail", false, "Print per-file metadata such as commit activity, contributor count, etc.")
	explain := flag.Bool("explain", false, "Print per-file component score breakdown (for debugging/tuning)")
	precision := flag.Int("precision", defaultPrecision, "Decimal precision for numeric columns (1 or 2)")
	output := flag.String("output", "text", "Output format: text (default) or csv")
	csvFile := flag.String("csv-file", "", "Optional path to write CSV output directly (overrides stdout)")
	follow := flag.Bool("follow", false, "Re-run per-file analysis with --follow for the top -limit files (slower but handles renames)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <repo-path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return nil, fmt.Errorf("repository path is required")
	}

	cfg.RepoPath = flag.Arg(0)
	if *limit > maxLimitDefault {
		return nil, fmt.Errorf("limit cannot exceed %d files", maxLimitDefault)
	}
	cfg.ResultLimit = *limit
	cfg.PathFilter = *filter
	cfg.Workers = *workers
	cfg.Mode = *mode
	cfg.Detail = *detail
	cfg.Explain = *explain

	// default excludes
	defaults := []string{
		// Dependency Lock File
		"Cargo.lock",        // Rust
		"go.sum",            // Go
		"package-lock.json", // JS/NPM
		"yarn.lock",         // JS/Yarn
		"pnpm-lock.yaml",    // JS/PNPM
		"composer.lock",     // PHP
		"uv.lock",           // Python

		// Generated Assets
		".min.js", ".min.css", // Minified JavaScript and CSS

		// Build Output Directories
		"dist/", "build/", "out/", "target/", "bin/",
	}
	cfg.Excludes = defaults
	if *exclude != "" {
		parts := strings.SplitSeq(*exclude, ",")
		for p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.Excludes = append(cfg.Excludes, p)
			}
		}
	}

	if *precision < 1 {
		*precision = 1
	}
	if *precision > 2 {
		*precision = 2
	}
	cfg.Precision = *precision
	cfg.Output = strings.ToLower(*output)
	cfg.CSVFile = *csvFile
	cfg.Follow = *follow

	// Parse start date if provided
	if *startDate != "" {
		t, err := time.Parse(time.RFC3339, *startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start date: %v", err)
		}
		cfg.StartTime = t
	}

	// Parse end date if provided
	if *endDate != "" {
		t, err := time.Parse(time.RFC3339, *endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end date: %v", err)
		}
		cfg.EndTime = t
	}

	return cfg, nil
}
