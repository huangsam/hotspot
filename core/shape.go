package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/schema"
)

// recommendPreset selects the best preset based on weighted signal strengths.
func recommendPreset(fileCount, uniqueContributors int, iacFileRatio float64) (schema.PresetName, []string) {
	var reasons []string
	thresh := schema.GetShapeThresholds()
	tmpl := schema.GetShapeTemplates()

	infraSignal := iacFileRatio / thresh.IaCRatio
	scaleSignal := float64(fileCount) / float64(thresh.FileCount)
	socialSignal := float64(uniqueContributors) / float64(thresh.ContributorCount)

	// 1. Check for Infra Dominance
	if infraSignal >= 1.0 {
		reasons = append(reasons, fmt.Sprintf(tmpl.Infra, iacFileRatio*100, infraSignal))
		return schema.PresetInfra, reasons
	}

	// 2. Check for Compounding Scale and Social complexity
	combinedSignal := scaleSignal + socialSignal
	if combinedSignal >= 1.2 {
		if scaleSignal >= 1.0 {
			reasons = append(reasons, fmt.Sprintf(tmpl.LargeFiles, fileCount, scaleSignal))
		}
		if socialSignal >= 1.0 {
			reasons = append(reasons, fmt.Sprintf(tmpl.LargeContributors, uniqueContributors, socialSignal))
		}
		// If it's the combination that pushed it over (neither is > 1.0), use the compounding reason
		if len(reasons) == 0 {
			reasons = append(reasons, fmt.Sprintf(tmpl.Compounding, combinedSignal))
		}
		return schema.PresetLarge, reasons
	}

	// 3. Check for Single Large Signal (Fallback)
	if scaleSignal >= 1.0 {
		reasons = append(reasons, fmt.Sprintf(tmpl.LargeFiles, fileCount, scaleSignal))
		return schema.PresetLarge, reasons
	}
	if socialSignal >= 1.0 {
		reasons = append(reasons, fmt.Sprintf(tmpl.LargeContributors, uniqueContributors, socialSignal))
		return schema.PresetLarge, reasons
	}

	reasons = append(reasons, tmpl.Small)
	return schema.PresetSmall, reasons
}

// ComputeRepoShape derives shape metrics from a file list and aggregate output.
func ComputeRepoShape(files []string, output *schema.AggregateOutput) schema.RepoShape {
	fileCount := len(files)

	// Total commits across all active files
	// Unique contributors across all files
	// Total churn for average calculation
	var totalCommits float64
	var totalChurn float64
	allContribs := make(map[string]struct{})
	activeFiles := 0

	for _, stat := range output.FileStats {
		totalCommits += float64(stat.Commits)
		totalChurn += float64(stat.Churn)
		activeFiles++

		for author := range stat.Contributors {
			allContribs[author] = struct{}{}
		}
	}

	avgChurnPerFile := 0.0
	if activeFiles > 0 {
		avgChurnPerFile = totalChurn / float64(activeFiles)
	}

	// IaC file ratio based on current HEAD file list
	iacCount := 0
	for _, f := range files {
		if schema.IsIaCFile(f) {
			iacCount++
		}
	}
	iacFileRatio := 0.0
	if fileCount > 0 {
		iacFileRatio = float64(iacCount) / float64(fileCount)
	}

	preset, reasons := recommendPreset(fileCount, len(allContribs), iacFileRatio)

	return schema.RepoShape{
		FileCount:          fileCount,
		TotalCommits:       totalCommits,
		UniqueContributors: len(allContribs),
		AvgChurnPerFile:    avgChurnPerFile,
		IaCFileRatio:       iacFileRatio,
		RecommendedPreset:  preset,
		Reasoning:          reasons,
		Preset:             schema.GetPreset(preset),
		AnalyzedAt:         time.Now().UTC(),
	}
}

// GetHotspotShapeResults runs an aggregation pass and computes the repo shape.
func GetHotspotShapeResults(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) (schema.RepoShape, time.Duration, error) {
	start := time.Now()

	files, err := client.ListFilesAtRef(ctx, cfg.Git.RepoPath, "HEAD")
	if err != nil {
		return schema.RepoShape{}, 0, fmt.Errorf("failed to list files: %w", err)
	}

	// FIX: Apply PathFilter so subdirectory shape analysis only looks at relevant files
	if cfg.Git.PathFilter != "" {
		var filtered []string
		for _, f := range files {
			if schema.IsPathInFilter(f, cfg.Git.PathFilter) {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	urn := git.ResolveURN(ctx, client, cfg.Git.RepoPath)
	output, err := agg.CachedAggregateActivity(ctx, cfg.Git, cfg.Compare, client, mgr, urn)
	if err != nil {
		return schema.RepoShape{}, 0, fmt.Errorf("aggregation failed: %w", err)
	}

	shape := ComputeRepoShape(files, output)
	return shape, time.Since(start), nil
}

// ExecuteHotspotShape runs shape analysis and writes the result.
// It prints the full shape metrics as JSON to stdout.
func ExecuteHotspotShape(ctx context.Context, cfg *config.Config, client git.Client, mgr iocache.CacheManager) error {
	shape, duration, err := GetHotspotShapeResults(ctx, cfg, client, mgr)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("Shape analysis complete in %s", duration))

	data, err := json.MarshalIndent(shape, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal shape: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}
