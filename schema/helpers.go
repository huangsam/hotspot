package schema

import (
	"sort"
	"strings"
	"unicode"
)

// cleanParts cleans a slice of name parts by trimming non-alphanumeric punctuation from ends,
// and additionally trims trailing periods for looser handling.
func cleanParts(parts []string) []string {
	var cleaned []string
	for _, p := range parts {
		cp := strings.TrimFunc(p, func(r rune) bool {
			if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '\'' || r == '.' {
				return false
			}
			return true
		})
		cp = strings.TrimSuffix(cp, ".")
		if cp != "" {
			cleaned = append(cleaned, cp)
		}
	}
	return cleaned
}

// getInitial extracts the initial from the last name part, using the first rune for Unicode safety.
func getInitial(last string) string {
	rr := []rune(last)
	if len(rr) > 0 {
		return string(rr[0])
	}
	return ""
}

// AbbreviateName formats "Samuel Huang" to "Samuel H".
// It handles names with parentheses, quotes, backticks, hyphens, and apostrophes appropriately.
// It also handles single-word names by returning them unchanged, and bot accounts without abbreviation.
func AbbreviateName(name string) string {
	// Trim leading/trailing whitespace.
	trimmedName := strings.TrimSpace(name)

	// Special case: bot accounts (e.g., dependabot[bot]) are not abbreviated.
	if strings.Contains(name, "[bot]") {
		parts := strings.Fields(trimmedName)
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
		return trimmedName
	}

	// Remove outer punctuation.
	trimmedName = strings.Trim(trimmedName, "()\"'`")

	// Split into parts.
	parts := strings.Fields(trimmedName)
	cleaned := cleanParts(parts)

	// Handle based on number of cleaned parts.
	if len(cleaned) >= 2 {
		first := cleaned[0]
		last := cleaned[len(cleaned)-1]
		initial := getInitial(last)
		if initial != "" {
			return first + " " + initial
		}
		return first
	}

	if len(cleaned) == 1 {
		return cleaned[0]
	}

	// Fallback.
	return trimmedName
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
