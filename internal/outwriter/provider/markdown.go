// Package provider implements the FormatProvider implementation for Markdown output.
package provider

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// MarkdownProvider implements the FormatProvider interface for Markdown output.
type MarkdownProvider struct{}

// NewMarkdownProvider creates a new markdown provider.
func NewMarkdownProvider() *MarkdownProvider {
	return &MarkdownProvider{}
}

// WriteFiles writes file analysis results in Markdown format.
func (p *MarkdownProvider) WriteFiles(w io.Writer, files []schema.FileResult, output config.OutputSettings, _ config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())

	if _, err := fmt.Fprintln(w, "## File Hotspots"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

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

	p.writeMarkdownTable(w, headers)

	for i, f := range files {
		row := []string{
			strconv.Itoa(i + 1),
			f.Path,
			fmtFloat(f.ModeScore),
			schema.GetPlainLabel(f.ModeScore),
		}
		if output.IsDetail() {
			row = append(row,
				f.UniqueContributors.Display(),
				f.Commits.Display(),
				f.LinesOfCode.Display(),
				f.Churn.Display(),
				f.AgeDays.Display(),
				fmtFloat(f.Gini),
			)
		}
		if output.IsExplain() {
			row = append(row, FormatTopMetricBreakdown(&f))
		}
		if output.IsOwner() {
			row = append(row, strings.Join(f.Owners, ", "))
		}
		p.writeMarkdownRow(w, row)
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "*Showing top %d files. Analysis completed in %v.*\n", len(files), duration); err != nil {
		return err
	}
	return nil
}

// WriteFolders writes folder analysis results in Markdown format.
func (p *MarkdownProvider) WriteFolders(w io.Writer, results []schema.FolderResult, output config.OutputSettings, _ config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())

	if _, err := fmt.Fprintln(w, "## Folder Hotspots"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	headers := []string{"Rank", "Path", "Score", "Label"}
	if output.IsDetail() {
		headers = append(headers, "Commits", "Churn", "LOC", "Contrib", "Gini")
	}
	if output.IsOwner() {
		headers = append(headers, "Owner")
	}

	p.writeMarkdownTable(w, headers)

	for i, r := range results {
		row := []string{
			strconv.Itoa(i + 1),
			r.Path,
			fmtFloat(r.Score),
			schema.GetPlainLabel(r.Score),
		}
		if output.IsDetail() {
			row = append(row,
				r.Commits.Display(),
				r.Churn.Display(),
				r.TotalLOC.Display(),
				r.UniqueContributors.Display(),
				fmtFloat(r.Gini),
			)
		}
		if output.IsOwner() {
			row = append(row, strings.Join(r.Owners, ", "))
		}
		p.writeMarkdownRow(w, row)
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "*Showing top %d folders. Analysis completed in %v.*\n", len(results), duration); err != nil {
		return err
	}
	return nil
}

// WriteComparison writes comparison analysis results in Markdown format.
func (p *MarkdownProvider) WriteComparison(w io.Writer, results schema.ComparisonResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())

	if _, err := fmt.Fprintln(w, "## Comparison Results"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	headers := []string{"Rank", "Path", "Before", "After", "Delta", "Status"}
	if output.IsDetail() {
		headers = append(headers, "Δ Churn")
	}
	if output.IsOwner() {
		headers = append(headers, "Ownership")
	}

	p.writeMarkdownTable(w, headers)

	for i, r := range results.Details {
		// Use plain labels for markdown
		row := []string{
			strconv.Itoa(i + 1),
			r.Path,
			fmtFloat(r.BeforeScore),
			fmtFloat(r.AfterScore),
			fmt.Sprintf("%.*f", output.GetPrecision(), r.Delta),
			string(r.Status),
		}
		if output.IsDetail() {
			row = append(row, r.DeltaChurn.Display())
		}
		if output.IsOwner() {
			row = append(row, FormatOwnershipDiff(r))
		}
		p.writeMarkdownRow(w, row)
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "**Summary**\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Net score delta: %.*f\n", output.GetPrecision(), results.Summary.NetScoreDelta); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Net churn delta: %s\n", results.Summary.NetChurnDelta.Display()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- New files: %d\n", results.Summary.TotalNewFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Modified files: %d\n", results.Summary.TotalModifiedFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- Ownership changes: %d\n", results.Summary.TotalOwnershipChanges); err != nil {
		return err
	}

	return nil
}

// WriteTimeseries writes timeseries analysis results in Markdown format.
func (p *MarkdownProvider) WriteTimeseries(w io.Writer, result schema.TimeseriesResult, output config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())

	if _, err := fmt.Fprintln(w, "## Timeseries Analysis"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	headers := []string{"Rank", "Path", "Period", "Score", "Mode", "Owner"}
	p.writeMarkdownTable(w, headers)

	for i, pt := range result.Points {
		ownersStr := "No owners"
		if len(pt.Owners) > 0 {
			ownersStr = strings.Join(pt.Owners, ", ")
		}
		row := []string{
			strconv.Itoa(i + 1),
			pt.Path,
			pt.Period,
			fmtFloat(pt.Score),
			string(pt.Mode),
			ownersStr,
		}
		p.writeMarkdownRow(w, row)
	}

	return nil
}

