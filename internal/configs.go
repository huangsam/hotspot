package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// Default values for configuration.
const (
	DefaultLookbackDays = 180
	DefaultResultLimit  = 25
	MaxResultLimit      = 1000
	DefaultPrecision    = 1
)

// DefaultWorkers is the default number of concurrent workers to use.
var DefaultWorkers = runtime.GOMAXPROCS(0)

// DateTimeFormat is the default date time representation.
var DateTimeFormat = time.RFC3339

// DateFormat is the default date representation.
var DateFormat = time.DateOnly

// ProfileConfig holds profiling settings
type ProfileConfig struct {
	Enabled bool
	Prefix  string
}

// Config holds the runtime configuration for the analysis.
// This struct remains the "final, validated" config.
type Config struct {
	RepoPath    string
	StartTime   time.Time
	EndTime     time.Time
	PathFilter  string
	ResultLimit int
	Workers     int
	Mode        string
	Excludes    []string
	Detail      bool
	Explain     bool
	Precision   int
	Output      string
	OutputFile  string
	Follow      bool
	Owner       bool

	CompareMode bool
	BaseRef     string
	TargetRef   string
	Lookback    time.Duration
}

// ConfigRawInput holds the raw inputs from all sources (flags, env, config file).
// Viper unmarshals into this struct.
type ConfigRawInput struct {
	// This is set manually from positional args, so no tag
	RepoPathStr string

	// --- Fields from rootCmd.PersistentFlags() ---
	Filter     string `mapstructure:"filter"`
	OutputFile string `mapstructure:"output-file"`
	Limit      int    `mapstructure:"limit"`
	Start      string `mapstructure:"start"`
	End        string `mapstructure:"end"`
	Workers    int    `mapstructure:"workers"`
	Mode       string `mapstructure:"mode"`
	Exclude    string `mapstructure:"exclude"`
	Precision  int    `mapstructure:"precision"`
	Output     string `mapstructure:"output"`
	Owner      bool   `mapstructure:"owner"`
	Detail     bool   `mapstructure:"detail"`

	// --- Fields from filesCmd.Flags() ---
	Explain bool `mapstructure:"explain"`
	Follow  bool `mapstructure:"follow"`

	// --- Fields from compareCmd.PersistentFlags() ---
	BaseRef   string `mapstructure:"base-ref"`
	TargetRef string `mapstructure:"target-ref"`
	Lookback  string `mapstructure:"lookback"`
}

// Clone returns a deep copy of the Config struct.
func (c *Config) Clone() *Config {
	// (Implementation unchanged)
	clone := *c
	if c.Excludes != nil {
		clone.Excludes = make([]string, len(c.Excludes))
		copy(clone.Excludes, c.Excludes)
	}
	return &clone
}

// CloneWithTimeWindow creates a copy of the Config and sets the new StartTime and EndTime.
func (c *Config) CloneWithTimeWindow(start time.Time, end time.Time) *Config {
	// (Implementation unchanged)
	clone := c.Clone()
	clone.StartTime = start
	clone.EndTime = end
	return clone
}

// ProcessAndValidate performs all complex parsing and validation on the raw inputs
// and updates the final Config struct.
func ProcessAndValidate(ctx context.Context, cfg *Config, client GitClient, input *ConfigRawInput) error {
	// All validation functions now read from 'input' and populate 'cfg'.
	if err := validateSimpleInputs(cfg, input); err != nil {
		return err
	}
	if err := processTimeRange(cfg, input); err != nil {
		return err
	}
	if err := processCompareMode(cfg, input); err != nil {
		return err
	}
	if err := resolveGitPathAndFilter(ctx, cfg, client, input); err != nil {
		return err
	}
	return nil
}

