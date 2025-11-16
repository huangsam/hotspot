package contract

import (
	"strings"
	"testing"
)

// FuzzShouldIgnore fuzzes the ShouldIgnore function with random paths and exclude patterns.
func FuzzShouldIgnore(f *testing.F) {
	seeds := []struct {
		path     string
		excludes string // comma-separated
	}{
		{"main.go", "*.log"},
		{"vendor/package/file.go", "vendor/"},
		{"test_file.min.js", "*.min.js"},
		{"config.json", ".json"},
		{"", ""},
		{"very/long/path/to/file.txt", "**/temp/**"},
	}
	for _, seed := range seeds {
		f.Add(seed.path, seed.excludes)
	}

	f.Fuzz(func(_ *testing.T, path string, excludesStr string) {
		excludes := []string{}
		if excludesStr != "" {
			// Simple split, may not handle complex cases but good for fuzzing
			for ex := range strings.SplitSeq(excludesStr, ",") {
				if trimmed := strings.TrimSpace(ex); trimmed != "" {
					excludes = append(excludes, trimmed)
				}
			}
		}
		_ = ShouldIgnore(path, excludes)
	})
}
