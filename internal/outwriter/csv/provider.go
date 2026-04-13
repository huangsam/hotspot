// Package csv provides a FormatProvider implementation for CSV output.
package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/outwriter/util"
	"github.com/huangsam/hotspot/schema"
)

// Provider implements the util.FormatProvider interface for CSV output.
type Provider struct{}

// NewProvider creates a new CSV provider.
func NewProvider() *Provider {
	return &Provider{}
}

// WriteFiles writes file analysis results in CSV format.
func (p *Provider) WriteFiles(w io.Writer, files []schema.FileResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat, intFmt := util.CreateFormatters(output.GetPrecision())
	header := []string{
		"rank",
		"file",
		"score",
		"label",
		"contributors",
		"commits",
		"size_kb",
		"age_days",
		"churn",
		"gini",
		"first_commit",
		"owner",
		"mode",
	}

	return util.WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for i, f := range files {
			rec := []string{
				strconv.Itoa(i + 1),                         // Rank
				f.Path,                                      // File Path
				fmtFloat(f.ModeScore),                       // Score
				schema.GetPlainLabel(f.ModeScore),           // Label
				fmt.Sprintf(intFmt, f.UniqueContributors),   // Contributors
				fmt.Sprintf(intFmt, f.Commits),              // Commits
				fmtFloat(float64(f.SizeBytes) / 1024.0),     // Size in KB
				fmt.Sprintf(intFmt, f.AgeDays),              // Age in Days
				fmt.Sprintf(intFmt, f.Churn),                // Churn
				fmtFloat(f.Gini),                            // Gini Coefficient
				f.FirstCommit.Format(schema.DateTimeFormat), // First Commit Date
				strings.Join(f.Owners, "|"),                 // Owners
				string(f.Mode),                              // Mode
			}
			if err := csvWriter.Write(rec); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteFolders writes folder analysis results in CSV format.
func (p *Provider) WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat, intFmt := util.CreateFormatters(output.GetPrecision())
	header := []string{
		"rank",
		"folder",
		"score",
		"label",
		"total_commits",
		"total_churn",
		"total_loc",
		"owner",
		"mode",
	}

	return util.WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for i, r := range results {
			row := []string{
				strconv.Itoa(i + 1),             // Rank
				r.Path,                          // Folder Path
				fmtFloat(r.Score),               // Score
				schema.GetPlainLabel(r.Score),   // Label
				fmt.Sprintf(intFmt, r.Commits),  // Total Commits
				fmt.Sprintf(intFmt, r.Churn),    // Total Churn
				fmt.Sprintf(intFmt, r.TotalLOC), // Total LOC
				strings.Join(r.Owners, "|"),     // Owners
				string(r.Mode),                  // Mode
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteComparison writes comparison analysis results in CSV format.
func (p *Provider) WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat, intFmt := util.CreateFormatters(output.GetPrecision())
	header := []string{
		"rank",
		"path",
		"base_score",
		"comp_score",
		"delta_score",
		"delta_commits",
		"delta_churn",
		"before_owners",
		"after_owners",
		"mode",
	}

	return util.WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for i, r := range results.Details {
			row := []string{
				strconv.Itoa(i + 1),                 // Rank
				r.Path,                              // Path
				fmtFloat(r.BeforeScore),             // Base Score
				fmtFloat(r.AfterScore),              // Current Score
				fmtFloat(r.Delta),                   // Delta Score (Current - Base)
				fmt.Sprintf(intFmt, r.DeltaCommits), // Delta Commits
				fmt.Sprintf(intFmt, r.DeltaChurn),   // Delta Churn
				strings.Join(r.BeforeOwners, "|"),   // Base Owners
				strings.Join(r.AfterOwners, "|"),    // Current Owners
				string(r.Mode),                      // Mode
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteTimeseries writes timeseries analysis results in CSV format.
func (p *Provider) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat, _ := util.CreateFormatters(output.GetPrecision())
	header := []string{
		"path",
		"period",
		"score",
		"label",
		"owners",
		"mode",
		"start",
		"end",
	}

	return util.WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for _, point := range result.Points {
			row := []string{
				point.Path,                                // Path
				point.Period,                              // Period
				fmtFloat(point.Score),                     // Score
				schema.GetPlainLabel(point.Score),         // Label
				strings.Join(point.Owners, "|"),           // Owners
				string(point.Mode),                        // Mode
				point.Start.Format(schema.DateTimeFormat), // Start
				point.End.Format(schema.DateTimeFormat),   // End
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteMetrics writes metrics definitions in CSV format.
func (p *Provider) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	renderModel := schema.BuildMetricsRenderModel(activeWeights)
	header := []string{
		"Mode",
		"Purpose",
		"Factors",
		"Formula",
	}

	return util.WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for _, m := range renderModel.Modes {
			row := []string{
				m.Name,
				m.Purpose,
				strings.Join(m.Factors, "|"),
				m.Formula,
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}
