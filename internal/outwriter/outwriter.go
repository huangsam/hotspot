// Package outwriter has output and writer logic.
package outwriter

import (
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
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
func (ow *OutWriter) WriteFiles(w io.Writer, results []schema.FileResult, cfg *contract.Config, duration time.Duration) error {
	return WriteFileResults(w, results, cfg, duration)
}

// WriteFolders writes folder analysis results using the configured output format.
func (ow *OutWriter) WriteFolders(w io.Writer, results []schema.FolderResult, cfg *contract.Config, duration time.Duration) error {
	return WriteFolderResults(w, results, cfg, duration)
}

// WriteComparison writes comparison analysis results using the configured output format.
func (ow *OutWriter) WriteComparison(w io.Writer, results schema.ComparisonResult, cfg *contract.Config, duration time.Duration) error {
	return WriteComparisonResults(w, results, cfg, duration)
}

// WriteTimeseries writes timeseries analysis results using the configured output format.
func (ow *OutWriter) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, cfg *contract.Config, duration time.Duration) error {
	return WriteTimeseriesResults(w, result, cfg, duration)
}

// WriteMetrics writes metrics definitions using the configured output format.
func (ow *OutWriter) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, cfg *contract.Config) error {
	return WriteMetricsDefinitions(w, activeWeights, cfg)
}
