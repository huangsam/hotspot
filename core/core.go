// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/viper"
)

// ExecutorFunc defines the function signature for executing different analysis modes.
type ExecutorFunc func(ctx context.Context, cfg *internal.Config) error

// ExecuteHotspotFiles runs the file-level analysis and prints results to stdout.
// It serves as the main entry point for the 'files' mode.
func ExecuteHotspotFiles(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()
	output, err := runSingleAnalysisCore(ctx, cfg, client)
	if err != nil {
		return err
	}
	resultsToRank := output.FileResults
	if cfg.Follow && len(resultsToRank) > 0 {
		rankedForFollow := rankFiles(resultsToRank, cfg.ResultLimit)
		resultsToRank = runFollowPass(ctx, cfg, client, rankedForFollow, output.AggregateOutput)
	}
	ranked := rankFiles(resultsToRank, cfg.ResultLimit)
	duration := time.Since(start)
	return internal.PrintFileResults(ranked, cfg, duration)
}

// ExecuteHotspotFolders runs the folder-level analysis and prints results to stdout.
// It serves as the main entry point for the 'folders' mode.
func ExecuteHotspotFolders(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()
	output, err := runSingleAnalysisCore(ctx, cfg, client)
	if err != nil {
		return err
	}
	folderResults := aggregateAndScoreFolders(cfg, output.FileResults)
	ranked := rankFolders(folderResults, cfg.ResultLimit)
	duration := time.Since(start)
	return internal.PrintFolderResults(ranked, cfg, duration)
}

// ExecuteHotspotCompare runs two file-level analyses (Base and Target)
// based on Git references and computes the delta results.
func ExecuteHotspotCompare(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()

	// Print single header for the comparison
	internal.LogCompareHeader(cfg)

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef)
	if err != nil {
		return err
	}
	comparisonResult := compareFileResults(baseOutput.FileResults, targetOutput.FileResults, cfg.ResultLimit, cfg.Mode)
	duration := time.Since(start)
	return internal.PrintComparisonResults(comparisonResult, cfg, duration)
}

// ExecuteHotspotCompareFolders runs two folder-level analyses (Base and Target)
// based on Git references and computes the delta results.
// It follows the same pattern as ExecuteHotspotCompare but aggregates to folders
// before performing the comparison.
func ExecuteHotspotCompareFolders(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()
	client := internal.NewLocalGitClient()

	// Print single header for the comparison
	internal.LogCompareHeader(cfg)

	baseOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.BaseRef)
	if err != nil {
		return err
	}
	targetOutput, err := runCompareAnalysisForRef(ctx, cfg, client, cfg.TargetRef)
	if err != nil {
		return err
	}
	comparisonResult := compareFolderMetrics(baseOutput.FolderResults, targetOutput.FolderResults, cfg.ResultLimit, cfg.Mode)
	duration := time.Since(start)
	return internal.PrintComparisonResults(comparisonResult, cfg, duration)
}

