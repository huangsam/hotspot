// Package provider implements the FormatProvider implementation for human-readable text output.
package provider

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// TextProvider implements the FormatProvider interface for human-readable text output.
type TextProvider struct{}

// NewTextProvider creates a new text provider.
func NewTextProvider() *TextProvider {
	return &TextProvider{}
}

// WriteFiles writes file analysis results in a human-readable table.
func (p *TextProvider) WriteFiles(w io.Writer, files []schema.FileResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
	table := tablewriter.NewWriter(w)
	defer func() { _ = table.Close() }()

	headers := []string{"Rank", "Path", "Score", "Label"}
	if output.IsDetail() {
		headers = append(headers, "Contrib", "Commits", "LOC", "Churn", "Age", "Gini")
	}
	if output.IsExplain() {
		headers = append(headers, "Explain")
	}
	if output.IsOwner() {
		headers = append(headers, "Owner")
	}
	table.Header(headers)

	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	var data [][]string
	for i, f := range files {
		label := schema.GetPlainLabel(f.ModeScore)
		if output.IsUseColors() {
			label = GetColorLabel(f.ModeScore)
		}
		row := []string{
			strconv.Itoa(i + 1), // Rank
			TruncatePath(f.Path, GetMaxTablePathWidth(output)), // File
			fmtFloat(f.ModeScore),                              // Score
			label,                                              // Label
		}
		if output.IsDetail() {
			row = append(
				row,
				f.UniqueContributors.Display(), // Contrib
				f.Commits.Display(),            // Commits
				f.LinesOfCode.Display(),        // LOC
				f.Churn.Display(),              // Churn
				f.AgeDays.Display(),            // Age
				fmtFloat(f.Gini),               // Gini
			)
		}
		if output.IsExplain() {
			topOnes := FormatTopMetricBreakdown(&f)
			row = append(row, topOnes)
		}
		if output.IsOwner() {
			row = append(row, schema.FormatOwners(f.Owners))
		}
		data = append(data, row)
	}

	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}

	numFiles := len(files)
	var totalCommits schema.Metric
	var totalChurn schema.Metric
	for _, f := range files {
		totalCommits += f.Commits
		totalChurn += f.Churn
	}
	if _, err := fmt.Fprintf(w, "Showing top %d files (total commits: %s, total churn: %s)\n", numFiles, totalCommits.Display(), totalChurn.Display()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Analysis completed in %v with %d workers. Cache backend: %s\n", duration, runtime.GetWorkers(), runtime.GetCacheBackend()); err != nil {
		return err
	}
	return nil
}

// WriteFolders writes folder analysis results in a human-readable table.
func (p *TextProvider) WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
	table := tablewriter.NewWriter(w)
	defer func() { _ = table.Close() }()

	headers := []string{"Rank", "Path", "Score", "Label"}
	if output.IsDetail() {
		headers = append(headers, "Commits", "Churn", "LOC")
	}
	if output.IsOwner() {
		headers = append(headers, "Owner")
	}
	table.Header(headers)

	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	var data [][]string
	for i, r := range results {
		label := schema.GetPlainLabel(r.Score)
		if output.IsUseColors() {
			label = GetColorLabel(r.Score)
		}
		row := []string{
			strconv.Itoa(i + 1), // Rank
			TruncatePath(r.Path, GetMaxTablePathWidth(output)), // Folder Path
			fmtFloat(r.Score), // Score
			label,             // Label
		}
		if output.IsDetail() {
			row = append(row,
				r.Commits.Display(),  // Total Commits
				r.Churn.Display(),    // Total Churn
				r.TotalLOC.Display(), // Total LOC
			)
		}
		if output.IsOwner() {
			row = append(row, schema.FormatOwners(r.Owners))
		}
		data = append(data, row)
	}

	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}

	numFolders := len(results)
	var totalCommits schema.Metric
	var totalChurn schema.Metric
	var totalLOC schema.Metric
	for _, r := range results {
		totalCommits += r.Commits
		totalChurn += r.Churn
		totalLOC += r.TotalLOC
	}
	if _, err := fmt.Fprintf(w, "Showing top %d folders (total commits: %s, total churn: %s, total LOC: %s)\n", numFolders, totalCommits.Display(), totalChurn.Display(), totalLOC.Display()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Analysis completed in %v with %d workers. Cache backend: %s\n", duration, runtime.GetWorkers(), runtime.GetCacheBackend()); err != nil {
		return err
	}
	return nil
}

