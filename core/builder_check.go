package core

import (
	"context"
	"fmt"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
)

// CheckResultBuilder builds the check result using a builder pattern.
type CheckResultBuilder struct {
	gitSettings     config.GitSettings
	scoringSettings config.ScoringSettings
	compareSettings config.ComparisonSettings
	client          git.Client
	mgr             iocache.CacheManager
	ctx             context.Context
	filesToAnalyze  []string
	cfgTarget       *config.Config
	fileResults     []schema.FileResult
	maxScores       map[schema.ScoringMode]float64
	failedFiles     []schema.CheckFailedFile
	maxScoreFiles   map[schema.ScoringMode][]schema.CheckMaxScoreFile
	avgScores       map[schema.ScoringMode]float64
	result          *schema.CheckResult
}

// NewCheckResultBuilder creates a new builder for check results.
func NewCheckResultBuilder(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	compareSettings config.ComparisonSettings,
	client git.Client,
	mgr iocache.CacheManager,
) *CheckResultBuilder {
	return &CheckResultBuilder{
		gitSettings:     gitSettings,
		scoringSettings: scoringSettings,
		compareSettings: compareSettings,
		client:          client,
		mgr:             mgr,
		ctx:             ctx,
	}
}

// ValidatePrerequisites validates config and gets files to analyze.
func (b *CheckResultBuilder) ValidatePrerequisites() (*CheckResultBuilder, error) {
	// Validate that compare mode is enabled
	if !b.compareSettings.IsEnabled() {
		return nil, fmt.Errorf("check command requires --base-ref and --target-ref flags. Example: hotspot check --base-ref main --target-ref feature")
	}

	// Get the list of changed files between base and target refs
	changedFiles, err := b.client.GetChangedFilesBetweenRefs(b.ctx, b.gitSettings.GetRepoPath(), b.compareSettings.GetBaseRef(), b.compareSettings.GetTargetRef())
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files between %q and %q: %w. Verify both refs exist in the repository", b.compareSettings.GetBaseRef(), b.compareSettings.GetTargetRef(), err)
	}

	if len(changedFiles) == 0 {
		fmt.Println("No files changed between refs - check passed")
		b.result = &schema.CheckResult{Passed: true}
		return b, nil
	}

	// Filter changed files to only include those we want to analyze
	b.filesToAnalyze = filterChangedFiles(changedFiles, b.gitSettings.GetExcludes())
	if len(b.filesToAnalyze) == 0 {
		fmt.Println("No relevant files to check (all excluded) - check passed")
		b.result = &schema.CheckResult{Passed: true}
		return b, nil
	}

	return b, nil
}

// PrepareAnalysisConfig sets up the time window for analysis.
func (b *CheckResultBuilder) PrepareAnalysisConfig() (*CheckResultBuilder, error) {
	// Get the time window for the target ref
	_, targetEndTime, err := getAnalysisWindowForRef(b.ctx, b.client, b.gitSettings.GetRepoPath(), b.compareSettings.GetTargetRef(), b.compareSettings.GetLookback())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze target ref %q: %w. Verify the ref exists and has commits", b.compareSettings.GetTargetRef(), err)
	}

	// Create config for target ref analysis
	targetStartTime := targetEndTime.Add(-b.compareSettings.GetLookback())

	// Create dynamic time window settings
	b.cfgTarget = &config.Config{
		Git: config.GitConfig{
			RepoPath:   b.gitSettings.GetRepoPath(),
			StartTime:  targetStartTime,
			EndTime:    targetEndTime,
			PathFilter: b.gitSettings.GetPathFilter(),
			Excludes:   b.gitSettings.GetExcludes(),
			Follow:     b.gitSettings.IsFollow(),
		},
		Scoring: b.scoringSettings.(config.ScoringConfig), // Safe cast since we know the implementation
		Runtime: config.RuntimeConfig{Workers: 1},         // Sequential for check
		Compare: b.compareSettings.(config.CompareConfig),
	}

	return b, nil
}

