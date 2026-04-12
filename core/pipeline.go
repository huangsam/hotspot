package core

import (
	"context"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
)

// AnalysisContext holds the entire state of an analysis pass as it moves through the pipeline.
type AnalysisContext struct {
	// Request state
	Context   context.Context
	Git       config.GitSettings
	Scoring   config.ScoringSettings
	Runtime   config.RuntimeSettings
	Output    config.OutputSettings
	Compare   config.ComparisonSettings
	TargetRef string // Defaults to "HEAD". Can be set for compare mode or timeseries.

	// Injected dependencies
	Client git.Client
	Mgr    iocache.CacheManager

	// Intermediate state
	AnalysisID      int64
	RepoURN         string
	AnalysisStore   iocache.AnalysisStore
	Files           []string
	AggregateOutput *schema.AggregateOutput

	// Output state
	FileResults   []schema.FileResult
	FolderResults []schema.FolderResult
}

// Stage represents a single step in the analysis process.
type Stage interface {
	Execute(ac *AnalysisContext) error
}

// Pipeline orchestrates a series of Stages.
type Pipeline struct {
	stages     []Stage
	deferStage Stage // Always runs after stages, even on error.
}

// NewPipeline creates a new pipeline with the given stages.
func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// WithDefer registers a stage that always executes after the pipeline,
// even if an earlier stage returns an error.
func (p *Pipeline) WithDefer(stage Stage) *Pipeline {
	p.deferStage = stage
	return p
}

// Execute runs the pipeline stages sequentially.
// The deferred stage (if set) always runs, regardless of errors.
func (p *Pipeline) Execute(ac *AnalysisContext) error {
	var pipelineErr error
	for _, stage := range p.stages {
		if err := stage.Execute(ac); err != nil {
			pipelineErr = err
			break
		}
	}
	if p.deferStage != nil {
		if deferErr := p.deferStage.Execute(ac); deferErr != nil {
			contract.LogWarn("Deferred pipeline stage failed", deferErr)
		}
	}
	return pipelineErr
}
