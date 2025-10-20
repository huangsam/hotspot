// Package internal has helpers that are only useful within the hotspot runtime.
package internal

import (
	"path/filepath"
	"strings"
)

// truncatePath truncates a file path to a maximum width with ellipsis prefix.
func truncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}

// getTextLabel returns a text label indicating the criticality level
// based on the file's importance score:
// - Critical (≥80)
// - High (≥60)
// - Moderate (≥40)
// - Low (<40)
func getTextLabel(score float64) string {
	switch {
	case score >= 80:
		return "Critical"
	case score >= 60:
		return "High"
	case score >= 40:
		return "Moderate"
	default:
		return "Low"
	}
}

// ShouldIgnore returns true if the given path matches any of the exclude patterns.
// It supports simple glob patterns (using filepath.Match) when the pattern
// contains wildcard characters (*, ?, [ ]). Patterns ending with '/' are treated
// as prefixes. Patterns starting with '.' are treated as suffix (extension) matches.
// A user can provide patterns like "vendor/", "node_modules/", "*.min.js".
func ShouldIgnore(path string, excludes []string) bool {
	for _, ex := range excludes {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}

		// If the pattern contains glob characters, try filepath.Match.
		if strings.ContainsAny(ex, "*?[") || strings.Contains(ex, "**") {
			pat := strings.ReplaceAll(ex, "**", "*")
			if ok, err := filepath.Match(pat, path); err == nil && ok {
				return true
			}
			// Also try matching against the base filename (e.g. *.min.js)
			if ok, err := filepath.Match(pat, filepath.Base(path)); err == nil && ok {
				return true
			}
			continue
		}

		// Handle prefix, suffix, or substring matches
		switch {
		case strings.HasSuffix(ex, "/"):
			if strings.HasPrefix(path, ex) {
				return true
			}
		case strings.HasPrefix(ex, "."):
			if strings.HasSuffix(path, ex) {
				return true
			}
		case strings.Contains(path, ex):
			return true
		}
	}
	return false
}