// WriteBlastRadius writes blast radius analysis results in Markdown format.
func (p *MarkdownProvider) WriteBlastRadius(w io.Writer, result schema.BlastRadiusResult, output config.OutputSettings, _ config.RuntimeSettings, duration time.Duration) error {
	fmtFloat := CreateFormatters(output.GetPrecision())

	if _, err := fmt.Fprintln(w, "## Blast Radius Analysis"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Found **%d** coupled pairs above threshold **%v**.\n\n", result.Summary.TotalPairs, result.Summary.Threshold); err != nil {
		return err
	}

	headers := []string{"Rank", "File A", "File B", "Score", "Co-Change"}
	p.writeMarkdownTable(w, headers)

	for i, pair := range result.Pairs {
		row := []string{
			strconv.Itoa(i + 1),
			pair.Source,
			pair.Target,
			fmtFloat(pair.Score),
			strconv.Itoa(pair.CoChange),
		}
		p.writeMarkdownRow(w, row)
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "*Blast radius analysis completed in %v. Total commits analyzed: %d.*\n", duration, result.Summary.TotalCommits); err != nil {
		return err
	}
	return nil
}

// WriteMetrics writes metrics definitions in Markdown format.
func (p *MarkdownProvider) WriteMetrics(w io.Writer, activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	renderModel := schema.BuildMetricsRenderModel(activeWeights)

	if _, err := fmt.Fprintf(w, "## %s\n\n", renderModel.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n\n", renderModel.Description); err != nil {
		return err
	}

	headers := []string{"Mode", "Purpose", "Factors", "Formula"}
	p.writeMarkdownTable(w, headers)

	for _, m := range renderModel.Modes {
		row := []string{
			m.Name,
			m.Purpose,
			strings.Join(m.Factors, ", "),
			m.Formula,
		}
		p.writeMarkdownRow(w, row)
	}

	return nil
}

// WriteHistory writes analysis history in Markdown format.
func (p *MarkdownProvider) WriteHistory(w io.Writer, runs []schema.AnalysisRunRecord, _ config.OutputSettings) error {
	if _, err := fmt.Fprintln(w, "## Analysis History"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if len(runs) == 0 {
		if _, err := fmt.Fprintln(w, "No analysis history found."); err != nil {
			return err
		}
		return nil
	}

	headers := []string{"ID", "Start Time", "Duration", "Files", "URN"}
	p.writeMarkdownTable(w, headers)

	for _, r := range runs {
		duration := "running"
		if r.RunDurationMs != nil {
			duration = fmt.Sprintf("%dms", *r.RunDurationMs)
		}
		files := "0"
		if r.TotalFilesAnalyzed != nil {
			files = strconv.Itoa(int(*r.TotalFilesAnalyzed))
		}
		row := []string{
			strconv.FormatInt(r.AnalysisID, 10),
			r.StartTime.Format("2006-01-02 15:04:05"),
			duration,
			files,
			r.URN,
		}
		p.writeMarkdownRow(w, row)
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "*Showing %d historical runs.*\n", len(runs)); err != nil {
		return err
	}
	return nil
}

// WriteBatch writes repository shapes in Markdown format.
func (p *MarkdownProvider) WriteBatch(w io.Writer, results []schema.RepoShape, _ config.OutputSettings) error {
	if _, err := fmt.Fprintln(w, "## Fleet Analysis Summary"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	headers := []string{"Repository", "Preset", "Mode", "Files", "Commits", "Owners"}
	p.writeMarkdownTable(w, headers)

	for _, s := range results {
		repoName := strings.TrimPrefix(s.URN, "git:")
		row := []string{
			repoName,
			string(s.RecommendedPreset),
			string(s.Preset.Mode),
			strconv.Itoa(s.FileCount),
			fmt.Sprintf("%.0f", s.TotalCommits),
			strconv.Itoa(s.UniqueContributors),
		}
		p.writeMarkdownRow(w, row)
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "*Batch analysis of %d repositories completed.*\n", len(results)); err != nil {
		return err
	}
	return nil
}

func (p *MarkdownProvider) writeMarkdownTable(w io.Writer, headers []string) {
	_, _ = fmt.Fprintf(w, "| %s |\n", strings.Join(headers, " | "))
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	_, _ = fmt.Fprintf(w, "| %s |\n", strings.Join(sep, " | "))
}

func (p *MarkdownProvider) writeMarkdownRow(w io.Writer, columns []string) {
	_, _ = fmt.Fprintf(w, "| %s |\n", strings.Join(columns, " | "))
}
