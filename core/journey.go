package core

import (
	"context"
	"fmt"
	"math"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
)

// GetHotspotJourneyResults discovers the N most recent tags and runs successive
// compare_hotspots between each adjacent pair, returning a unified JourneyResult.
// transitions controls how many tag-to-tag steps to analyze (e.g. 3 = last 4 tags).
func GetHotspotJourneyResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager, transitions int) (schema.JourneyResult, error) {
	if transitions < 1 {
		transitions = 3
	}

	// Fetch enough tags to cover the requested transitions (+1 for the base of the first pair)
	tags, err := client.GetTags(ctx, cfg.Git.RepoPath, transitions+1)
	if err != nil {
		return schema.JourneyResult{}, fmt.Errorf("failed to list tags: %w", err)
	}
	if len(tags) < 2 {
		return schema.JourneyResult{}, fmt.Errorf("need at least 2 tags to generate a journey (found %d)", len(tags))
	}

	// Tags arrive newest-first; walk pairs in reverse so Steps are returned newest-first too.
	steps := make([]schema.JourneyStep, 0, len(tags)-1)
	summary := schema.JourneySummary{
		Mode: string(cfg.Scoring.Mode),
	}

	peakDelta := math.Inf(-1)

	for i := 0; i < len(tags)-1; i++ {
		targetRef := tags[i]
		baseRef := tags[i+1]

		stepCfg := cfg.Clone()
		stepCfg.Compare.BaseRef = targetRef // compare relative to the tag's own time window
		stepCfg.Compare.TargetRef = baseRef
		// Swap so base is the older tag
		stepCfg.Compare.BaseRef = baseRef
		stepCfg.Compare.TargetRef = targetRef
		stepCfg.Compare.Enabled = true

		result, _, err := GetHotspotCompareResults(ctx, stepCfg, client, mgr)
		if err != nil {
			return schema.JourneyResult{}, fmt.Errorf("comparison %s..%s failed: %w", baseRef, targetRef, err)
		}

		steps = append(steps, schema.JourneyStep{
			BaseRef:   baseRef,
			TargetRef: targetRef,
			Result:    result,
		})

		// Accumulate summary
		summary.TotalSteps++
		summary.TotalNewFiles += result.Summary.TotalNewFiles
		summary.TotalInactiveFiles += result.Summary.TotalInactiveFiles
		summary.NetScoreDelta += result.Summary.NetScoreDelta

		if result.Summary.NetScoreDelta > peakDelta {
			peakDelta = result.Summary.NetScoreDelta
			summary.PeakDeltaStep = fmt.Sprintf("%s..%s", baseRef, targetRef)
		}
	}

	return schema.JourneyResult{
		Summary: summary,
		Steps:   steps,
	}, nil
}
