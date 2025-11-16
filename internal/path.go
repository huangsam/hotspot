package internal

import (
	"github.com/huangsam/hotspot/internal/contract"
)

// ShouldIgnore returns true if the given path matches any of the exclude patterns.
// It supports simple glob patterns (using filepath.Match) when the pattern
// contains wildcard characters (*, ?, [ ]). Patterns ending with '/' are treated
// as prefixes. Patterns starting with '.' are treated as suffix (extension) matches.
// A user can provide patterns like "vendor/", "node_modules/", "*.min.js".
func ShouldIgnore(path string, excludes []string) bool {
	return contract.ShouldIgnore(path, excludes)
}

// truncatePath truncates a file path to a maximum width with ellipsis prefix.
// Requires maxWidth > 3 to ensure there's space for both the "..." prefix and at least one character of content.
// Without this check, small maxWidth values could cause slice bounds errors in the truncation calculation.
func truncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth && maxWidth > 3 {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}
