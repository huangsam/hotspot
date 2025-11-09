package schema

import (
	"sort"
	"strings"
)

// AbbreviateName formats "Samuel Huang" to "Samuel H"
func AbbreviateName(name string) string {
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		return strings.Join([]string{parts[0], string(parts[1][0])}, " ")
	}
	return name
}

// AbbreviateOwners applies abbreviation to all owners in the slice
func AbbreviateOwners(owners []string) []string {
	abbreviated := make([]string, len(owners))
	for i, owner := range owners {
		abbreviated[i] = AbbreviateName(owner)
	}
	return abbreviated
}

// FormatOwners formats the top owners as "S. Huang, J. Doe"
func FormatOwners(owners []string) string {
	var abbreviated []string
	for _, owner := range owners {
		abbreviated = append(abbreviated, AbbreviateName(owner))
	}
	return strings.Join(abbreviated, ", ")
}

// OwnersEqual compares two slices of owners, considering them equal if they contain the same owners
// regardless of order
func OwnersEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create sorted copies for comparison
	aSorted := make([]string, len(a))
	copy(aSorted, a)
	sort.Strings(aSorted)

	bSorted := make([]string, len(b))
	copy(bSorted, b)
	sort.Strings(bSorted)

	for i := range aSorted {
		if aSorted[i] != bSorted[i] {
			return false
		}
	}
	return true
}
