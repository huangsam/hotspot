package internal

import (
	"strings"
	"testing"
)

// TestShouldIgnore tests path exclusion logic
func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		excludes   []string
		wantIgnore bool
	}{
		{
			name:       "empty excludes",
			path:       "src/main.go",
			excludes:   []string{},
			wantIgnore: false,
		},
		{
			name:       "prefix match",
			path:       "vendor/github.com/lib/file.go",
			excludes:   []string{"vendor/"},
			wantIgnore: true,
		},
		{
			name:       "suffix match",
			path:       "dist/bundle.min.js",
			excludes:   []string{".min.js"},
			wantIgnore: true,
		},
		{
			name:       "glob match basename",
			path:       "src/file.min.js",
			excludes:   []string{"*.min.js"},
			wantIgnore: true,
		},
		{
			name:       "glob match with test suffix",
			path:       "test/unit_test.go",
			excludes:   []string{"*_test.go"},
			wantIgnore: true,
		},
		{
			name:       "substring match",
			path:       "src/generated/code.go",
			excludes:   []string{"generated"},
			wantIgnore: true,
		},
		{
			name:       "no match",
			path:       "src/core/engine.go",
			excludes:   []string{"vendor/", "node_modules/", ".min.js"},
			wantIgnore: false,
		},
		{
			name:       "multiple excludes with match",
			path:       "node_modules/react/index.js",
			excludes:   []string{"vendor/", "node_modules/", "third_party/"},
			wantIgnore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIgnore(tt.path, tt.excludes)
			if got != tt.wantIgnore {
				t.Errorf("ShouldIgnore(%q, %v) = %v, want %v", tt.path, tt.excludes, got, tt.wantIgnore)
			}
		})
	}
}

// TestTruncatePath tests path truncation logic
func TestTruncatePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		maxWidth int
		wantLen  int
	}{
		{
			name:     "short path no truncation",
			path:     "main.go",
			maxWidth: 20,
			wantLen:  7,
		},
		{
			name:     "exact length no truncation",
			path:     "src/main.go",
			maxWidth: 11,
			wantLen:  11,
		},
		{
			name:     "long path truncated",
			path:     "very/long/path/to/some/deeply/nested/file.go",
			maxWidth: 20,
			wantLen:  20,
		},
		{
			name:     "unicode characters",
			path:     "src/文件/test.go",
			maxWidth: 10,
			wantLen:  10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncatePath(tt.path, tt.maxWidth)
			gotLen := len([]rune(got))

			// Verify result length is within bounds
			if gotLen > tt.maxWidth {
				t.Errorf("truncatePath(%q, %d) produced %q with length %d, exceeds maxWidth",
					tt.path, tt.maxWidth, got, gotLen)
			}

			// For paths that should be truncated, verify they start with "..."
			if len([]rune(tt.path)) > tt.maxWidth {
				if !strings.HasPrefix(got, "...") {
					t.Errorf("truncatePath(%q, %d) = %q, expected to start with '...'",
						tt.path, tt.maxWidth, got)
				}
			}

			// Verify expected length
			if gotLen != tt.wantLen {
				t.Errorf("truncatePath(%q, %d) length = %d, want %d (result: %q)",
					tt.path, tt.maxWidth, gotLen, tt.wantLen, got)
			}
		})
	}
}

// TestGetTextLabel tests criticality label assignment
func TestGetTextLabel(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{0, "Low"},
		{39.9, "Low"},
		{40, "Moderate"},
		{59.9, "Moderate"},
		{60, "High"},
		{79.9, "High"},
		{80, "Critical"},
		{100, "Critical"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := getTextLabel(tt.score)
			if got != tt.want {
				t.Errorf("getTextLabel(%f) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}
