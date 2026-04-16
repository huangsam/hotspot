// Package provider implements the FormatProvider implementation for Parquet output.
package provider

import (
	"fmt"
	"io"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/parquet"
	"github.com/huangsam/hotspot/schema"
)

// ParquetProvider implements the FormatProvider interface for Parquet output.
type ParquetProvider struct{}

// NewParquetProvider creates a new Parquet provider.
func NewParquetProvider() *ParquetProvider {
	return &ParquetProvider{}
}

// WriteFiles writes file analysis results in Parquet format.
// Note: Parquet output requires a file path. If io.Writer is not an *os.File,
// it might fallback or error. However, the CLI usually provides a file.
func (p *ParquetProvider) WriteFiles(_ io.Writer, files []schema.FileResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	outputPath := output.GetOutputFile()
	if outputPath == "" {
		// Parquet usually requires a seekable file, it's hard to stream to stdout directly with standard libraries
		return fmt.Errorf("parquet output requires a specific output file path via --output-file")
	}

	// Map schema.FileResult to parquet.FileScoresMetrics
	records := make([]parquet.FileScoresMetrics, len(files))
	for i, f := range files {
		var owner *string
		if len(f.Owners) > 0 {
			o := f.Owners[0]
			owner = &o
		}

		// Note: We don't have all scores here, only the current mode's score.
		// We'll fill what we have.
		records[i] = parquet.FileScoresMetrics{
			FilePath:           f.Path,
			AnalysisTime:       time.Now().UTC(),
			TotalCommits:       f.Commits.Float64(),
			TotalChurn:         f.Churn.Float64(),
			LinesOfCode:        f.LinesOfCode.Float64(),
			ContributorCount:   f.UniqueContributors.Float64(),
			AgeDays:            f.AgeDays.Float64(),
			RecentLinesAdded:   f.RecentLinesAdded.Float64(),
			RecentLinesDeleted: f.RecentLinesDeleted.Float64(),
			GiniCoefficient:    f.Gini,
			FileOwner:          owner,
			ScoreLabel:         string(f.Mode),
		}

		// Set the appropriate score field based on mode
		switch f.Mode {
		case schema.HotMode:
			records[i].ScoreHot = f.ModeScore
		case schema.RiskMode:
			records[i].ScoreRisk = f.ModeScore
		case schema.ComplexityMode:
			records[i].ScoreComplexity = f.ModeScore
		case schema.StaleMode:
			records[i].ScoreStale = f.ModeScore
		}
	}

	return parquet.WriteFileScoresMetricsParquet(records, outputPath)
}

// WriteFolders is not specifically implemented for Parquet.
func (p *ParquetProvider) WriteFolders(w io.Writer, _ []schema.FolderResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	_, err := fmt.Fprintln(w, "Parquet output is not supported for folder analysis.")
	return err
}

// WriteComparison is not specifically implemented for Parquet.
func (p *ParquetProvider) WriteComparison(w io.Writer, _ schema.ComparisonResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	_, err := fmt.Fprintln(w, "Parquet output is not supported for comparison analysis.")
	return err
}

// WriteTimeseries is not specifically implemented for Parquet.
func (p *ParquetProvider) WriteTimeseries(w io.Writer, _ schema.TimeseriesResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	_, err := fmt.Fprintln(w, "Parquet output is not supported for timeseries analysis.")
	return err
}

// WriteBlastRadius is not specifically implemented for Parquet.
func (p *ParquetProvider) WriteBlastRadius(w io.Writer, _ schema.BlastRadiusResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	_, err := fmt.Fprintln(w, "Parquet output is not supported for blast radius analysis.")
	return err
}

// WriteMetrics is not specifically implemented for Parquet.
func (p *ParquetProvider) WriteMetrics(w io.Writer, _ map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	_, err := fmt.Fprintln(w, "Parquet output is not supported for metrics definitions.")
	return err
}
