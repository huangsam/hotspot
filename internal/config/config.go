// Package config provides configuration management for the hotspot application.
package config

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/schema"
)

// --- Settings Interfaces (Strangler Fig Pattern) ---

// GitSettings defines requirements for repository and filtering configuration.
type GitSettings interface {
	GetRepoPath() string
	GetStartTime() time.Time
	GetEndTime() time.Time
	GetPathFilter() string
	GetExcludes() []string
	IsFollow() bool
}

// ScoringSettings defines requirements for algorithm and weight configuration.
type ScoringSettings interface {
	GetMode() schema.ScoringMode
	GetCustomWeights() map[schema.ScoringMode]map[schema.BreakdownKey]float64
	GetComputedWeights() map[schema.ScoringMode]map[schema.BreakdownKey]float64
	GetRiskThresholds() map[schema.ScoringMode]float64
}

// OutputSettings defines requirements for presentation and export configuration.
type OutputSettings interface {
	GetResultLimit() int
	GetPrecision() int
	GetFormat() schema.OutputMode
	GetOutputFile() string
	GetWidth() int
	IsUseColors() bool
	IsDetail() bool
	IsExplain() bool
	IsOwner() bool
}

// RuntimeSettings defines requirements for execution and persistence configuration.
type RuntimeSettings interface {
	GetWorkers() int
	GetCacheBackend() schema.DatabaseBackend
	GetCacheDBConnect() string
	GetAnalysisBackend() schema.DatabaseBackend
	GetAnalysisDBConnect() string
}

// ComparisonSettings defines requirements for reference comparison configuration.
type ComparisonSettings interface {
	IsEnabled() bool
	GetBaseRef() string
	GetTargetRef() string
	GetLookback() time.Duration
}

// TimeseriesSettings defines requirements for trend analysis configuration.
type TimeseriesSettings interface {
	GetPath() string
	GetInterval() time.Duration
	GetPoints() int
}

// Default values for configuration.
const (
	DefaultLookbackDays = 180
	DefaultResultLimit  = 25
	MaxResultLimit      = 1000
	DefaultPrecision    = 1
)

// CacheGranularity defines the time granularity for caching analysis results.
// This ensures consistent cache key generation and time window alignment across
// the application and tests.
const CacheGranularity = time.Hour

// DefaultWorkers is the default number of concurrent workers to use.
var DefaultWorkers = runtime.GOMAXPROCS(0)

// GitConfig holds repository and filtering settings.
type GitConfig struct {
	RepoPath   string
	StartTime  time.Time
	EndTime    time.Time
	PathFilter string
	Excludes   []string
	Follow     bool
}

// GetRepoPath returns the repository path.
func (c GitConfig) GetRepoPath() string { return c.RepoPath }

// GetStartTime returns the start time for analysis.
func (c GitConfig) GetStartTime() time.Time { return c.StartTime }

// GetEndTime returns the end time for analysis.
func (c GitConfig) GetEndTime() time.Time { return c.EndTime }

// GetPathFilter returns the path filter string.
func (c GitConfig) GetPathFilter() string { return c.PathFilter }

// GetExcludes returns the list of excluded paths.
func (c GitConfig) GetExcludes() []string { return c.Excludes }

// IsFollow returns whether to follow renames.
func (c GitConfig) IsFollow() bool { return c.Follow }

// ScoringConfig holds algorithm and weight settings.
type ScoringConfig struct {
	Mode            schema.ScoringMode
	CustomWeights   map[schema.ScoringMode]map[schema.BreakdownKey]float64
	ComputedWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64
	RiskThresholds  map[schema.ScoringMode]float64
}

// GetMode returns the current scoring mode.
func (c ScoringConfig) GetMode() schema.ScoringMode { return c.Mode }

// GetCustomWeights returns the map of custom scoring weights.
func (c ScoringConfig) GetCustomWeights() map[schema.ScoringMode]map[schema.BreakdownKey]float64 {
	return c.CustomWeights
}

// GetComputedWeights returns the map of computed scoring weights.
func (c ScoringConfig) GetComputedWeights() map[schema.ScoringMode]map[schema.BreakdownKey]float64 {
	return c.ComputedWeights
}

// GetRiskThresholds returns the map of risk thresholds for each mode.
func (c ScoringConfig) GetRiskThresholds() map[schema.ScoringMode]float64 { return c.RiskThresholds }

