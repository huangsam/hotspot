// Package provider implements the FormatProvider implementation for JSON output.
package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// JSONProvider implements FormatProvider for JSON output.
type JSONProvider struct{}

// NewJSONProvider creates a new JSON provider.
func NewJSONProvider() *JSONProvider {
	return &JSONProvider{}
}

// WriteFiles serializes file analysis results to JSON.
func (p *JSONProvider) WriteFiles(w io.Writer, results []schema.FileResult, _ config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	output := schema.FileResultsOutput{
		Results:  schema.EnrichFiles(results),
		Metadata: schema.BuildMetadata(runtime, duration),
	}
	return p.encode(w, output)
}

// WriteFolders serializes folder analysis results to JSON.
func (p *JSONProvider) WriteFolders(w io.Writer, results []schema.FolderResult, _ config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	output := schema.FolderResultsOutput{
		Results:  schema.EnrichFolders(results),
		Metadata: schema.BuildMetadata(runtime, duration),
	}
	return p.encode(w, output)
}

// WriteComparison serializes comparison results to JSON.
func (p *JSONProvider) WriteComparison(w io.Writer, results schema.ComparisonResult, _ config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	output := schema.ComparisonResultsOutput{
		Results:  results,
		Metadata: schema.BuildMetadata(runtime, duration),
	}
	return p.encode(w, output)
}

// WriteTimeseries serializes timeseries points to JSON.
func (p *JSONProvider) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return p.encode(w, result)
}

// WriteBlastRadius serializes blast radius results to JSON.
func (p *JSONProvider) WriteBlastRadius(w io.Writer, result schema.BlastRadiusResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return p.encode(w, result)
}

// WriteMetrics serializes metrics definitions to JSON.
func (p *JSONProvider) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	model := schema.BuildMetricsRenderModel(activeWeights)
	return p.encode(w, model)
}

// encode is a private helper to handle consistent JSON formatting.
func (p *JSONProvider) encode(w io.Writer, data any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// WriteHistory serializes analysis history to JSON.
func (p *JSONProvider) WriteHistory(w io.Writer, runs []schema.AnalysisRunRecord, _ config.OutputSettings) error {
	return p.encode(w, runs)
}

// WriteBatch serializes repository shapes to JSON.
func (p *JSONProvider) WriteBatch(w io.Writer, results []schema.RepoShape, _ config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	output := schema.BatchAnalysisResultsOutput{
		Results:  results,
		Metadata: schema.BuildMetadata(runtime, duration),
	}
	return p.encode(w, output)
}
