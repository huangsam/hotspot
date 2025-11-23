package core

import (
	"context"
	"fmt"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// CheckResultBuilder builds the check result using a builder pattern.
type CheckResultBuilder struct {
	cfg            *contract.Config
	client         contract.GitClient
	mgr            contract.CacheManager
	ctx            context.Context
	filesToAnalyze []string
	cfgTarget      *contract.Config
	fileResults    []schema.FileResult
	maxScores      map[schema.ScoringMode]float64
	failedFiles    []schema.CheckFailedFile
	maxScoreFiles  map[schema.ScoringMode][]schema.CheckMaxScoreFile
	result         *schema.CheckResult
}

// NewCheckResultBuilder creates a new builder for check results.
func NewCheckResultBuilder(ctx context.Context, cfg *contract.Config, mgr contract.CacheManager) *CheckResultBuilder {
	return &CheckResultBuilder{
		cfg:    cfg,
		client: contract.NewLocalGitClient(),
		mgr:    mgr,
		ctx:    ctx,
	}
}

// ValidatePrerequisites validates config and gets files to analyze.
func (b *CheckResultBuilder) ValidatePrerequisites() (*CheckResultBuilder, error) {
	// Validate that compare mode is enabled
	if !b.cfg.CompareMode {
		return nil, fmt.Errorf("check requires --base-ref and --target-ref flags")
	}

	// Get the list of changed files between base and target refs
	changedFiles, err := b.client.GetChangedFilesBetweenRefs(b.ctx, b.cfg.RepoPath, b.cfg.BaseRef, b.cfg.TargetRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if len(changedFiles) == 0 {
		fmt.Println("No files changed between refs - check passed")
		b.result = &schema.CheckResult{Passed: true}
		return b, nil
	}

	// Filter changed files to only include those we want to analyze
	b.filesToAnalyze = filterChangedFiles(changedFiles, b.cfg.Excludes)
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
	_, targetEndTime, err := getAnalysisWindowForRef(b.ctx, b.client, b.cfg.RepoPath, b.cfg.TargetRef, b.cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve time window for target ref '%s': %w", b.cfg.TargetRef, err)
	}

	// Create config for target ref analysis
	targetStartTime := targetEndTime.Add(-b.cfg.Lookback)
	b.cfgTarget = b.cfg.CloneWithTimeWindow(targetStartTime, targetEndTime)

	return b, nil
}

// RunAnalysis performs the aggregation and file analysis.
func (b *CheckResultBuilder) RunAnalysis() (*CheckResultBuilder, error) {
	// Run aggregation once (shared for all modes)
	output, err := agg.CachedAggregateActivity(b.ctx, b.cfgTarget, b.client, b.mgr)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate activity: %w", err)
	}

	// Analyze files once (all scores are computed upfront in FileResult)
	cfgDefault := b.cfgTarget.Clone()
	cfgDefault.Mode = schema.HotMode // Mode doesn't matter since all scores are computed
	b.fileResults = analyzeRepo(b.ctx, cfgDefault, b.client, output, b.filesToAnalyze)

	return b, nil
}

// ComputeMetrics calculates max scores and identifies failed files.
func (b *CheckResultBuilder) ComputeMetrics() *CheckResultBuilder {
	// Compute max scores for each mode
	b.maxScores = make(map[schema.ScoringMode]float64)
	b.maxScoreFiles = make(map[schema.ScoringMode][]schema.CheckMaxScoreFile)

	for _, mode := range schema.AllScoringModes {
		maxScore := 0.0
		var filesWithMax []schema.CheckMaxScoreFile

		for _, file := range b.fileResults {
			score := file.AllScores[mode]
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
	}

	// Check all files against thresholds for all modes
	b.failedFiles = []schema.CheckFailedFile{}
	for _, file := range b.fileResults {
		for _, mode := range schema.AllScoringModes {
			score := file.AllScores[mode]
			threshold := b.cfg.RiskThresholds[mode]
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
		BaseRef:       b.cfg.BaseRef,
		TargetRef:     b.cfg.TargetRef,
		Thresholds:    b.cfg.RiskThresholds,
		MaxScores:     b.maxScores,
		MaxScoreFiles: b.maxScoreFiles,
		Lookback:      b.cfg.Lookback,
	}
	return b
}

// GetResult returns the built CheckResult.
func (b *CheckResultBuilder) GetResult() *schema.CheckResult {
	return b.result
}
