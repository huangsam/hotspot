package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlainLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{
			name:     "smallest value possible",
			input:    0.0,
			expected: LowValue,
		},
		{
			name:     "just before moderate",
			input:    39.9,
			expected: LowValue,
		},
		{
			name:     "exactly moderate",
			input:    40.0,
			expected: ModerateValue,
		},
		{
			name:     "just before high",
			input:    59.9,
			expected: ModerateValue,
		},
		{
			name:     "exactly high",
			input:    60.0,
			expected: HighValue,
		},
		{
			name:     "just before critical",
			input:    79.9,
			expected: HighValue,
		},
		{
			name:     "exactly critical",
			input:    80.0,
			expected: CriticalValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetPlainLabel(tt.input))
		})
	}
}

func TestGetColorLabel(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		label string
	}{
		{"low", 30, LowValue},
		{"moderate", 50, ModerateValue},
		{"high", 70, HighValue},
		{"critical", 90, CriticalValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetColorLabel(tt.score)
			// Should contain the plain label
			assert.Contains(t, result, tt.label)
		})
	}
}

func TestSelectOutputFile(t *testing.T) {
	t.Run("empty path returns stdout", func(t *testing.T) {
		file, err := SelectOutputFile("")
		require.NoError(t, err)
		assert.Equal(t, os.Stdout, file)
	})

	t.Run("valid path creates file", func(t *testing.T) {
		tempFile := filepath.Join(os.TempDir(), "test_output.txt")
		defer func() { _ = os.Remove(tempFile) }() // cleanup

		file, err := SelectOutputFile(tempFile)
		require.NoError(t, err)
		assert.NotNil(t, file)
		_ = file.Close()

		// Verify file was created
		_, err = os.Stat(tempFile)
		assert.NoError(t, err)
	})
}

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
			assert.Equal(t, tt.wantIgnore, got)
		})
	}
}

func TestGetDBFilePath(t *testing.T) {
	path := GetDBFilePath()

	// Should not be empty
	assert.NotEmpty(t, path)

	// Should contain the database name
	assert.Contains(t, path, ".hotspot_cache.db")

	// Should be in home directory
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, homeDir), "path %s should start with home dir %s", path, homeDir)
}

func TestNormalizeTimeseriesPath(t *testing.T) {
	repoPath := "/home/user/project"

	tests := []struct {
		name        string
		userPath    string
		expected    string
		expectError bool
	}{
		{
			name:     "relative path",
			userPath: "src/main.go",
			expected: "src/main.go",
		},
		{
			name:     "relative path with dot",
			userPath: "./src/main.go",
			expected: "src/main.go",
		},
		{
			name:     "absolute path within repo",
			userPath: "/home/user/project/src/main.go",
			expected: "src/main.go",
		},
		{
			name:     "path with parent directory",
			userPath: "src/../lib/utils.go",
			expected: "lib/utils.go",
		},
		{
			name:     "directory path",
			userPath: "src/",
			expected: "src",
		},
		{
			name:        "absolute path outside repo",
			userPath:    "/tmp/file.go",
			expectError: true,
		},
		{
			name:        "path going outside repo",
			userPath:    "../../../outside.go",
			expectError: true,
		},
		{
			name:     "empty path",
			userPath: "",
			expected: ".",
		},
		{
			name:     "root path",
			userPath: ".",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeTimeseriesPath(repoPath, tt.userPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
