// Package json provides a FormatProvider implementation for JSON output.
package json

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// Provider implements util.FormatProvider for JSON output.
type Provider struct{}

// NewProvider creates a new JSON provider.
func NewProvider() *Provider {
	return &Provider{}
}

// WriteFiles serializes file analysis results to JSON.
func (p *Provider) WriteFiles(w io.Writer, results []schema.FileResult, _ config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	output := schema.FileResultsOutput{
		Results:  schema.EnrichFiles(results),
		Metadata: schema.BuildMetadata(runtime, duration),
	}
	return p.encode(w, output)
}

// WriteFolders serializes folder analysis results to JSON.
func (p *Provider) WriteFolders(w io.Writer, results []schema.FolderResult, _ config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	output := schema.FolderResultsOutput{
		Results:  schema.EnrichFolders(results),
		Metadata: schema.BuildMetadata(runtime, duration),
	}
	return p.encode(w, output)
}

// WriteComparison serializes comparison results to JSON.
func (p *Provider) WriteComparison(w io.Writer, results schema.ComparisonResult, _ config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	output := schema.ComparisonResultsOutput{
		Results:  results,
		Metadata: schema.BuildMetadata(runtime, duration),
	}
	return p.encode(w, output)
}

// WriteTimeseries serializes timeseries points to JSON.
func (p *Provider) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return p.encode(w, result)
}

// WriteMetrics serializes metrics definitions to JSON.
func (p *Provider) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	model := schema.BuildMetricsRenderModel(activeWeights)
	return p.encode(w, model)
}

// encode is a private helper to handle consistent JSON formatting.
func (p *Provider) encode(w io.Writer, data any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}