// OutputConfig holds presentation and export settings.
type OutputConfig struct {
	ResultLimit int
	Precision   int
	Format      schema.OutputMode
	OutputFile  string
	Width       int // Terminal width override (0 = auto-detect)
	UseColors   bool
	Detail      bool
	Explain     bool
	Owner       bool
}

// GetResultLimit returns the maximum number of results to return.
func (c OutputConfig) GetResultLimit() int { return c.ResultLimit }

// GetPrecision returns the decimal precision for scores.
func (c OutputConfig) GetPrecision() int { return c.Precision }

// GetFormat returns the output format mode.
func (c OutputConfig) GetFormat() schema.OutputMode { return c.Format }

// GetOutputFile returns the path to the output file.
func (c OutputConfig) GetOutputFile() string { return c.OutputFile }

// GetWidth returns the terminal width override.
func (c OutputConfig) GetWidth() int { return c.Width }

// IsUseColors returns whether to use colors in output.
func (c OutputConfig) IsUseColors() bool { return c.UseColors }

// IsDetail returns whether to show detailed breakdown.
func (c OutputConfig) IsDetail() bool { return c.Detail }

// IsExplain returns whether to show scoring explanations.
func (c OutputConfig) IsExplain() bool { return c.Explain }

// IsOwner returns whether to show ownership statistics.
func (c OutputConfig) IsOwner() bool { return c.Owner }

// RuntimeConfig holds execution and persistence settings.
type RuntimeConfig struct {
	Workers           int
	CacheBackend      schema.DatabaseBackend
	CacheDBConnect    string
	AnalysisBackend   schema.DatabaseBackend
	AnalysisDBConnect string
}

// GetWorkers returns the number of concurrent workers.
func (c RuntimeConfig) GetWorkers() int { return c.Workers }

// GetCacheBackend returns the cache database backend.
func (c RuntimeConfig) GetCacheBackend() schema.DatabaseBackend { return c.CacheBackend }

// GetCacheDBConnect returns the cache connection string.
func (c RuntimeConfig) GetCacheDBConnect() string { return c.CacheDBConnect }

// GetAnalysisBackend returns the analysis database backend.
func (c RuntimeConfig) GetAnalysisBackend() schema.DatabaseBackend {
	return c.AnalysisBackend
}

// GetAnalysisDBConnect returns the analysis connection string.
func (c RuntimeConfig) GetAnalysisDBConnect() string { return c.AnalysisDBConnect }

// CompareConfig holds settings for reference comparisons.
type CompareConfig struct {
	Enabled   bool
	BaseRef   string
	TargetRef string
	Lookback  time.Duration
}

// IsEnabled returns whether comparison is enabled.
func (c CompareConfig) IsEnabled() bool { return c.Enabled }

// GetBaseRef returns the base git reference for comparison.
func (c CompareConfig) GetBaseRef() string { return c.BaseRef }

// GetTargetRef returns the target git reference for comparison.
func (c CompareConfig) GetTargetRef() string { return c.TargetRef }

// GetLookback returns the time lookback for comparison.
func (c CompareConfig) GetLookback() time.Duration { return c.Lookback }

// TimeseriesConfig holds settings for trend analysis.
type TimeseriesConfig struct {
	Path     string
	Interval time.Duration
	Points   int
}

// GetPath returns the file or folder path for trend analysis.
func (c TimeseriesConfig) GetPath() string { return c.Path }

// GetInterval returns the time interval between points.
func (c TimeseriesConfig) GetInterval() time.Duration { return c.Interval }

// GetPoints returns the number of time points to analyze.
func (c TimeseriesConfig) GetPoints() int { return c.Points }

// Config holds the runtime configuration for the analysis.
// This struct remains the "final, validated" config.
type Config struct {
	Git        GitConfig
	Scoring    ScoringConfig
	Output     OutputConfig
	Runtime    RuntimeConfig
	Compare    CompareConfig
	Timeseries TimeseriesConfig
}

