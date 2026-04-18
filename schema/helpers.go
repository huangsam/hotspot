package schema

import (
	"fmt"
	"path/filepath"
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

// AbbreviateOwners applies abbreviation to all owners in the slice.
func AbbreviateOwners(owners []string) []string {
	abbreviated := make([]string, len(owners))
	for i, owner := range owners {
		abbreviated[i] = AbbreviateName(owner)
	}
	return abbreviated
}

// FormatOwners formats the top owners as "S. Huang, J. Doe".
func FormatOwners(owners []string) string {
	var abbreviated []string
	for _, owner := range owners {
		abbreviated = append(abbreviated, AbbreviateName(owner))
	}
	return strings.Join(abbreviated, ", ")
}

// OwnersEqual compares two slices of owners, considering them equal if they contain the same owners
// regardless of order.
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

// PathMatcher provides efficient path matching against a set of exclude patterns.
// It pre-processes patterns to avoid redundant string operations during matching.
type PathMatcher struct {
	rules []ignoreRule
}

type ignoreKind int

const (
	kindEverything ignoreKind = iota
	kindPrefix
	kindSuffix
	kindSubstring
	kindDirComponent
	kindGlob
)

type ignoreRule struct {
	pattern string
	kind    ignoreKind
}

// NewPathMatcher creates a new PathMatcher from a set of exclude patterns.
func NewPathMatcher(excludes []string) *PathMatcher {
	var rules []ignoreRule
	for _, ex := range excludes {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}

		// Normalize to forward slashes
		ex = filepath.ToSlash(ex)

		switch {
		case ex == "**":
			rules = append(rules, ignoreRule{kind: kindEverything})
		case strings.HasPrefix(ex, "**/"):
			// Recursive directory or glob
			pattern := ex[3:]
			switch {
			case pattern == "":
				rules = append(rules, ignoreRule{kind: kindEverything})
			case strings.ContainsAny(pattern, "*?["):
				rules = append(rules, ignoreRule{pattern: pattern, kind: kindGlob})
			default:
				rules = append(rules, ignoreRule{pattern: pattern, kind: kindDirComponent})
			}
		case strings.HasSuffix(ex, "/**"):
			rules = append(rules, ignoreRule{pattern: ex[:len(ex)-3], kind: kindPrefix})
		case strings.ContainsAny(ex, "*?["):
			rules = append(rules, ignoreRule{pattern: ex, kind: kindGlob})
		case strings.HasSuffix(ex, "/"):
			rules = append(rules, ignoreRule{pattern: ex, kind: kindDirComponent})
		case strings.HasPrefix(ex, "."):
			rules = append(rules, ignoreRule{pattern: ex, kind: kindSuffix})
		default:
			rules = append(rules, ignoreRule{pattern: ex, kind: kindSubstring})
		}
	}
	return &PathMatcher{rules: rules}
}

// Match returns true if the path matches any of the exclude patterns.
func (m *PathMatcher) Match(path string) bool {
	if m == nil || len(m.rules) == 0 {
		return false
	}

	// Normalize path once
	path = filepath.ToSlash(path)
	if path == "" {
		return false
	}

	for _, rule := range m.rules {
		switch rule.kind {
		case kindEverything:
			return true
		case kindPrefix:
			if strings.HasPrefix(path, rule.pattern) {
				return true
			}
		case kindSuffix:
			if strings.HasSuffix(path, rule.pattern) {
				return true
			}
		case kindSubstring:
			if strings.Contains(path, rule.pattern) {
				return true
			}
		case kindDirComponent:
			// Match if it's the full path, a prefix, or a component in the middle.
			// rule.pattern already has a trailing slash if it was "dir/" or "**/dir/".
			if path == strings.TrimSuffix(rule.pattern, "/") || strings.HasPrefix(path, rule.pattern) || strings.Contains(path, "/"+rule.pattern) {
				return true
			}
		case kindGlob:
			// Try matching full path
			if ok, err := filepath.Match(rule.pattern, path); err == nil && ok {
				return true
			}
			// Try matching against the base filename
			if ok, err := filepath.Match(rule.pattern, filepath.Base(path)); err == nil && ok {
				return true
			}
			// If it's a recursive glob like **/*.go, we also need to match path components
			if strings.Contains(path, "/") {
				parts := strings.SplitSeq(path, "/")
				for part := range parts {
					if ok, err := filepath.Match(rule.pattern, part); err == nil && ok {
						return true
					}
				}
			}
		}
	}
	return false
}

// ShouldIgnore returns true if the given path matches any of the exclude patterns.
// This is a convenience wrapper around PathMatcher for one-off checks.
// For repeated checks, create a PathMatcher using NewPathMatcher instead.
func ShouldIgnore(path string, excludes []string) bool {
	if len(excludes) == 0 {
		return false
	}
	return NewPathMatcher(excludes).Match(path)
}

// ParseBoolString parses a string value into a boolean.
// Accepts "yes", "no", "true", "false", "1", "0" (case-insensitive).
// Returns an error for invalid values.
func ParseBoolString(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "yes", "true", "1":
		return true, nil
	case "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean string: %s (expected yes/no/true/false/1/0)", s)
	}
}

// NormalizeTimeseriesPath normalizes a user-provided path relative to the repo root
// and ensures it's within the repository boundaries.
func NormalizeTimeseriesPath(repoPath, userPath string) (string, error) {
	// Handle absolute paths by making them relative to repo
	if filepath.IsAbs(userPath) {
		relPath, err := filepath.Rel(repoPath, userPath)
		if err != nil {
			return "", fmt.Errorf("path is outside repository: %s", userPath)
		}
		userPath = relPath
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(userPath)

	// Ensure the path doesn't go outside the repo (no leading .. after cleaning)
	if strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("path is outside repository: %s", userPath)
	}

	// Convert to forward slashes for consistency with Git paths
	normalized := strings.ReplaceAll(cleanPath, string(filepath.Separator), "/")

	// Remove leading ./ if present
	normalized = strings.TrimPrefix(normalized, "./")

	return normalized, nil
}
