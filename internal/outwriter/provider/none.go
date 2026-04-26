package provider

import (
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// NoneProvider is a no-op implementation of FormatProvider.
type NoneProvider struct{}

// NewNoneProvider creates a new NoneProvider.
func NewNoneProvider() *NoneProvider {
	return &NoneProvider{}
}

// WriteFiles is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteFiles(_ io.Writer, _ []schema.FileResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return nil
}

// WriteFolders is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteFolders(_ io.Writer, _ []schema.FolderResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return nil
}

// WriteComparison is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteComparison(_ io.Writer, _ schema.ComparisonResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return nil
}

// WriteMetrics is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteMetrics(_ io.Writer, _ map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	return nil
}

// WriteTimeseries is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteTimeseries(_ io.Writer, _ schema.TimeseriesResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return nil
}

// WriteBlastRadius is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteBlastRadius(_ io.Writer, _ schema.BlastRadiusResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	return nil
}

// WriteHistory is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteHistory(_ io.Writer, _ []schema.AnalysisRunRecord, _ config.OutputSettings) error {
	return nil
}

// WriteBatch is a no-op implementation of FormatProvider.
func (p *NoneProvider) WriteBatch(_ io.Writer, _ []schema.RepoShape, _ config.OutputSettings) error {
	return nil
}