// RawInput holds the raw inputs from all sources (flags, env, config file).
// Viper unmarshals into this struct.
type RawInput struct {
	// This is set manually from positional args, so no tag
	RepoPathStr string

	// --- Fields from rootCmd.PersistentFlags() ---
	Filter            string `mapstructure:"filter"`
	OutputFile        string `mapstructure:"output-file"`
	Limit             int    `mapstructure:"limit"`
	Start             string `mapstructure:"start"`
	End               string `mapstructure:"end"`
	Workers           int    `mapstructure:"workers"`
	Mode              string `mapstructure:"mode"`
	Exclude           string `mapstructure:"exclude"`
	Precision         int    `mapstructure:"precision"`
	Output            string `mapstructure:"output"`
	Owner             bool   `mapstructure:"owner"`
	Detail            bool   `mapstructure:"detail"`
	Width             int    `mapstructure:"width"`
	CacheBackend      string `mapstructure:"cache-backend"`
	CacheDBConnect    string `mapstructure:"cache-db-connect"`
	AnalysisBackend   string `mapstructure:"analysis-backend"`
	AnalysisDBConnect string `mapstructure:"analysis-db-connect"`
	Color             string `mapstructure:"color"`

	// --- Fields from filesCmd.Flags() ---
	Explain bool `mapstructure:"explain"`
	Follow  bool `mapstructure:"follow"`

	// --- Fields from compareCmd.PersistentFlags() ---
	BaseRef   string `mapstructure:"base-ref"`
	TargetRef string `mapstructure:"target-ref"`
	Lookback  string `mapstructure:"lookback"`

	// --- Fields from timeseriesCmd.Flags() ---
	Path     string `mapstructure:"path"`
	Interval string `mapstructure:"interval"`
	Points   int    `mapstructure:"points"`

	// --- Fields from checkCmd.Flags() ---
	ThresholdsStr string `mapstructure:"thresholds-override"`

	// --- Custom weights from config file ---
	Weights WeightsRawInput `mapstructure:"weights"`

	// --- Risk thresholds from config file ---
	Thresholds ThresholdsRawInput `mapstructure:"thresholds"`
}

// Clone returns a deep copy of the Config struct.
func (c *Config) Clone() *Config {
	clone := *c
	if c.Git.Excludes != nil {
		clone.Git.Excludes = make([]string, len(c.Git.Excludes))
		copy(clone.Git.Excludes, c.Git.Excludes)
	}
	if c.Scoring.CustomWeights != nil {
		clone.Scoring.CustomWeights = make(map[schema.ScoringMode]map[schema.BreakdownKey]float64)
		for mode, modeMap := range c.Scoring.CustomWeights {
			clone.Scoring.CustomWeights[mode] = make(map[schema.BreakdownKey]float64)
			maps.Copy(clone.Scoring.CustomWeights[mode], modeMap)
		}
	}
	if c.Scoring.ComputedWeights != nil {
		clone.Scoring.ComputedWeights = make(map[schema.ScoringMode]map[schema.BreakdownKey]float64)
		for mode, modeMap := range c.Scoring.ComputedWeights {
			clone.Scoring.ComputedWeights[mode] = make(map[schema.BreakdownKey]float64)
			maps.Copy(clone.Scoring.ComputedWeights[mode], modeMap)
		}
	}
	if c.Scoring.RiskThresholds != nil {
		clone.Scoring.RiskThresholds = make(map[schema.ScoringMode]float64)
		maps.Copy(clone.Scoring.RiskThresholds, c.Scoring.RiskThresholds)
	}
	return &clone
}

// CloneWithTimeWindow creates a copy of the Config and sets the new StartTime and EndTime.
func (c *Config) CloneWithTimeWindow(start time.Time, end time.Time) *Config {
	clone := c.Clone()
	clone.Git.StartTime = start
	clone.Git.EndTime = end
	return clone
}

// GetAnalysisStartTime returns the configured start time, truncated to the caching granularity.
// This ensures consistent time window alignment across the application and tests.
func (c *Config) GetAnalysisStartTime() time.Time {
	return c.Git.StartTime.Truncate(CacheGranularity)
}

// GetAnalysisEndTime returns the configured end time, truncated to the caching granularity.
// This ensures consistent time window alignment across the application and tests.
func (c *Config) GetAnalysisEndTime() time.Time {
	return c.Git.EndTime.Truncate(CacheGranularity)
}

// ProcessAndValidate performs all complex parsing and validation on the raw inputs
// and updates the final Config struct.
func ProcessAndValidate(ctx context.Context, cfg *Config, client git.Client, input *RawInput) error {
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
	if err := processTimeseriesMode(cfg, input); err != nil {
		return err
	}
	if err := processCustomWeights(cfg, input); err != nil {
		return err
	}
	if err := processRiskThresholds(cfg, input); err != nil {
		return err
	}
	if err := ResolveGitPathAndFilter(ctx, cfg, client, input); err != nil {
		return err
	}
	return nil
}