// WriteComparison writes comparison analysis results in a human-readable table.
func (p *TextProvider) WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
	table := tablewriter.NewWriter(w)
	defer func() { _ = table.Close() }()

	headers := []string{"Rank", "Path", "Before", "After", "Delta", "Status"}
	if output.IsDetail() {
		headers = append(headers, "Δ Churn")
	}
	if output.IsOwner() {
		headers = append(headers, "Ownership")
	}
	table.Header(headers)

	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	var data [][]string
	for i, r := range results.Details {
		deltaStr := FormatComparisonDelta(r.Delta, output.GetPrecision(), output.IsUseColors())

		row := []string{
			strconv.Itoa(i + 1), // Rank
			TruncatePath(r.Path, GetMaxTablePathWidth(output)), // File Path
			fmtFloat(r.BeforeScore),                            // Base Score
			fmtFloat(r.AfterScore),                             // Comparison Score
			deltaStr,                                           // Delta Score
			string(r.Status),                                   // Status
		}
		if output.IsDetail() {
			row = append(row, r.DeltaChurn.Display())
		}
		if output.IsOwner() {
			row = append(row, FormatOwnershipDiff(r))
		}
		data = append(data, row)
	}

	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}

	numItems := len(results.Details)
	if _, err := fmt.Fprintf(w, "Showing top %d changes\n", numItems); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Net score delta: %.*f, Net churn delta: %s\n", output.GetPrecision(), results.Summary.NetScoreDelta, results.Summary.NetChurnDelta.Display()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "New files: %d, Inactive files: %d, Modified files: %d, Ownership changes: %d\n", results.Summary.TotalNewFiles, results.Summary.TotalInactiveFiles, results.Summary.TotalModifiedFiles, results.Summary.TotalOwnershipChanges); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Analysis completed in %v with %d workers. Cache backend: %s\n", duration, runtime.GetWorkers(), runtime.GetCacheBackend()); err != nil {
		return err
	}
	return nil
}

// WriteTimeseries writes timeseries analysis results in a human-readable table.
func (p *TextProvider) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
	table := tablewriter.NewWriter(w)
	defer func() { _ = table.Close() }()

	headers := []string{"Rank", "Path", "Period", "Score", "Mode", "Owner"}
	table.Header(headers)

	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	var data [][]string
	for i, pt := range result.Points {
		ownersStr := "No owners"
		if len(pt.Owners) > 0 {
			ownersStr = schema.FormatOwners(pt.Owners)
		}
		row := []string{
			strconv.Itoa(i + 1),
			TruncatePath(pt.Path, GetMaxTablePathWidth(output)),
			pt.Period,
			fmtFloat(pt.Score),
			string(pt.Mode),
			ownersStr,
		}
		data = append(data, row)
	}

	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Timeseries analysis completed in %v with %d workers. Cache backend: %s\n", duration, runtime.GetWorkers(), runtime.GetCacheBackend()); err != nil {
		return err
	}
	return nil
}

// WriteBlastRadius writes blast radius analysis results in a human-readable table.
func (p *TextProvider) WriteBlastRadius(w io.Writer, result schema.BlastRadiusResult, output config.OutputSettings, runtime config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())
	table := tablewriter.NewWriter(w)
	defer func() { _ = table.Close() }()

	headers := []string{"Rank", "File A", "File B", "Score", "Co-Change"}
	table.Header(headers)

	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignRight
	})

	var data [][]string
	for i, pair := range result.Pairs {
		row := []string{
			strconv.Itoa(i + 1),
			TruncatePath(pair.Source, GetMaxTablePathWidth(output)),
			TruncatePath(pair.Target, GetMaxTablePathWidth(output)),
			fmtFloat(pair.Score),
			strconv.Itoa(pair.CoChange),
		}
		data = append(data, row)
	}

	if err := table.Bulk(data); err != nil {
		return err
	}
	if err := table.Render(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Found %d coupled pairs above threshold %v (total commits analyzed: %d)\n", result.Summary.TotalPairs, result.Summary.Threshold, result.Summary.TotalCommits); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Blast radius analysis completed in %v with %d workers. Cache backend: %s\n", duration, runtime.GetWorkers(), runtime.GetCacheBackend()); err != nil {
		return err
	}
	return nil
}

// WriteMetrics writes metrics definitions in a human-readable text format.
func (p *TextProvider) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	renderModel := schema.BuildMetricsRenderModel(activeWeights)

	if _, err := fmt.Fprintln(w, renderModel.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, strings.Repeat("-", len(renderModel.Title))); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, renderModel.Description); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for _, m := range renderModel.Modes {
		if _, err := fmt.Fprintf(w, "%s: %s\n", GetDisplayNameForMode(m.Name), m.Purpose); err != nil {
			return err
		}
		factors := strings.Join(m.Factors, ", ")
		if _, err := fmt.Fprintf(w, "  Factors: %s\n", factors); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Formula: %s\n", m.Formula); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

// WriteHistory writes analysis history in a human-readable table.
func (p *TextProvider) WriteHistory(w io.Writer, runs []schema.AnalysisRunRecord, _ config.OutputSettings) error {
	if len(runs) == 0 {
		if _, err := fmt.Fprintln(w, "No analysis history found."); err != nil {
			return err
		}
		return nil
	}

	table := tablewriter.NewWriter(w)
	defer func() { _ = table.Close() }()

	table.Header([]string{"ID", "Start Time", "Duration", "Files", "URN"})
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.Row.Alignment.Global = tw.AlignLeft
	})

	var data [][]string
	for _, r := range runs {
		duration := "running"
		if r.RunDurationMs != nil {
			duration = fmt.Sprintf("%dms", *r.RunDurationMs)
		}
		files := "0"
		if r.TotalFilesAnalyzed != nil {
			files = strconv.Itoa(int(*r.TotalFilesAnalyzed))
		}
		data = append(data, []string{
			strconv.FormatInt(r.AnalysisID, 10),
			r.StartTime.Format("2006-01-02 15:04:05"),
			duration,
			files,
			r.URN,
		})
	}

	if err := table.Bulk(data); err != nil {
		return err
	}
	return table.Render()
}
