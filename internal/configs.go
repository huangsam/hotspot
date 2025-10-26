package internal

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Default values for configuration.
const (
	DefaultLookbackDays = 90
	DefaultResultLimit  = 10
	MaxResultLimit      = 1000
	DefaultWorkers      = 4
	DefaultPrecision    = 1
)

// TimeFormat is the default time representation.
var TimeFormat = time.RFC3339

// Config holds the runtime configuration for the analysis.
// Fields that are set directly by simple flags remain the same (e.g., ResultLimit).
// Fields that require complex parsing (like dates and excludes) are set by the
// ProcessAndValidate function after flags are read.
type Config struct {
	RepoPath    string    // Absolute path to the Git repository (set by positional arg)
	StartTime   time.Time // Start of time range for commit analysis
	EndTime     time.Time // End of time range for commit analysis
	PathFilter  string    // Optional path prefix filter for files
	ResultLimit int       // Maximum number of files to show in results
	Workers     int       // Number of concurrent workers for analysis
	Mode        string    // Scoring mode: "hot" or "risk"
	Excludes    []string  // Path prefixes/suffixes to ignore (FINAL processed list)
	Detail      bool
	Explain     bool
	Precision   int
	Output      string
	CSVFile     string
	JSONFile    string
	Follow      bool
}

// ConfigRawInput holds the raw string inputs from flags that require parsing/validation.
// These fields are bound directly to Cobra's flags in main.go.
type ConfigRawInput struct {
	RepoPathStr  string
	StartTimeStr string
	EndTimeStr   string
	ExcludeStr   string
	Mode         string
	Output       string
	Precision    int
	ResultLimit  int
	Workers      int
}

// ProcessAndValidate performs all complex parsing and validation on the raw inputs
// and updates the final Config struct.
func ProcessAndValidate(cfg *Config, input *ConfigRawInput) error {
	// --- 1. ResultLimit Validation ---
	if input.ResultLimit <= 0 || input.ResultLimit > MaxResultLimit {
		return fmt.Errorf("limit must be greater than 0 and cannot exceed %d (received %d)", MaxResultLimit, input.ResultLimit)
	}
	cfg.ResultLimit = input.ResultLimit // Final assignment

	// --- 2. Workers Validation ---
	if input.Workers <= 0 {
		return fmt.Errorf("workers must be greater than 0 (received %d)", input.Workers)
	}
	cfg.Workers = input.Workers // Final assignment

	// --- 3. Mode Validation ---
	validModes := map[string]bool{"hot": true, "risk": true, "complexity": true, "stale": true}
	cfg.Mode = strings.ToLower(input.Mode)
	if _, ok := validModes[cfg.Mode]; !ok {
		return fmt.Errorf("invalid mode '%s'. must be hot, risk, complexity, stale", input.Mode)
	}

	// --- 4. Precision and Output Validation ---
	if input.Precision < 1 || input.Precision > 2 {
		return fmt.Errorf("precision must be 1 or 2 (received %d)", input.Precision)
	}
	cfg.Precision = input.Precision

	cfg.Output = strings.ToLower(input.Output)
	validOutputs := map[string]bool{"text": true, "csv": true, "json": true}
	if _, ok := validOutputs[cfg.Output]; !ok {
		return fmt.Errorf("invalid output format '%s'. must be text, csv, json", cfg.Output)
	}

	// --- 5. Date Parsing and Time Range Validation ---

	// Set defaults if strings are empty
	cfg.EndTime = time.Now()
	cfg.StartTime = cfg.EndTime.Add(-DefaultLookbackDays * 24 * time.Hour)

	if input.StartTimeStr != "" {
		t, err := time.Parse(time.RFC3339, input.StartTimeStr)
		if err != nil {
			return fmt.Errorf("invalid start date format for '%s'. must be RFC3339: %v", input.StartTimeStr, err)
		}
		cfg.StartTime = t
	}
	if input.EndTimeStr != "" {
		t, err := time.Parse(time.RFC3339, input.EndTimeStr)
		if err != nil {
			return fmt.Errorf("invalid end date format for '%s'. must be RFC3339: %v", input.EndTimeStr, err)
		}
		cfg.EndTime = t
	}
	if !cfg.StartTime.IsZero() && !cfg.EndTime.IsZero() && cfg.StartTime.After(cfg.EndTime) {
		return fmt.Errorf("start time (%s) cannot be after end time (%s)", cfg.StartTime.Format(time.RFC3339), cfg.EndTime.Format(time.RFC3339))
	}

	// --- 6. Excludes Processing ---
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

	if input.ExcludeStr != "" {
		parts := strings.SplitSeq(input.ExcludeStr, ",")
		for p := range parts {
			trimmedP := strings.TrimSpace(p)
			if trimmedP != "" {
				cfg.Excludes = append(cfg.Excludes, trimmedP)
			}
		}
	}

	// --- 7. Git Repository Path Resolution and Implicit PathFilter ---

	searchPath := input.RepoPathStr // This is the CWD or the user's positional argument

	// 7a. Find the absolute Git repository root path
	// We run the command using the user's path as context
	rootOut, err := RunGitCommand(searchPath, "rev-parse", "--show-toplevel")
	if err != nil {
		// If git fails to find the repo root, return the error
		return err
	}

	// Set cfg.RepoPath to the absolute Git root (CRUCIAL for consistent Git command execution)
	gitRoot := strings.TrimSpace(string(rootOut))
	cfg.RepoPath = gitRoot

	// 7b. Calculate and apply the Implicit PathFilter
	// Only proceed if the user hasn't already set the explicit --filter flag
	if cfg.PathFilter == "" {
		// Get the absolute path of the directory being analyzed (searchPath)
		absSearchPath, err := filepath.Abs(searchPath)
		if err != nil {
			return err
		}

		// Ensure paths are clean before comparison (removes trailing slashes, etc.)
		absSearchPath = filepath.Clean(absSearchPath)

		// Only set an implicit filter if the execution path is NOT the repo root
		if absSearchPath != gitRoot {

			// Calculate the path relative to the Git root
			relativePath, err := filepath.Rel(gitRoot, absSearchPath)
			if err != nil {
				return err
			}

			// If the relative path is not '.', set it as the filter
			if relativePath != "." {
				// We must use '/' for path filtering, as Git paths are always '/' based.
				// The PathFilter is used to filter paths returned by Git.
				cfg.PathFilter = relativePath + "/"
			}
		}
	}

	return nil
}