// ValidateDatabaseConnectionString validates the format of database connection strings
// for MySQL and PostgreSQL backends.
func ValidateDatabaseConnectionString(backend schema.DatabaseBackend, connStr string) error {
	switch backend {
	case schema.SQLiteBackend, schema.NoneBackend:
		return nil
	case schema.MySQLBackend:
		if connStr == "" {
			return fmt.Errorf("database connection string is required for MySQL backend. Example: 'user:password@tcp(localhost:3306)/hotspot'")
		}
		if !strings.Contains(connStr, "@tcp(") {
			return fmt.Errorf("MySQL connection string must contain '@tcp(' for host:port specification. Format: 'user:password@tcp(host:port)/dbname'. Got: %q", connStr)
		}
		if !strings.Contains(connStr, "/") {
			return fmt.Errorf("MySQL connection string must contain '/' followed by database name. Format: 'user:password@tcp(host:port)/dbname'. Got: %q", connStr)
		}
	case schema.PostgreSQLBackend:
		if connStr == "" {
			return fmt.Errorf("database connection string is required for PostgreSQL backend. Example: 'postgres://user:password@localhost:5432/hotspot?sslmode=disable'")
		}
		if !strings.Contains(connStr, "host=") {
			return fmt.Errorf("PostgreSQL connection string must contain 'host=' parameter. Format: 'postgres://user:password@host:port/dbname?sslmode=disable'. Got: %q", connStr)
		}
		if !strings.Contains(connStr, "dbname=") {
			return fmt.Errorf("PostgreSQL connection string must contain 'dbname=' parameter. Format: 'postgres://user:password@host:port/dbname?sslmode=disable'. Got: %q", connStr)
		}
	}
	return nil
}

// validateBackendConfigs validates cache and analysis backend configurations.
func validateBackendConfigs(cfg *Config, input *RawInput) error {
	// --- Cache Backend Validation ---
	cfg.Runtime.CacheBackend = schema.DatabaseBackend(strings.ToLower(input.CacheBackend))
	if _, ok := schema.ValidDatabaseBackends[cfg.Runtime.CacheBackend]; !ok {
		return fmt.Errorf("invalid cache backend '%s'. Must be one of: sqlite (default), mysql, postgresql, none", input.CacheBackend)
	}
	cfg.Runtime.CacheDBConnect = input.CacheDBConnect
	if err := ValidateDatabaseConnectionString(cfg.Runtime.CacheBackend, cfg.Runtime.CacheDBConnect); err != nil {
		return err
	}

	// --- Analysis Backend Validation ---
	cfg.Runtime.AnalysisBackend = schema.DatabaseBackend(strings.ToLower(input.AnalysisBackend))
	if cfg.Runtime.AnalysisBackend != "" {
		if _, ok := schema.ValidDatabaseBackends[cfg.Runtime.AnalysisBackend]; !ok {
			return fmt.Errorf("invalid analysis backend '%s'. Must be one of: sqlite, mysql, postgresql, none", input.AnalysisBackend)
		}
		cfg.Runtime.AnalysisDBConnect = input.AnalysisDBConnect
		if err := ValidateDatabaseConnectionString(cfg.Runtime.AnalysisBackend, cfg.Runtime.AnalysisDBConnect); err != nil {
			return err
		}

		// Validate that cache and analysis use different databases
		if cfg.Runtime.CacheBackend == cfg.Runtime.AnalysisBackend && cfg.Runtime.CacheBackend != schema.NoneBackend {
			// For SQLite, resolve to actual file paths to catch default path conflicts
			if cfg.Runtime.CacheBackend == schema.SQLiteBackend {
				cacheDBPath := cfg.Runtime.CacheDBConnect
				if cacheDBPath == "" {
					cacheDBPath = contract.GetCacheDBFilePath()
				}
				analysisDBPath := cfg.Runtime.AnalysisDBConnect
				if analysisDBPath == "" {
					analysisDBPath = contract.GetAnalysisDBFilePath()
				}
				if cacheDBPath == analysisDBPath {
					return fmt.Errorf("cache and analysis storage must use different SQLite database files. Both resolve to %q", cacheDBPath)
				}
			}
		}
	}

	return nil
}

