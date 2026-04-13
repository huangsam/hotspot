// Package outwriter has output and writer logic.
package outwriter

import (
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/outwriter/json"
	"github.com/huangsam/hotspot/schema"
)

// FormatProvider defines the behavior for specific output formats (e.g. JSON, CSV, Table).
type FormatProvider interface {
	WriteFiles(w io.Writer, results []schema.FileResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error
	WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error
	WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error
	WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error
	WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, output config.OutputSettings) error
}

// OutWriter provides a unified interface for all output operations.
type OutWriter struct {
	providers map[schema.OutputMode]FormatProvider
}

// NewOutWriter creates a new instance of the output writer and registers handlers.
func NewOutWriter() *OutWriter {
	ow := &OutWriter{
		providers: make(map[schema.OutputMode]FormatProvider),
	}

	// Register specific providers
	jsonProvider := json.NewProvider()
	ow.providers[schema.JSONOut] = jsonProvider

	// For other modes, register a legacy dispatcher as a bridge.
	legacy := &legacyDispatcher{}
	for mode := range schema.ValidOutputModes {
		if _, exists := ow.providers[mode]; !exists {
			ow.providers[mode] = legacy
		}
	}

	return ow
}

// WriteFiles writes file analysis results using the configured output format.
func (ow *OutWriter) WriteFiles(w io.Writer, results []schema.FileResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return ow.providers[output.GetFormat()].WriteFiles(w, results, output, runtime, duration)
}

// WriteFolders writes folder analysis results using the configured output format.
func (ow *OutWriter) WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return ow.providers[output.GetFormat()].WriteFolders(w, results, output, runtime, duration)
}

// WriteComparison writes comparison analysis results using the configured output format.
func (ow *OutWriter) WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return ow.providers[output.GetFormat()].WriteComparison(w, results, output, runtime, duration)
}

// WriteTimeseries writes timeseries analysis results using the configured output format.
func (ow *OutWriter) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return ow.providers[output.GetFormat()].WriteTimeseries(w, result, output, runtime, duration)
}

// WriteMetrics writes metrics definitions using the configured output format.
func (ow *OutWriter) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, output config.OutputSettings) error {
	return ow.providers[output.GetFormat()].WriteMetrics(w, activeWeights, output)
}

// legacyDispatcher acts as a bridge to the existing package-level functions.
type legacyDispatcher struct{}

func (d *legacyDispatcher) WriteFiles(w io.Writer, results []schema.FileResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteFileResults(w, results, output, runtime, duration)
}

func (d *legacyDispatcher) WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteFolderResults(w, results, output, runtime, duration)
}

func (d *legacyDispatcher) WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteComparisonResults(w, results, output, runtime, duration)
}

func (d *legacyDispatcher) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteTimeseriesResults(w, result, output, runtime, duration)
}

func (d *legacyDispatcher) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, output config.OutputSettings) error {
	return WriteMetricsDefinitions(w, activeWeights, output)
}
