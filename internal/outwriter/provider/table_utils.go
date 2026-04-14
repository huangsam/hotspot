package provider

import (
	"fmt"
	"os"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
	"golang.org/x/term"
)

// GetMaxTablePathWidth calculates the maximum width for file paths in table output
// based on terminal width and table configuration.
func GetMaxTablePathWidth(output config.OutputSettings) int {
	var termWidth int

	// Check for absolute width override from flag/env
	if output.GetWidth() > 0 {
		termWidth = output.GetWidth()
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
	if output.IsDetail() {
		baseWidth += 55 // All detail columns (Contrib + Commits + LOC + Churn + Age + Gini) with formatting
	}

	// Add explain column
	if output.IsExplain() {
		baseWidth += 35 // Explain column with formatting
	}

	// Add owner column
	if output.IsOwner() {
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

// FormatOwnershipDiff computes and formats the ownership difference between before and after states.
func FormatOwnershipDiff(r schema.ComparisonDetail) string {
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
