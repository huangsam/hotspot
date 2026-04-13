// Package outwriter has output and writer logic.
package outwriter

import (
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/outwriter/csv"
	"github.com/huangsam/hotspot/internal/outwriter/describe"
	"github.com/huangsam/hotspot/internal/outwriter/json"
	"github.com/huangsam/hotspot/internal/outwriter/markdown"
	"github.com/huangsam/hotspot/internal/outwriter/parquet"
	"github.com/huangsam/hotspot/internal/outwriter/text"
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
	ow.providers[schema.JSONOut] = json.NewProvider()
	ow.providers[schema.CSVOut] = csv.NewProvider()
	ow.providers[schema.TextOut] = text.NewProvider()
	ow.providers[schema.MarkdownOut] = markdown.NewProvider()
	ow.providers[schema.Describe] = describe.NewProvider()
	ow.providers[schema.ParquetOut] = parquet.NewProvider()

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
