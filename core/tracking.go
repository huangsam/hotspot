package core

import (
	"context"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/schema"
)

// recordFileAnalysis records file metrics and scores to the database.
func recordFileAnalysis(ctx context.Context, scoringSettings config.ScoringSettings, analysisID int64, path string, result *schema.FileResult) {
	// Get the cache manager from context
	mgr := cacheManagerFromContext(ctx)
	if mgr == nil {
		return
	}

	analysisStore := mgr.GetAnalysisStore()
	if analysisStore == nil {
		return
	}

	now := time.Now()

	// Record raw git metrics
	metrics := schema.FileMetrics{
		AnalysisTime:           now,
		TotalCommits:           result.Commits,
		TotalChurn:             result.Churn,
		LinesAdded:             result.LinesAdded,
		LinesDeleted:           result.LinesDeleted,
		DecayedCommits:         result.DecayedCommits,
		DecayedChurn:           result.DecayedChurn,
		LinesOfCode:            result.LinesOfCode,
		ContributorCount:       result.UniqueContributors,
		RecentCommits:          result.RecentCommits,
		RecentChurn:            result.RecentChurn,
		RecentLinesAdded:       result.RecentLinesAdded,
		RecentLinesDeleted:     result.RecentLinesDeleted,
		RecentContributorCount: result.RecentContributors,
		AgeDays:                result.AgeDays, // Convert int to float64 for type compatibility with FileMetrics struct
		GiniCoefficient:        result.Gini,
		FileOwner:              getOwnerString(result.Owners),
	}

	// Compute all four scoring modes
	allScores := result.AllScores

	// Record final scores
	scores := schema.FileScores{
		AnalysisTime:    now,
		HotScore:        allScores[schema.HotMode],
		RiskScore:       allScores[schema.RiskMode],
		ComplexityScore: allScores[schema.ComplexityMode],
		StaleScore:      allScores[schema.StaleMode],
		ScoreLabel:      string(scoringSettings.GetMode()),
	}

	// Record both metrics and scores in one operation
	if err := analysisStore.RecordFileMetricsAndScores(analysisID, path, metrics, scores); err != nil {
		logTrackingError("RecordFileMetricsAndScores", path, err)
	}
}

// getOwnerString converts the owners slice to a string.
func getOwnerString(owners []string) string {
	if len(owners) == 0 {
		return ""
	}
	return owners[0] // Return the primary owner
}

// logTrackingError logs database tracking errors to stderr without disrupting analysis.
func logTrackingError(operation, path string, err error) {
	logger.Warn(fmt.Sprintf("Analysis tracking failed for %s on %s", operation, path), err)
}
