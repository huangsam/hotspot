// Package outwriter has output and writer logic.
package outwriter

import (
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// OutWriter provides a unified interface for all output operations.
// It encapsulates the various output formats and provides a clean API for the core logic.
type OutWriter struct{}

// NewOutWriter creates a new instance of the output writer.
func NewOutWriter() *OutWriter {
	return &OutWriter{}
}

// WriteFiles writes file analysis results using the configured output format.
func (ow *OutWriter) WriteFiles(w io.Writer, results []schema.FileResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteFileResults(w, results, output, runtime, duration)
}

// WriteFolders writes folder analysis results using the configured output format.
func (ow *OutWriter) WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteFolderResults(w, results, output, runtime, duration)
}

// WriteComparison writes comparison analysis results using the configured output format.
func (ow *OutWriter) WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteComparisonResults(w, results, output, runtime, duration)
}

// WriteTimeseries writes timeseries analysis results using the configured output format.
func (ow *OutWriter) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	return WriteTimeseriesResults(w, result, output, runtime, duration)
}

// WriteMetrics writes metrics definitions using the configured output format.
func (ow *OutWriter) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, output config.OutputSettings) error {
	return WriteMetricsDefinitions(w, activeWeights, output)
}
