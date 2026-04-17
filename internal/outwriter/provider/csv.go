package provider

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// CSVProvider implements the FormatProvider interface for CSV output.
type CSVProvider struct{}

// NewCSVProvider creates a new CSV provider.
func NewCSVProvider() *CSVProvider {
	return &CSVProvider{}
}

// WriteFiles writes file analysis results in CSV format.
func (p *CSVProvider) WriteFiles(w io.Writer, files []schema.FileResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
	header := []string{
		"rank",
		"file",
		"score",
		"label",
		"contributors",
		"commits",
		"loc",
		"size_kb",
		"age_days",
		"churn",
		"gini",
		"first_commit",
		"owner",
		"mode",
		"explain",
	}

	return WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for i, f := range files {
			rec := []string{
				strconv.Itoa(i + 1),                         // Rank
				f.Path,                                      // File Path
				fmtFloat(f.ModeScore),                       // Score
				schema.GetPlainLabel(f.ModeScore),           // Label
				f.UniqueContributors.Display(),              // Contributors
				f.Commits.Display(),                         // Commits
				f.LinesOfCode.Display(),                     // Lines of Code
				fmtFloat(float64(f.SizeBytes) / 1024.0),     // Size in KB
				f.AgeDays.Display(),                         // Age in Days
				f.Churn.Display(),                           // Churn
				fmtFloat(f.Gini),                            // Gini Coefficient
				f.FirstCommit.Format(schema.DateTimeFormat), // First Commit Date
				strings.Join(f.Owners, "|"),                 // Owners
				string(f.Mode),                              // Mode
				FormatTopMetricBreakdown(&f),                // Explain
			}
			if err := csvWriter.Write(rec); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteFolders writes folder analysis results in CSV format.
func (p *CSVProvider) WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
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

	return WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for i, r := range results {
			row := []string{
				strconv.Itoa(i + 1),           // Rank
				r.Path,                        // Folder Path
				fmtFloat(r.Score),             // Score
				schema.GetPlainLabel(r.Score), // Label
				r.Commits.Display(),           // Total Commits
				r.Churn.Display(),             // Total Churn
				r.TotalLOC.Display(),          // Total LOC
				strings.Join(r.Owners, "|"),   // Owners
				string(r.Mode),                // Mode
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteComparison writes comparison analysis results in CSV format.
func (p *CSVProvider) WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
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

	return WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for i, r := range results.Details {
			row := []string{
				strconv.Itoa(i + 1),               // Rank
				r.Path,                            // Path
				fmtFloat(r.BeforeScore),           // Base Score
				fmtFloat(r.AfterScore),            // Current Score
				fmtFloat(r.Delta),                 // Delta Score (Current - Base)
				r.DeltaCommits.Display(),          // Delta Commits
				r.DeltaChurn.Display(),            // Delta Churn
				strings.Join(r.BeforeOwners, "|"), // Base Owners
				strings.Join(r.AfterOwners, "|"),  // Current Owners
				string(r.Mode),                    // Mode
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteTimeseries writes timeseries analysis results in CSV format.
func (p *CSVProvider) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
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

	return WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
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

// WriteBlastRadius writes blast radius analysis results in CSV format.
func (p *CSVProvider) WriteBlastRadius(w io.Writer, result schema.BlastRadiusResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
	header := []string{
		"rank",
		"file_a",
		"file_b",
		"score",
		"co_change",
	}

	return WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for i, pair := range result.Pairs {
			row := []string{
				strconv.Itoa(i + 1),
				pair.Source,
				pair.Target,
				fmtFloat(pair.Score),
				strconv.Itoa(pair.CoChange),
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

// WriteMetrics writes metrics definitions in CSV format.
func (p *CSVProvider) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	renderModel := schema.BuildMetricsRenderModel(activeWeights)
	header := []string{
		"Mode",
		"Purpose",
		"Factors",
		"Formula",
	}

	return WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
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

// WriteHistory writes analysis history in CSV format.
func (p *CSVProvider) WriteHistory(w io.Writer, runs []schema.AnalysisRunRecord, _ config.OutputSettings) error {
	header := []string{"analysis_id", "start_time", "end_time", "duration_ms", "total_files", "urn"}

	return WriteCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
		for _, r := range runs {
			endTime := ""
			if r.EndTime != nil {
				endTime = r.EndTime.Format(schema.DateTimeFormat)
			}
			duration := ""
			if r.RunDurationMs != nil {
				duration = strconv.Itoa(int(*r.RunDurationMs))
			}
			files := ""
			if r.TotalFilesAnalyzed != nil {
				files = strconv.Itoa(int(*r.TotalFilesAnalyzed))
			}

			row := []string{
				strconv.FormatInt(r.AnalysisID, 10),
				r.StartTime.Format(schema.DateTimeFormat),
				endTime,
				duration,
				files,
				r.URN,
			}
			if err := csvWriter.Write(row); err != nil {
				return err
			}
		}
		return nil
	})
}
