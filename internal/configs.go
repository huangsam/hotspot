package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Default values for configuration.
const (
	DefaultLookbackDays = 180
	DefaultResultLimit  = 25
	MaxResultLimit      = 1000
	DefaultWorkers      = 8
	DefaultPrecision    = 1
)

// DateTimeFormat is the default date time representation.
var DateTimeFormat = time.RFC3339

// DateFormat is the default date representation
var DateFormat = time.DateOnly

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
	OutputFile  string
	Follow      bool
	Owner       bool
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
	if err := validateSimpleInputs(cfg, input); err != nil {
		return err
	}
	if err := processTimeRange(cfg, input); err != nil {
		return err
	}
	if err := resolveGitPathAndFilter(cfg, input); err != nil {
		return err
	}
	return nil
}

// validateSimpleInputs processes and validates all non-path related fields.
func validateSimpleInputs(cfg *Config, input *ConfigRawInput) error {
	// --- 1. ResultLimit Validation ---
	if input.ResultLimit <= 0 || input.ResultLimit > MaxResultLimit {
		return fmt.Errorf("limit must be greater than 0 and cannot exceed %d (received %d)", MaxResultLimit, input.ResultLimit)
	}
	cfg.ResultLimit = input.ResultLimit

	// --- 2. Workers Validation ---
	if input.Workers <= 0 {
		return fmt.Errorf("workers must be greater than 0 (received %d)", input.Workers)
	}
	cfg.Workers = input.Workers

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

	// --- 6. Excludes Processing ---
	// (This logic remains here as it's a non-external string/list manipulation)
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

		// Media assets
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico",
		".mp4", ".mov", ".webm",
		".mp3", ".ogg",
		".pdf", ".webp",

		// Data assets
		".json", ".csv",

		// Documentation
		".md", "LICENSE",

		// Other assets
		".DS_Store", ".gitignore",

		// Build Output Directories
		"dist/", "build/", "out/", "target/", "bin/",
	}
	cfg.Excludes = defaults // Set defaults first

	if input.ExcludeStr != "" {
		// NOTE: strings.SplitSeq is likely not a standard Go function; assuming it's a typo for strings.Split or a custom function.
		parts := strings.SplitSeq(input.ExcludeStr, ",")
		for p := range parts {
			trimmedP := strings.TrimSpace(p)
			if trimmedP != "" {
				cfg.Excludes = append(cfg.Excludes, trimmedP)
			}
		}
	}

	return nil
}

// processTimeRange handles the complex date parsing and time range validation.
func processTimeRange(cfg *Config, input *ConfigRawInput) error {
	// Set defaults if strings are empty
	cfg.EndTime = time.Now()
	cfg.StartTime = cfg.EndTime.Add(-DefaultLookbackDays * 24 * time.Hour)

	if input.StartTimeStr != "" {
		t, err := time.Parse(DateTimeFormat, input.StartTimeStr)
		if err != nil {
			return fmt.Errorf("invalid start date format for '%s': %v", input.StartTimeStr, err)
		}
		cfg.StartTime = t
	}

	if input.EndTimeStr != "" {
		t, err := time.Parse(DateTimeFormat, input.EndTimeStr)
		if err != nil {
			return fmt.Errorf("invalid end date format for '%s': %v", input.EndTimeStr, err)
		}
		cfg.EndTime = t
	}

	if !cfg.StartTime.IsZero() && !cfg.EndTime.IsZero() && cfg.StartTime.After(cfg.EndTime) {
		return fmt.Errorf("start time (%s) cannot be after end time (%s)", cfg.StartTime.Format(DateTimeFormat), cfg.EndTime.Format(DateTimeFormat))
	}

	return nil
}

// resolveGitPathAndFilter resolves the Git repository path and set the implicit path filter.
func resolveGitPathAndFilter(cfg *Config, input *ConfigRawInput) error {
	// 1. Determine the absolute path of the user's input
	searchPath := input.RepoPathStr
	absSearchPath, err := filepath.Abs(searchPath)
	if err != nil {
		return err
	}
	absSearchPath = filepath.Clean(absSearchPath)

	// 2. Determine the path to use for the 'git -C' command
	// Check if the input is a file (or if stat fails, assume it might be a file)
	info, statErr := os.Stat(absSearchPath)

	// We use the directory containing the search path for the Git command's context.
	gitContextPath := absSearchPath
	if statErr == nil && !info.IsDir() {
		// If the path is a file, the Git context must be its parent directory.
		gitContextPath = filepath.Dir(absSearchPath)
	}
	// If it's a directory or the path doesn't exist yet, we use absSearchPath as is.

	// 7a. Find the absolute Git repository root path
	// We run the command using the safe gitContextPath
	rootOut, err := RunGitCommand(gitContextPath, "rev-parse", "--show-toplevel")
	if err != nil {
		// If git still fails (e.g., gitContextPath is not in a repo), return the error.
		return err
	}

	// Set cfg.RepoPath to the absolute Git root
	gitRoot := strings.TrimSpace(string(rootOut))
	cfg.RepoPath = gitRoot

	// 7b. Calculate and apply the Implicit PathFilter
	// The filter is still based on the *original* absSearchPath, not the context directory.
	if cfg.PathFilter != "" {
		return nil
	}

	// Only set an implicit filter if the execution path is NOT the repo root
	if absSearchPath != gitRoot {
		// Calculate the path relative to the Git root
		relativePath, err := filepath.Rel(gitRoot, absSearchPath)
		if err != nil {
			return err
		}

		if relativePath != "." {
			// Assume relativePath is the filter
			filter := relativePath

			// Check the original input path type to append the slash if it's a directory
			if statErr == nil && info.IsDir() {
				filter += "/"
			}

			// Normalize the filter to use Git-style forward slashes (/)
			cfg.PathFilter = strings.ReplaceAll(filter, string(os.PathSeparator), "/")
		}
	}

	return nil
}
