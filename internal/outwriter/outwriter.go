// Package outwriter has output and writer logic.
package outwriter

import (
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

// WriteFiles prints file analysis results using the configured output format.
func (ow *OutWriter) WriteFiles(results []schema.FileResult, cfg *contract.Config, duration time.Duration) error {
	return WriteFileResults(results, cfg, duration)
}

// WriteFolders prints folder analysis results using the configured output format.
func (ow *OutWriter) WriteFolders(results []schema.FolderResult, cfg *contract.Config, duration time.Duration) error {
	return WriteFolderResults(results, cfg, duration)
}

// WriteComparison prints comparison analysis results using the configured output format.
func (ow *OutWriter) WriteComparison(results schema.ComparisonResult, cfg *contract.Config, duration time.Duration) error {
	return WriteComparisonResults(results, cfg, duration)
}

// WriteTimeseries prints timeseries analysis results using the configured output format.
func (ow *OutWriter) WriteTimeseries(result schema.TimeseriesResult, cfg *contract.Config, duration time.Duration) error {
	return WriteTimeseriesResults(result, cfg, duration)
}

// WriteMetrics prints metrics definitions using the configured output format.
func (ow *OutWriter) WriteMetrics(activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, cfg *contract.Config) error {
	return WriteMetricsDefinitions(activeWeights, cfg)
}
