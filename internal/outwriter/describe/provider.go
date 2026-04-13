// Package describe provides a FormatProvider implementation for executive summary output.
package describe

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
)

// Provider implements the util.FormatProvider interface for executive summary output.
type Provider struct{}

// NewProvider creates a new describe provider.
func NewProvider() *Provider {
	return &Provider{}
}

// WriteFiles writes the analysis results in a markdown-based executive summary.
func (p *Provider) WriteFiles(w io.Writer, files []schema.FileResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	if _, err := fmt.Fprintln(w, "# Repository Health Executive Summary"); err != nil {
		return err
	}

	// 1. Surfacing High-Criticality Hotspots
	if _, err := fmt.Fprintln(w, "\n## Critical Hotspots"); err != nil {
		return err
	}

	foundCritical := false
	for _, f := range files {
		if f.ModeScore >= 80 {
			foundCritical = true
			if err := p.writeFileSummary(w, f); err != nil {
				return err
			}
		}
	}
	if !foundCritical {
		if _, err := fmt.Fprintln(w, "_No critical hotspots identified in current analysis window._"); err != nil {
			return err
		}
	}

	// 2. Surfacing Moderate Risks
	if _, err := fmt.Fprintln(w, "\n## Moderate Risks"); err != nil {
		return err
	}

	foundModerate := false
	for _, f := range files {
		if f.ModeScore >= 40 && f.ModeScore < 80 {
			foundModerate = true
			if err := p.writeFileSummary(w, f); err != nil {
				return err
			}
		}
	}
	if !foundModerate {
		if _, err := fmt.Fprintln(w, "_No significant risks identified in current window._"); err != nil {
			return err
		}
	}

	// 3. Overall Repository Health Insight
	if _, err := fmt.Fprintln(w, "\n## Strategic Recommendations (A2A Intent)"); err != nil {
		return err
	}
	if len(files) > 0 {
		topFile := files[0]
		reasoning := "Monitor for further volatility."
		if len(topFile.Reasoning) > 0 {
			reasoning = topFile.Reasoning[0]
		}
		if _, err := fmt.Fprintf(w, "- **Priority 1**: The top hotspot is `%s` (Score: %.1f). Recommended action: %s\n", topFile.Path, topFile.ModeScore, reasoning); err != nil {
			return err
		}
	}

	return nil
}

// WriteFolders is not specifically implemented for describe mode, fallback to no-op or message.
func (p *Provider) WriteFolders(w io.Writer, _ []schema.FolderResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	_, err := fmt.Fprintln(w, "Describe mode is not supported for folder analysis.")
	return err
}

// WriteComparison is not specifically implemented for describe mode.
func (p *Provider) WriteComparison(w io.Writer, _ schema.ComparisonResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	_, err := fmt.Fprintln(w, "Describe mode is not supported for comparison analysis.")
	return err
}

// WriteTimeseries is not specifically implemented for describe mode.
func (p *Provider) WriteTimeseries(w io.Writer, _ schema.TimeseriesResult, _ config.OutputSettings, _ config.RuntimeSettings, _ time.Duration) error {
	_, err := fmt.Fprintln(w, "Describe mode is not supported for timeseries analysis.")
	return err
}

// WriteMetrics is not specifically implemented for describe mode.
func (p *Provider) WriteMetrics(w io.Writer, _ map[schema.ScoringMode]map[schema.BreakdownKey]float64, _ config.OutputSettings) error {
	_, err := fmt.Fprintln(w, "Describe mode is not supported for metrics definitions.")
	return err
}

func (p *Provider) writeFileSummary(w io.Writer, f schema.FileResult) error {
	reasoning := "No specific reasons identified."
	if len(f.Reasoning) > 0 {
		reasoning = strings.Join(f.Reasoning, " ")
	}
	_, err := fmt.Fprintf(w, "- **`%s`** (Score: %.1f): %s\n", f.Path, f.ModeScore, reasoning)
	return err
}
