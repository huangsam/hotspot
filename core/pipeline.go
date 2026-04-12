package core

import (
	"context"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/contract"
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
	Client contract.GitClient
	Mgr    contract.CacheManager

	// Intermediate state
	AnalysisID      int64
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
	stages []Stage
}

// NewPipeline creates a new pipeline with the given stages.
func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// Execute runs the pipeline stages sequentially.
func (p *Pipeline) Execute(ac *AnalysisContext) error {
	for _, stage := range p.stages {
		if err := stage.Execute(ac); err != nil {
			return err
		}
	}
	return nil
}
