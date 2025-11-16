package outwriter

import (
	"fmt"

	"github.com/huangsam/hotspot/schema"
)

// formatOwnershipDiff computes and formats the ownership difference between before and after states
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
