package internal

import (
	"path/filepath"
	"strings"
)

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
