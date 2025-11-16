// Package outwriter has output and writer logic.
package outwriter

import (
	"fmt"
	"os"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"golang.org/x/term"
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
	return PrintFileResults(results, cfg, duration)
}

// WriteFolders prints folder analysis results using the configured output format.
func (ow *OutWriter) WriteFolders(results []schema.FolderResult, cfg *contract.Config, duration time.Duration) error {
	return PrintFolderResults(results, cfg, duration)
}

// WriteComparison prints comparison analysis results using the configured output format.
func (ow *OutWriter) WriteComparison(results schema.ComparisonResult, cfg *contract.Config, duration time.Duration) error {
	return PrintComparisonResults(results, cfg, duration)
}

// WriteTimeseries prints timeseries analysis results using the configured output format.
func (ow *OutWriter) WriteTimeseries(result schema.TimeseriesResult, cfg *contract.Config, duration time.Duration) error {
	return PrintTimeseriesResults(result, cfg, duration)
}

// WriteMetrics prints metrics definitions using the configured output format.
func (ow *OutWriter) WriteMetrics(activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64, cfg *contract.Config) error {
	return PrintMetricsDefinitions(activeWeights, cfg)
}

// GetMaxTablePathWidth calculates the maximum width for file paths in table output
// based on terminal width and table configuration.
func GetMaxTablePathWidth(cfg *contract.Config) int {
	var termWidth int

	// Check for absolute width override from flag/env
	if cfg.Width > 0 {
		termWidth = cfg.Width
	}

	if termWidth == 0 { // Not set by override
		// Get terminal width
		detectedWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || detectedWidth <= 0 {
			// Fallback to conservative default if terminal size can't be detected
			termWidth = 80 // Conservative default for narrow terminals and CI
		} else {
			termWidth = detectedWidth
		}
	}

	// Reserve space for fixed columns with table formatting
	baseWidth := 25 // Rank + Score + Label with borders/padding

	// Add detail columns with formatting
	if cfg.Detail {
		baseWidth += 55 // All detail columns (Contrib + Commits + LOC + Churn + Age + Gini) with formatting
	}

	// Add explain column
	if cfg.Explain {
		baseWidth += 35 // Explain column with formatting
	}

	// Add owner column
	if cfg.Owner {
		baseWidth += 25 // Owner column with formatting
	}

	// Reserve generous space for table borders, separators, and padding
	baseWidth += 20

	// Calculate available space for path
	available := termWidth - baseWidth
	if available < 15 {
		// Minimum reasonable path width
		return 15
	}
	if available > 70 {
		// Maximum path width to prevent overly long paths
		return 70
	}
	return available
}

// formatOwnershipDiff computes and formats the ownership difference between before and after states.
func formatOwnershipDiff(r schema.ComparisonDetails) string {
	beforeOwners := r.BeforeOwners
	afterOwners := r.AfterOwners

	switch r.Status {
	case schema.NewStatus:
		// New file - show current owners
		if len(afterOwners) > 0 {
			return fmt.Sprintf("New: %s", schema.FormatOwners(afterOwners))
		}
		return "New"

	case schema.InactiveStatus:
		// Inactive file - show previous owners
		if len(beforeOwners) > 0 {
			return fmt.Sprintf("Removed: %s", schema.FormatOwners(beforeOwners))
		}
		return "Removed"

	default:
		// Active file - compare ownership stability
		if len(afterOwners) > 0 {
			if schema.OwnersEqual(beforeOwners, afterOwners) {
				return fmt.Sprintf("%s (stable)", schema.FormatOwners(afterOwners))
			}
			return schema.FormatOwners(afterOwners)
		}
		if len(beforeOwners) > 0 {
			return fmt.Sprintf("No owners (was: %s)", schema.FormatOwners(beforeOwners))
		}
		return "No owners"
	}
}