// RunAnalysis performs the aggregation and file analysis.
func (b *CheckResultBuilder) RunAnalysis() (*CheckResultBuilder, error) {
	// Resolve URN for cache key consistency
	repoPath := b.cfgTarget.Git.GetRepoPath()
	var urn string
	if url, err := b.client.GetRemoteURL(b.ctx, repoPath); err == nil && url != "" {
		urn = "git:" + url
	} else {
		absPath, _ := b.client.GetRepoRoot(b.ctx, repoPath)
		if absPath == "" {
			absPath = repoPath
		}
		urn = "local:" + absPath
	}

	// Run aggregation once (shared for all modes)
	output, err := agg.CachedAggregateActivity(b.ctx, b.cfgTarget.Git, b.cfgTarget.Compare, b.client, b.mgr, urn)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze repository activity: %w. Verify the repository has Git history and is readable", err)
	}

	// Analyze files once (all scores are computed upfront in FileResult)
	b.fileResults = analyzeRepo(b.ctx, b.cfgTarget.Git, b.cfgTarget.Scoring, b.cfgTarget.Runtime, b.client, output, b.filesToAnalyze)

	return b, nil
}

// ComputeMetrics calculates max scores and identifies failed files.
func (b *CheckResultBuilder) ComputeMetrics() *CheckResultBuilder {
	// Compute max scores for each mode
	b.maxScores = make(map[schema.ScoringMode]float64)
	b.maxScoreFiles = make(map[schema.ScoringMode][]schema.CheckMaxScoreFile)
	b.avgScores = make(map[schema.ScoringMode]float64)

	for _, mode := range schema.AllScoringModes {
		maxScore := 0.0
		var filesWithMax []schema.CheckMaxScoreFile
		sumScore := 0.0
		fileCount := len(b.fileResults)

		for _, file := range b.fileResults {
			score := file.AllScores[mode]
			sumScore += score
			if score > maxScore {
				maxScore = score
				filesWithMax = []schema.CheckMaxScoreFile{{
					Path:   file.Path,
					Owners: file.Owners,
				}}
			} else if score == maxScore && maxScore > 0 {
				// Include files that tie for the max score
				filesWithMax = append(filesWithMax, schema.CheckMaxScoreFile{
					Path:   file.Path,
					Owners: file.Owners,
				})
			}
		}

		b.maxScores[mode] = maxScore
		b.maxScoreFiles[mode] = filesWithMax
		if fileCount > 0 {
			b.avgScores[mode] = sumScore / float64(fileCount)
		}
	}

	// Check all files against thresholds for all modes
	b.failedFiles = []schema.CheckFailedFile{}
	thresholds := b.scoringSettings.GetRiskThresholds()
	for _, file := range b.fileResults {
		for _, mode := range schema.AllScoringModes {
			score := file.AllScores[mode]
			threshold := thresholds[mode]
			if score > threshold {
				b.failedFiles = append(b.failedFiles, schema.CheckFailedFile{
					Path:      file.Path,
					Mode:      mode,
					Score:     score,
					Threshold: threshold,
				})
			}
		}
	}

	return b
}

// BuildResult constructs the final CheckResult.
func (b *CheckResultBuilder) BuildResult() *CheckResultBuilder {
	b.result = &schema.CheckResult{
		Passed:        len(b.failedFiles) == 0,
		FailedFiles:   b.failedFiles,
		TotalFiles:    len(b.filesToAnalyze),
		CheckedModes:  schema.AllScoringModes,
		BaseRef:       b.compareSettings.GetBaseRef(),
		TargetRef:     b.compareSettings.GetTargetRef(),
		Thresholds:    b.scoringSettings.GetRiskThresholds(),
		MaxScores:     b.maxScores,
		MaxScoreFiles: b.maxScoreFiles,
		Lookback:      b.compareSettings.GetLookback(),
		AvgScores:     b.avgScores,
	}
	return b
}

// GetResult returns the built CheckResult.
func (b *CheckResultBuilder) GetResult() *schema.CheckResult {
	return b.result
}

// filterChangedFiles filters the list of changed files based on excludes.
func filterChangedFiles(files []string, excludes []string) []string {
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if !schema.ShouldIgnore(f, excludes) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