// validateSimpleInputs processes and validates all non-path related fields.
func validateSimpleInputs(cfg *Config, input *ConfigRawInput) error {
	// --- 0. Transfer simple non-validated fields from input -> cfg ---
	cfg.PathFilter = input.Filter
	cfg.OutputFile = input.OutputFile
	cfg.Detail = input.Detail
	cfg.Explain = input.Explain
	cfg.Owner = input.Owner
	cfg.Follow = input.Follow

	// --- 1. ResultLimit Validation ---
	if input.Limit <= 0 || input.Limit > MaxResultLimit {
		return fmt.Errorf("limit must be greater than 0 and cannot exceed %d (received %d)", MaxResultLimit, input.Limit)
	}
	cfg.ResultLimit = input.Limit

	// --- 2. Workers Validation ---
	if input.Workers <= 0 {
		return fmt.Errorf("workers must be greater than 0 (received %d)", input.Workers)
	}
	cfg.Workers = input.Workers

	// --- 3. Mode Validation ---
	validModes := map[string]bool{schema.HotMode: true, schema.RiskMode: true, schema.ComplexityMode: true, schema.StaleMode: true}
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
	defaults := []string{
		// (List of defaults unchanged)
		"Cargo.lock", "go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "composer.lock", "uv.lock",
		".min.js", ".min.css",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".mp4", ".mov", ".webm", ".mp3", ".ogg", ".pdf", ".webp",
		".json", ".csv",
		".md", "LICENSE",
		".DS_Store", ".gitignore",
		"dist/", "build/", "out/", "target/", "bin/",
	}
	cfg.Excludes = defaults // Set defaults first

	if input.Exclude != "" {
		parts := strings.SplitSeq(input.Exclude, ",") // Use simple Split
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
	now := time.Now()
	cfg.EndTime = now
	cfg.StartTime = cfg.EndTime.Add(-DefaultLookbackDays * 24 * time.Hour)

	parseAbsolute := func(s string) (time.Time, error) {
		return time.Parse(DateTimeFormat, s)
	}

	// --- Process Start Time ---
	if input.Start != "" {
		t, err := parseAbsolute(input.Start)
		if err == nil {
			cfg.StartTime = t
		} else {
			t, relErr := parseRelativeTime(input.Start, now)
			if relErr != nil {
				return fmt.Errorf("invalid start date format for '%s'. Expected absolute ISO8601 or 'N [units] ago': %v", input.Start, err)
			}
			cfg.StartTime = t
		}
	}

	// --- Process End Time ---
	if input.End != "" {
		t, err := parseAbsolute(input.End)
		if err == nil {
			cfg.EndTime = t
		} else {
			t, relErr := parseRelativeTime(input.End, now)
			if relErr != nil {
				return fmt.Errorf("invalid end date format for '%s'. Expected absolute ISO8601 or 'N [units] ago': %v", input.End, err)
			}
			cfg.EndTime = t
		}
	}

	// --- Final Validation ---
	if !cfg.StartTime.IsZero() && !cfg.EndTime.IsZero() && cfg.StartTime.After(cfg.EndTime) {
		return fmt.Errorf("start time (%s) cannot be after end time (%s)", cfg.StartTime.Format(DateTimeFormat), cfg.EndTime.Format(DateTimeFormat))
	}

	return nil
}

// processCompareMode handles the comparison references and lookback.
func processCompareMode(cfg *Config, input *ConfigRawInput) error {
	cfg.BaseRef = strings.TrimSpace(input.BaseRef)
	cfg.TargetRef = strings.TrimSpace(input.TargetRef)

	if cfg.BaseRef == "" && cfg.TargetRef == "" {
		cfg.CompareMode = false
		return nil
	}
	cfg.CompareMode = true

	if cfg.BaseRef == "" {
		return fmt.Errorf("must specify --base-ref when running the compare command")
	}
	if cfg.TargetRef == "" {
		cfg.TargetRef = "HEAD"
	}

	lookback, err := ParseLookbackDuration(input.Lookback)
	if err != nil {
		return err
	}
	cfg.Lookback = lookback

	return nil
}

// ProcessProfilingConfig handles the profiling flag and sets up profiling configuration.
func ProcessProfilingConfig(profile *ProfileConfig, profilePrefix string) error {
	if profilePrefix != "" {
		profile.Enabled = true
		profile.Prefix = profilePrefix
	}
	return nil
}

// resolveGitPathAndFilter resolves the Git repository path and set the implicit path filter.
func resolveGitPathAndFilter(ctx context.Context, cfg *Config, client GitClient, input *ConfigRawInput) error {
	// (Implementation unchanged, as it already reads from input.RepoPathStr)
	searchPath := input.RepoPathStr
	absSearchPath, err := filepath.Abs(searchPath)
	if err != nil {
		return err
	}
	absSearchPath = filepath.Clean(absSearchPath)

	info, statErr := os.Stat(absSearchPath)
	gitContextPath := absSearchPath
	if statErr == nil && !info.IsDir() {
		gitContextPath = filepath.Dir(absSearchPath)
	}

	gitRoot, err := client.GetRepoRoot(ctx, gitContextPath)
	if err != nil {
		return err
	}

	cfg.RepoPath = gitRoot

	if cfg.PathFilter != "" { // User-provided --filter flag takes precedence
		return nil
	}

	if absSearchPath != gitRoot {
		relativePath, err := filepath.Rel(gitRoot, absSearchPath)
		if err != nil {
			return err
		}

		if relativePath != "." {
			filter := relativePath
			if statErr == nil && info.IsDir() {
				filter += "/"
			}
			cfg.PathFilter = strings.ReplaceAll(filter, string(os.PathSeparator), "/")
		}
	}

	return nil
}
