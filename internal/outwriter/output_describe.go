package outwriter

import (
	"fmt"
	"io"
	"strings"

	"github.com/huangsam/hotspot/schema"
)

// writeDescribeResultsForFiles writes the analysis results in a markdown-based executive summary.
func writeDescribeResultsForFiles(w io.Writer, files []schema.FileResult) error {
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
			if err := writeFileSummary(w, f); err != nil {
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
			if err := writeFileSummary(w, f); err != nil {
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

func writeFileSummary(w io.Writer, f schema.FileResult) error {
	reasoning := "No specific reasons identified."
	if len(f.Reasoning) > 0 {
		reasoning = strings.Join(f.Reasoning, " ")
	}
	_, err := fmt.Fprintf(w, "- **`%s`** (Score: %.1f): %s\n", f.Path, f.ModeScore, reasoning)
	return err
}
