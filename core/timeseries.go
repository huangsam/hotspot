package core

import (
	"context"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
)

// Timeseries analysis constraints.
const (
	minCommits        = 30
	minLookback       = 3 * 30 * 24 * time.Hour // T_min: 3 months (temporal coverage constraint)
	maxSearchDuration = 6 * 30 * 24 * time.Hour // T_Max: 6 months (performance constraint for Git search)
)

// runTimeseriesAnalysis performs the core timeseries analysis logic.
func runTimeseriesAnalysis(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	client git.Client,
	normalizedPath string,
	isFolder bool,
	now time.Time,
	interval time.Duration,
	numPoints int,
	mgr iocache.CacheManager,
) []schema.TimeseriesPoint {
	var timeseriesPoints []schema.TimeseriesPoint
	currentEnd := now

	// Process each time window in reverse chronological order
	for i := range numPoints {
		// 1. Establish the End Time for this point/window (fixed step-back)
		if i > 0 {
			currentEnd = currentEnd.Add(-interval)
		}

		// 2. Calculate T_M_min: time to get minCommits, limited by maxSearchDuration
		// The Git search will be confined to [currentEnd - maxSearchDuration, currentEnd]
		commitTime, err := client.GetOldestCommitDateForPath(
			ctx,
			gitSettings.GetRepoPath(),
			normalizedPath,
			currentEnd,
			minCommits,
			maxSearchDuration, // Limiting the Git traversal depth
		)

		var lookbackFromCommits time.Duration
		if err != nil || commitTime.IsZero() {
			// If the search fails or finds fewer than minCommits within T_Max,
			// assume the path is sparse and use the larger of T_min or T_Max.
			lookbackFromCommits = max(minLookback, maxSearchDuration)
		} else {
			lookbackFromCommits = currentEnd.Sub(commitTime)
		}

		// 3. T_L is the max of the two constraints (T_min and T_M_min)
		lookbackDuration := max(minLookback, lookbackFromCommits)
		startTime := currentEnd.Add(-lookbackDuration)

		// Create dynamic time window settings
		gitWin := config.GitConfig{
			RepoPath:  gitSettings.GetRepoPath(),
			StartTime: startTime,
			EndTime:   currentEnd,
		}

		// --- Execute Analysis Core ---
		score, owners := analyzeTimeseriesPoint(ctx, gitWin, scoringSettings, client, normalizedPath, isFolder, mgr)
		// --- End Execute Analysis Core ---

		// 4. Generate period label
		var period string
		intervalDays := int(interval.Hours() / 24)
		if i == 0 {
			period = fmt.Sprintf("0-%dd ago", intervalDays)
		} else {
			startDays := intervalDays * i
			endDays := startDays + intervalDays
			period = fmt.Sprintf("%d-%dd ago", startDays, endDays)
		}

		timeseriesPoints = append(timeseriesPoints, schema.TimeseriesPoint{
			Period:   period,
			Start:    startTime,
			End:      currentEnd,
			Score:    score,
			Path:     normalizedPath,
			Owners:   owners,
			Mode:     scoringSettings.GetMode(),
			Lookback: lookbackDuration,
		})
	}

	return timeseriesPoints
}

// analyzeTimeseriesPoint performs the analysis for a single timeseries point.
func analyzeTimeseriesPoint(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	client git.Client,
	path string,
	isFolder bool,
	mgr iocache.CacheManager,
) (float64, []string) {
	ac := &AnalysisContext{
		Context: WithSuppressHeader(ctx), Git: gitSettings, Scoring: scoringSettings,
		Runtime: config.RuntimeConfig{Workers: 1}, Output: config.OutputConfig{ResultLimit: 10},
		Compare: config.CompareConfig{}, Client: client, Mgr: mgr,
		TargetRef: "HEAD",
	}

	pCfg := pipelineConfig{withTrackedAnalysis: true, withFolderAggregation: isFolder}
	if err := executePipeline(ac, pCfg); err != nil {
		// If no data in this window (e.g. no commits), score is 0
		return 0, []string{}
	}

	// Extract result
	if isFolder {
		for _, fr := range ac.FolderResults {
			if fr.Path == path {
				return fr.Score, fr.Owners
			}
		}
	} else {
		for _, fr := range ac.FileResults {
			if fr.Path == path {
				return fr.ModeScore, fr.Owners
			}
		}
	}
	return 0, []string{}
}
