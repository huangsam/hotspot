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
func recordFileAnalysis(ctx context.Context, scoringSettings config.ScoringSettings, analysisID int64, result *schema.FileResult) {
	BatchRecordFileAnalysis(ctx, scoringSettings, analysisID, []schema.FileResult{*result})
}

// BatchRecordFileAnalysis records multiple file metrics and scores to the database in one operation.
func BatchRecordFileAnalysis(ctx context.Context, scoringSettings config.ScoringSettings, analysisID int64, results []schema.FileResult) {
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
	batchResults := make([]schema.BatchFileResult, len(results))

	for i, result := range results {
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
			AgeDays:                result.AgeDays,
			GiniCoefficient:        result.Gini,
			FileOwner:              getOwnerString(result.Owners),
			RecencySignal:          result.RecencySignal,
			RecencyThresholdLow:    result.RecencyThresholdLow,
			RecencyThresholdHigh:   result.RecencyThresholdHigh,
		}

		// Compute scores
		allScores := result.AllScores

		// Record final scores
		scores := schema.FileScores{
			AnalysisTime:    now,
			HotScore:        allScores[schema.HotMode],
			RiskScore:       allScores[schema.RiskMode],
			ComplexityScore: allScores[schema.ComplexityMode],
			ROIScore:        allScores[schema.ROIMode],
			ScoreLabel:      string(scoringSettings.GetMode()),
			Reasoning:       result.Reasoning,
		}

		batchResults[i] = schema.BatchFileResult{
			Path:    result.Path,
			Metrics: metrics,
			Scores:  scores,
		}
	}

	// Record both metrics and scores in one operation
	if err := analysisStore.RecordFileResultsBatch(analysisID, batchResults); err != nil {
		logger.Warn(fmt.Sprintf("Batch analysis tracking failed for %d files", len(results)), err)
	}
}

// getOwnerString converts the owners slice to a string.
func getOwnerString(owners []string) string {
	if len(owners) == 0 {
		return ""
	}
	return owners[0] // Return the primary owner
}