// validateSimpleInputs processes and validates all non-path related fields.
func validateSimpleInputs(cfg *Config, input *RawInput) error {
	// --- 0. Transfer simple non-validated fields from input -> cfg ---
	cfg.Git.PathFilter = input.Filter
	cfg.Output.OutputFile = input.OutputFile
	cfg.Output.Detail = input.Detail
	cfg.Output.Explain = input.Explain
	cfg.Output.Owner = input.Owner
	cfg.Git.Follow = input.Follow
	cfg.Output.Width = input.Width

	// Parse color flag
	colors, err := contract.ParseBoolString(input.Color)
	if err != nil {
		return fmt.Errorf("invalid --color value: %w", err)
	}
	cfg.Output.UseColors = colors

	// --- 1. ResultLimit Validation ---
	if input.Limit <= 0 || input.Limit > MaxResultLimit {
		return fmt.Errorf("--limit (%d) must be between 1 and %d. Limit controls how many results to display", input.Limit, MaxResultLimit)
	}
	cfg.Output.ResultLimit = input.Limit

	// --- 2. Workers Validation ---
	if input.Workers <= 0 {
		return fmt.Errorf("--workers (%d) must be greater than 0. Recommend 1-%d based on your CPU cores", input.Workers, runtime.NumCPU())
	}
	cfg.Runtime.Workers = input.Workers

	// --- 3. Mode Validation ---
	cfg.Scoring.Mode = schema.ScoringMode(strings.ToLower(input.Mode))
	if _, ok := schema.ValidScoringModes[cfg.Scoring.Mode]; !ok {
		return fmt.Errorf("invalid mode '%s'. Must be one of: hot (activity), risk (knowledge distribution), complexity (technical debt), stale (maintenance debt)", input.Mode)
	}

	// --- 4. Precision and Output Validation ---
	if input.Precision < 1 || input.Precision > 2 {
		return fmt.Errorf("--precision (%d) must be 1 or 2 (controls decimal places in output)", input.Precision)
	}
	cfg.Output.Precision = input.Precision

	cfg.Output.Format = schema.OutputMode(strings.ToLower(input.Output))
	if _, ok := schema.ValidOutputModes[cfg.Output.Format]; !ok {
		return fmt.Errorf("invalid output format '%s'. Must be one of: text (pretty table), csv (comma-separated), json (structured), parquet (analytics)", cfg.Output.Format)
	}

	// --- 5. Backend Validation ---
	if err := validateBackendConfigs(cfg, input); err != nil {
		return err
	}

	// --- 6. Excludes Processing ---
	defaults := []string{
		"Cargo.lock", "go.sum", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "composer.lock", "uv.lock",
		".min.js", ".min.css",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".mp4", ".mov", ".webm", ".mp3", ".ogg", ".pdf", ".webp",
		".json", ".csv",
		".md", "LICENSE",
		".DS_Store", ".gitignore",
		"dist/", "build/", "out/", "target/", "bin/",
	}
	cfg.Git.Excludes = defaults // Set defaults first

	if input.Exclude != "" {
		parts := strings.SplitSeq(input.Exclude, ",") // Use simple Split
		for p := range parts {
			trimmedP := strings.TrimSpace(p)
			if trimmedP != "" {
				cfg.Git.Excludes = append(cfg.Git.Excludes, trimmedP)
			}
		}
	}

	return nil
}

// processTimeRange handles the complex date parsing and time range validation.
func processTimeRange(cfg *Config, input *RawInput) error {
	now := time.Now()
	cfg.Git.EndTime = now
	cfg.Git.StartTime = cfg.Git.EndTime.Add(-DefaultLookbackDays * 24 * time.Hour)

	parseAbsolute := func(s string) (time.Time, error) {
		return time.Parse(schema.DateTimeFormat, s)
	}

	// --- Process Start Time ---
	if input.Start != "" {
		t, err := parseAbsolute(input.Start)
		if err == nil {
			cfg.Git.StartTime = t
		} else {
			t, relErr := schema.ParseRelativeTime(input.Start, now)
			if relErr != nil {
				return fmt.Errorf("invalid start date format for '%s'. Expected absolute ISO8601 or 'N [units] ago': %v", input.Start, err)
			}
			cfg.Git.StartTime = t
		}
	}

	// --- Process End Time ---
	if input.End != "" {
		t, err := parseAbsolute(input.End)
		if err == nil {
			cfg.Git.EndTime = t
		} else {
			t, relErr := schema.ParseRelativeTime(input.End, now)
			if relErr != nil {
				return fmt.Errorf("invalid end date format for '%s'. Expected absolute ISO8601 or 'N [units] ago': %v", input.End, err)
			}
			cfg.Git.EndTime = t
		}
	}

	// --- Final Validation ---
	if !cfg.Git.StartTime.IsZero() && !cfg.Git.EndTime.IsZero() && cfg.Git.StartTime.After(cfg.Git.EndTime) {
		return fmt.Errorf("start time (%s) cannot be after end time (%s)", cfg.Git.StartTime.Format(schema.DateTimeFormat), cfg.Git.EndTime.Format(schema.DateTimeFormat))
	}

	return nil
}