// ExecuteHotspotTimeseries runs multiple analyses over disjoint time windows for a specific path.
func ExecuteHotspotTimeseries(ctx context.Context, cfg *internal.Config) error {
	start := time.Now()

	// Get timeseries-specific parameters from Viper
	path := viper.GetString("path")
	intervalStr := viper.GetString("interval")
	numPoints := viper.GetInt("points")

	if path == "" {
		return errors.New("--path is required")
	}
	if intervalStr == "" {
		return errors.New("--interval is required")
	}
	if numPoints < 2 {
		return errors.New("--points must be at least 2")
	}

	interval, err := internal.ParseLookbackDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	numWindows := numPoints
	windowSize := interval / time.Duration(numWindows)
	now := time.Now()
	client := internal.NewLocalGitClient()

	// Normalize and validate the path relative to repo root
	normalizedPath, err := internal.NormalizeTimeseriesPath(cfg.RepoPath, path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists and determine if it's a file or folder
	fullPath := filepath.Join(cfg.RepoPath, normalizedPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", normalizedPath)
	}
	isFolder := info.IsDir()

	// Print single header for the entire timeseries analysis
	internal.LogTimeseriesHeader(cfg, interval, numPoints)

	var timeseriesPoints []schema.TimeseriesPoint

	for i := range numWindows {
		end := now.Add(-time.Duration(i) * windowSize)
		startTime := end.Add(-windowSize)
		cfgWindow := cfg.CloneWithTimeWindow(startTime, end)

		var score float64
		var owners []string
		if isFolder {
			suppressCtx := withSuppressHeader(ctx, true)
			output, err := runSingleAnalysisCore(suppressCtx, cfgWindow, client) // suppress individual headers
			if err != nil {
				// If no data in this window, score is 0
				score = 0
				owners = []string{} // Empty slice instead of nil
			} else {
				folderResults := aggregateAndScoreFolders(cfgWindow, output.FileResults)
				for _, fr := range folderResults {
					if fr.Path == normalizedPath {
						score = fr.Score
						owners = fr.Owners
						break
					}
				}
				// If path not found, owners remains empty slice
			}
		} else {
			suppressCtx := withSuppressHeader(ctx, true)
			output, err := runSingleAnalysisCore(suppressCtx, cfgWindow, client) // suppress individual headers
			if err != nil {
				// If no data in this window, score is 0
				score = 0
				owners = []string{} // Empty slice instead of nil
			} else {
				for _, fr := range output.FileResults {
					if fr.Path == normalizedPath {
						score = fr.Score
						owners = fr.Owners
						break
					}
				}
				// If path not found, owners remains empty slice
			}
		}

		// Generate period label
		windowDays := int(windowSize.Hours() / 24)
		var period string
		if i == 0 {
			period = fmt.Sprintf("Current (%dd)", windowDays)
		} else {
			startAgo := i * windowDays
			endAgo := (i + 1) * windowDays
			period = fmt.Sprintf("%dd to %dd Ago", startAgo, endAgo)
		}

		timeseriesPoints = append(timeseriesPoints, schema.TimeseriesPoint{
			Period: period,
			Score:  score,
			Path:   normalizedPath,
			Owners: owners,
			Mode:   cfg.Mode,
		})
	}

	result := schema.TimeseriesResult{Points: timeseriesPoints}
	duration := time.Since(start)
	return internal.PrintTimeseriesResults(result, cfg, duration)
}

// loadActiveWeights loads custom weights from the config file if available.
// Returns nil if no custom weights are found or if config loading fails.
func loadActiveWeights() (map[string]map[string]float64, error) {
	// Set up Viper for config loading (similar to main.go initConfig)
	viper.SetConfigName(".hotspot")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")
	viper.SetEnvPrefix("HOTSPOT")
	viper.AutomaticEnv()

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// No config file found, return nil (use defaults)
			return nil, nil
		}
		// Other error reading config, return the error
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal just the weights part
	var config struct {
		Weights internal.WeightsRawInput `mapstructure:"weights"`
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Process the weights similar to processCustomWeights
	activeWeights := make(map[string]map[string]float64)
	modes := []string{schema.StaleMode, schema.RiskMode, schema.HotMode, schema.ComplexityMode}
	modeWeights := map[string]*internal.ModeWeightsRaw{
		schema.StaleMode:      config.Weights.Stale,
		schema.RiskMode:       config.Weights.Risk,
		schema.HotMode:        config.Weights.Hot,
		schema.ComplexityMode: config.Weights.Complexity,
	}

	for _, mode := range modes {
		rawMode := modeWeights[mode]
		if rawMode == nil {
			continue
		}

		modeMap := make(map[string]float64)

		if rawMode.InvRecent != nil {
			modeMap[schema.BreakdownInvRecent] = *rawMode.InvRecent
		}
		if rawMode.Size != nil {
			modeMap[schema.BreakdownSize] = *rawMode.Size
		}
		if rawMode.Age != nil {
			modeMap[schema.BreakdownAge] = *rawMode.Age
		}
		if rawMode.Commits != nil {
			modeMap[schema.BreakdownCommits] = *rawMode.Commits
		}
		if rawMode.Contributors != nil {
			modeMap[schema.BreakdownContrib] = *rawMode.Contributors
		}
		if rawMode.InvContributors != nil {
			modeMap[schema.BreakdownInvContrib] = *rawMode.InvContributors
		}
		if rawMode.Churn != nil {
			modeMap[schema.BreakdownChurn] = *rawMode.Churn
		}
		if rawMode.Gini != nil {
			modeMap[schema.BreakdownGini] = *rawMode.Gini
		}
		if rawMode.LOC != nil {
			modeMap[schema.BreakdownLOC] = *rawMode.LOC
		}
		if rawMode.LowRecent != nil {
			modeMap[schema.BreakdownLowRecent] = *rawMode.LowRecent
		}

		if len(modeMap) > 0 {
			activeWeights[mode] = modeMap
		}
	}

	return activeWeights, nil
}

// ExecuteHotspotMetrics displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func ExecuteHotspotMetrics() error {
	// Load active weights from config file if available
	activeWeights, err := loadActiveWeights()
	if err != nil {
		// If we can't load config, just show defaults
		activeWeights = nil
	}
	return internal.PrintMetricsDefinitions(activeWeights)
}
