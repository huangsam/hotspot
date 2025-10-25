package schema

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultLimit     = 10
	maxLimitDefault  = 1000
	defaultWorkers   = 4
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
	// Initialize with defaults. EndTime defaults to current time for analysis range.
	cfg := &Config{Workers: defaultWorkers, EndTime: time.Now()}

	// Define flags
	limit := flag.Int("limit", defaultLimit, fmt.Sprintf("Number of files to display (default: %d, max: %d)", defaultLimit, maxLimitDefault))
	filter := flag.String("filter", "", "Filter files by path prefix")
	startDate := flag.String("start", "", "Start date in ISO8601 format (e.g., 2023-01-01T00:00:00Z)")
	endDate := flag.String("end", "", "End date in ISO8601 format (defaults to current time)")
	workers := flag.Int("workers", defaultWorkers, "Number of concurrent workers")
	mode := flag.String("mode", "hot", "Scoring mode: hot, risk, complexity, or stale")
	exclude := flag.String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore (e.g. go.sum)")
	detail := flag.Bool("detail", false, "Print per-file metadata such as commit activity, contributor count, etc.")
	explain := flag.Bool("explain", false, "Print per-file component score breakdown (for debugging/tuning)")
	precision := flag.Int("precision", defaultPrecision, "Decimal precision for numeric columns: 1 or 2")
	output := flag.String("output", "text", "Output format: text or csv")
	csvFile := flag.String("csv-file", "", "Optional path to write CSV output directly (overrides stdout)")
	follow := flag.Bool("follow", false, "Re-run per-file analysis with --follow for the top -limit files (slower but handles renames)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <repo-path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// --- 1. Repository Path Validation ---
	if flag.NArg() != 1 {
		flag.Usage()
		return nil, fmt.Errorf("repository path is required")
	}
	cfg.RepoPath = flag.Arg(0)

	// --- 2. ResultLimit Validation ---
	if *limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0 (received %d)", *limit)
	}
	if *limit > maxLimitDefault {
		return nil, fmt.Errorf("limit cannot exceed %d files (received %d)", maxLimitDefault, *limit)
	}
	cfg.ResultLimit = *limit

	// --- 3. Workers Validation ---
	if *workers <= 0 {
		return nil, fmt.Errorf("workers must be greater than 0 (received %d)", *workers)
	}
	cfg.Workers = *workers

	// --- 4. Mode Validation ---
	validModes := map[string]bool{"hot": true, "risk": true, "complexity": true, "stale": true}
	cfg.Mode = strings.ToLower(*mode)
	if _, ok := validModes[cfg.Mode]; !ok {
		return nil, fmt.Errorf("invalid mode '%s'. Must be one of: hot, risk, complexity, stale", *mode)
	}

	// --- 5. Precision Validation ---
	if *precision < 1 || *precision > 2 {
		return nil, fmt.Errorf("precision must be 1 or 2 (received %d)", *precision)
	}
	cfg.Precision = *precision

	// --- 6. Output Format Validation ---
	cfg.Output = strings.ToLower(*output)
	if cfg.Output != "text" && cfg.Output != "csv" {
		return nil, fmt.Errorf("invalid output format '%s'. Must be 'text' or 'csv'", *output)
	}
	cfg.CSVFile = *csvFile // CSVFile is just a path, no complex validation needed here.

	// --- 7. Date Parsing and Time Range Validation ---

	// Parse start date if provided
	if *startDate != "" {
		t, err := time.Parse(time.RFC3339, *startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start date format for '%s'. Must be RFC3339 (e.g., 2023-01-01T00:00:00Z): %v", *startDate, err)
		}
		cfg.StartTime = t
	}

	// Parse end date if provided
	if *endDate != "" {
		t, err := time.Parse(time.RFC3339, *endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end date format for '%s'. Must be RFC3339: %v", *endDate, err)
		}
		cfg.EndTime = t
	}

	// Check time range consistency (Start before End)
	if !cfg.StartTime.IsZero() && !cfg.EndTime.IsZero() && cfg.StartTime.After(cfg.EndTime) {
		return nil, fmt.Errorf("start time (%s) cannot be after end time (%s)", cfg.StartTime.Format(time.RFC3339), cfg.EndTime.Format(time.RFC3339))
	}

	// --- 8. Remaining Config Assignment & Excludes Processing ---

	cfg.PathFilter = *filter
	cfg.Detail = *detail
	cfg.Explain = *explain
	cfg.Follow = *follow

	// Excludes processing (using standard strings.Split)
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
		parts := strings.SplitSeq(*exclude, ",") // Using standard Split
		for p := range parts {
			trimmedP := strings.TrimSpace(p)
			if trimmedP != "" {
				cfg.Excludes = append(cfg.Excludes, trimmedP)
			}
		}
	}

	return cfg, nil
}