// RevalidateCompare re-parses and validates comparison parameters.
func RevalidateCompare(cfg *Config, lookbackStr string) error {
	if lookbackStr != "" {
		lookback, err := schema.ParseLookbackDuration(lookbackStr)
		if err != nil {
			return err
		}
		cfg.Compare.Lookback = lookback
	}
	if cfg.Compare.BaseRef == "" {
		return fmt.Errorf("--base-ref is required for compare")
	}
	return nil
}

// RevalidateTimeseries re-parses and validates timeseries parameters.
func RevalidateTimeseries(cfg *Config, intervalStr string) error {
	if intervalStr != "" {
		interval, err := schema.ParseLookbackDuration(intervalStr)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		cfg.Timeseries.Interval = interval
	}
	if cfg.Timeseries.Points < 1 && cfg.Timeseries.Points != 0 {
		return fmt.Errorf("--points must be at least 1")
	}
	return nil
}

// RevalidateTimeRange re-parses and validates start/end time range parameters.
// Both startStr and endStr accept ISO8601 absolute dates or relative expressions
// like "30d ago" or "6 months ago". Empty strings leave the existing cfg values
// unchanged, so callers can safely pass an empty string for either field.
func RevalidateTimeRange(cfg *Config, startStr, endStr string) error {
	now := time.Now()

	parseTime := func(s string) (time.Time, error) {
		t, err := time.Parse(schema.DateTimeFormat, s)
		if err == nil {
			return t, nil
		}
		return schema.ParseRelativeTime(s, now)
	}

	if startStr != "" {
		t, err := parseTime(startStr)
		if err != nil {
			return fmt.Errorf("invalid start time %q: expected ISO8601 or relative expression like '30d ago': %w", startStr, err)
		}
		cfg.Git.StartTime = t
	}

	if endStr != "" {
		t, err := parseTime(endStr)
		if err != nil {
			return fmt.Errorf("invalid end time %q: expected ISO8601 or relative expression like '7d ago': %w", endStr, err)
		}
		cfg.Git.EndTime = t
	}

	if !cfg.Git.StartTime.IsZero() && !cfg.Git.EndTime.IsZero() && cfg.Git.StartTime.After(cfg.Git.EndTime) {
		return fmt.Errorf("start time (%s) cannot be after end time (%s)",
			cfg.Git.StartTime.Format(schema.DateTimeFormat),
			cfg.Git.EndTime.Format(schema.DateTimeFormat))
	}

	return nil
}

// processCompareMode handles the comparison references and lookback.
func processCompareMode(cfg *Config, input *RawInput) error {
	cfg.Compare.BaseRef = strings.TrimSpace(input.BaseRef)
	cfg.Compare.TargetRef = strings.TrimSpace(input.TargetRef)

	if cfg.Compare.BaseRef == "" && cfg.Compare.TargetRef == "" {
		cfg.Compare.Enabled = false
		return nil
	}
	cfg.Compare.Enabled = true

	if cfg.Compare.BaseRef == "" {
		return fmt.Errorf("--base-ref is required for compare command. Example: hotspot compare files --base-ref main --target-ref feature")
	}
	if cfg.Compare.TargetRef == "" {
		cfg.Compare.TargetRef = "HEAD"
	}

	lookback, err := schema.ParseLookbackDuration(input.Lookback)
	if err != nil {
		return err
	}
	cfg.Compare.Lookback = lookback

	return nil
}

