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
		headers = append(headers, "Commits", "Churn", "LOC")
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