// processTimeseriesMode handles the timeseries parameters.
func processTimeseriesMode(cfg *Config, input *RawInput) error {
	cfg.Timeseries.Path = strings.TrimSpace(input.Path)
	cfg.Timeseries.Points = input.Points

	if input.Interval != "" {
		interval, err := schema.ParseLookbackDuration(input.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		cfg.Timeseries.Interval = interval
	}

	// Basic validation
	if cfg.Timeseries.Points < 1 && cfg.Timeseries.Points != 0 {
		return fmt.Errorf("--points must be at least 1")
	}

	return nil
}

// ProcessWeightsRawInput converts WeightsRawInput into the final weights map.
// If validateSum is true, it validates that weights for each mode sum to 1.0.
func ProcessWeightsRawInput(weights WeightsRawInput, validateSum bool) (map[schema.ScoringMode]map[schema.BreakdownKey]float64, error) {
	result := make(map[schema.ScoringMode]map[schema.BreakdownKey]float64)

	modes := []schema.ScoringMode{schema.StaleMode, schema.RiskMode, schema.HotMode, schema.ComplexityMode}
	modeWeights := map[schema.ScoringMode]*ModeWeightsRaw{
		schema.StaleMode:      weights.Stale,
		schema.RiskMode:       weights.Risk,
		schema.HotMode:        weights.Hot,
		schema.ComplexityMode: weights.Complexity,
	}

	// Process each mode's raw weights and validate sums if required.
	// Skip modes that are nil (not provided)
	for _, mode := range modes {
		rawMode := modeWeights[mode]
		if rawMode == nil {
			continue
		}

		modeMap := make(map[schema.BreakdownKey]float64)
		sum := 0.0

		if rawMode.InvRecent != nil {
			modeMap[schema.BreakdownInvRecent] = *rawMode.InvRecent
			sum += *rawMode.InvRecent
		}
		if rawMode.Size != nil {
			modeMap[schema.BreakdownSize] = *rawMode.Size
			sum += *rawMode.Size
		}
		if rawMode.Age != nil {
			modeMap[schema.BreakdownAge] = *rawMode.Age
			sum += *rawMode.Age
		}
		if rawMode.Commits != nil {
			modeMap[schema.BreakdownCommits] = *rawMode.Commits
			sum += *rawMode.Commits
		}
		if rawMode.Contributors != nil {
			modeMap[schema.BreakdownContrib] = *rawMode.Contributors
			sum += *rawMode.Contributors
		}
		if rawMode.InvContributors != nil {
			modeMap[schema.BreakdownInvContrib] = *rawMode.InvContributors
			sum += *rawMode.InvContributors
		}
		if rawMode.Churn != nil {
			modeMap[schema.BreakdownChurn] = *rawMode.Churn
			sum += *rawMode.Churn
		}
		if rawMode.Gini != nil {
			modeMap[schema.BreakdownGini] = *rawMode.Gini
			sum += *rawMode.Gini
		}
		if rawMode.LOC != nil {
			modeMap[schema.BreakdownLOC] = *rawMode.LOC
			sum += *rawMode.LOC
		}
		if rawMode.LowRecent != nil {
			modeMap[schema.BreakdownLowRecent] = *rawMode.LowRecent
			sum += *rawMode.LowRecent
		}

		// Only add to result if we have at least one weight
		if len(modeMap) > 0 {
			if validateSum && (sum < 0.999 || sum > 1.001) {
				return nil, fmt.Errorf("custom weights for mode %s must sum to 1.0, got %.3f", mode, sum)
			}
			result[mode] = modeMap
		}
	}

	return result, nil
}

// processCustomWeights converts the raw input into the final cfg.Scoring.CustomWeights map
// and validates that the provided weights for any mode sum up to 1.0.
// Also computes the final ComputedWeights for each mode.
func processCustomWeights(cfg *Config, input *RawInput) error {
	weights, err := ProcessWeightsRawInput(input.Weights, true)
	if err != nil {
		return err
	}
	cfg.Scoring.CustomWeights = weights

	// Compute final weights for each mode
	cfg.Scoring.ComputedWeights = make(map[schema.ScoringMode]map[schema.BreakdownKey]float64)
	for _, mode := range []schema.ScoringMode{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.StaleMode} {
		// Start with default weights
		defaultWeights := schema.GetDefaultWeights(mode)

		// Override with custom weights if provided
		modeWeights := make(map[schema.BreakdownKey]float64)
		maps.Copy(modeWeights, defaultWeights)

		if cfg.Scoring.CustomWeights != nil {
			if customModeWeights, ok := cfg.Scoring.CustomWeights[mode]; ok {
				maps.Copy(modeWeights, customModeWeights)
			}
		}

		cfg.Scoring.ComputedWeights[mode] = modeWeights
	}

	return nil
}

// processRiskThresholds converts the raw threshold input into the final cfg.Scoring.RiskThresholds map.
// If no thresholds are provided in the config, it initializes with default values (50.0 for all modes).
// Command-line --thresholds-override flag takes precedence over config file settings.
func processRiskThresholds(cfg *Config, input *RawInput) error {
	thresholds := make(map[schema.ScoringMode]float64)

	// Set defaults first (50.0 for all modes)
	thresholds[schema.HotMode] = 50.0
	thresholds[schema.RiskMode] = 50.0
	thresholds[schema.ComplexityMode] = 50.0
	thresholds[schema.StaleMode] = 50.0

	// Override with config file values if provided
	if input.Thresholds.Hot != nil {
		thresholds[schema.HotMode] = *input.Thresholds.Hot
	}
	if input.Thresholds.Risk != nil {
		thresholds[schema.RiskMode] = *input.Thresholds.Risk
	}
	if input.Thresholds.Complexity != nil {
		thresholds[schema.ComplexityMode] = *input.Thresholds.Complexity
	}
	if input.Thresholds.Stale != nil {
		thresholds[schema.StaleMode] = *input.Thresholds.Stale
	}

	// Override with command-line flag if provided (takes precedence)
	if input.ThresholdsStr != "" {
		parsedThresholds, err := parseRiskThresholdsString(input.ThresholdsStr)
		if err != nil {
			return fmt.Errorf("invalid --thresholds format: %w", err)
		}
		// Merge parsed values
		maps.Copy(thresholds, parsedThresholds)
	}

	// Validate thresholds
	for mode, threshold := range thresholds {
		if threshold < 0.0 || threshold > 100.0 {
			return fmt.Errorf("risk threshold for mode %s must be between 0.0 and 100.0 (received %.2f)", mode, threshold)
		}
	}

	cfg.Scoring.RiskThresholds = thresholds
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

// ResolveGitPathAndFilter resolves the Git repository path and set the implicit path filter.
func ResolveGitPathAndFilter(ctx context.Context, cfg *Config, client git.Client, input *RawInput) error {
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

	cfg.Git.RepoPath = gitRoot

	if cfg.Git.PathFilter != "" { // User-provided --filter flag takes precedence
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
			cfg.Git.PathFilter = strings.ReplaceAll(filter, string(os.PathSeparator), "/")
		}
	}

	return nil
}

// parseRiskThresholdsString parses a string like "hot:50,risk:60,complexity:70,stale:80"
// into a map of ScoringMode to float64.
func parseRiskThresholdsString(s string) (map[schema.ScoringMode]float64, error) {
	thresholds := make(map[schema.ScoringMode]float64)

	if s == "" {
		return thresholds, nil
	}

	parts := strings.SplitSeq(s, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		keyValue := strings.Split(part, ":")
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid threshold format '%s', expected 'mode:value'", part)
		}

		modeStr := strings.TrimSpace(keyValue[0])
		valueStr := strings.TrimSpace(keyValue[1])

		var mode schema.ScoringMode
		switch strings.ToLower(modeStr) {
		case "hot":
			mode = schema.HotMode
		case "risk":
			mode = schema.RiskMode
		case "complexity":
			mode = schema.ComplexityMode
		case "stale":
			mode = schema.StaleMode
		default:
			return nil, fmt.Errorf("invalid mode '%s', must be hot, risk, complexity, or stale", modeStr)
		}

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid threshold value '%s' for mode %s: %w", valueStr, mode, err)
		}

		thresholds[mode] = value
	}

	return thresholds, nil
}

// ProfileConfig holds profiling settings.
type ProfileConfig struct {
	Enabled bool
	Prefix  string
}

// WeightsRawInput holds the raw weight inputs from the config file.
type WeightsRawInput struct {
	Hot        *ModeWeightsRaw `mapstructure:"hot"`
	Risk       *ModeWeightsRaw `mapstructure:"risk"`
	Complexity *ModeWeightsRaw `mapstructure:"complexity"`
	Stale      *ModeWeightsRaw `mapstructure:"stale"`
}

// ModeWeightsRaw holds the raw factor weights for a single mode.
type ModeWeightsRaw struct {
	InvRecent       *float64 `mapstructure:"inv_recent"`
	Size            *float64 `mapstructure:"size"`
	Age             *float64 `mapstructure:"age"`
	Commits         *float64 `mapstructure:"commits"`
	Contributors    *float64 `mapstructure:"contrib"`
	InvContributors *float64 `mapstructure:"inv_contrib"`
	Churn           *float64 `mapstructure:"churn"`
	Gini            *float64 `mapstructure:"gini"`
	LOC             *float64 `mapstructure:"loc"`
	LowRecent       *float64 `mapstructure:"low_recent"`
}

// ThresholdsRawInput holds the raw risk thresholds from the config file.
type ThresholdsRawInput struct {
	Hot        *float64 `mapstructure:"hot"`
	Risk       *float64 `mapstructure:"risk"`
	Complexity *float64 `mapstructure:"complexity"`
	Stale      *float64 `mapstructure:"stale"`
}
