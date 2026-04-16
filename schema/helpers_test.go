package schema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbbreviateName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// Basic cases
		{"popcorn", "popcorn"},            // single-part name
		{"Samuel Huang", "Samuel H"},      // standard two-part name
		{"First Second Third", "First T"}, // three substantial parts, uses last

		// Punctuation
		{"`backtickname", "backtickname"},    // name with backticks
		{"Ava (Billy) Cathy", "Ava C"},       // name with parentheses
		{"O'Neill John", "O'Neill J"},        // name with apostrophe
		{"Anne-Marie Smith", "Anne-Marie S"}, // name with hyphen
		{"Test-Name", "Test-Name"},           // hyphen in middle, single part

		// Spaces
		{"  Alice  ", "Alice"},   // leading/trailing spaces
		{"John   Doe", "John D"}, // multiple spaces

		// Initials and suffixes
		{"A B", "A B"},                      // two parts, uses last single letter
		{"X Y Z", "X Z"},                    // three parts, uses last single letter
		{"A B C D", "A D"},                  // four parts, uses last single letter
		{"A. B. C.", "A C"},                 // initials with periods, trimmed
		{"John D. Smith", "John S"},         // Initial as a middle component
		{"J. R. R. Tolkien", "J T"},         // Multiple initials
		{"Charles Darwin III", "Charles I"}, // Suffix as the last component
		{"Mr. Robert E. Lee", "Mr L"},       // Honorific and middle initial
		{"Dr. Mary J. Jane", "Dr J"},        // Honorific and middle initial (with period)

		// Symbols and special cases
		{"*Security-Bot*", "Security-Bot"},         // Leading/trailing symbols
		{"[John Smith]", "John S"},                 // Name fully wrapped in brackets
		{"C++-Bot", "C++-Bot"},                     // Single-part name with internal symbols
		{"123 Test", "123 T"},                      // starts with number
		{"user@example.com", "user@example.com"},   // E-mail as a name (single part)
		{"O'Malley-Ryan, Sean", "O'Malley-Ryan S"}, // Comma and hyphenated first name
		{"Ludwig van Beethoven", "Ludwig B"},       // Name with common prefix "van"
		{"Leonardo da Vinci", "Leonardo V"},        // Name with common prefix "da"

		// Bot accounts
		{"dependabot[bot]", "dependabot[bot]"},   // bot account, no abbreviation
		{"dependabot [bot]", "dependabot [bot]"}, // bot account with space, no abbreviation

		// Unicode
		{"张三", "张三"},                            // Chinese name, single part
		{"李 明", "李 明"},                          // Two-part Chinese name (First + Initial of Last character)
		{"राम कुमार", "राम क"},                  // Hindi name with Unicode
		{"Hans Müller", "Hans M"},               // German name with umlaut
		{"Jean-Pierre Dubois", "Jean-Pierre D"}, // French name with hyphen
		{"José María", "José M"},                // Spanish name with accent
		{"山田太郎", "山田太郎"},                        // Japanese name, single part
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AbbreviateName(tt.name)
			assert.Equal(t, tt.want, got, "AbbreviateName(%q) should match expected result", tt.name)
		})
	}
}

func TestAbbreviateOwnersAndFormat(t *testing.T) {
	// Test that AbbreviateOwners correctly abbreviates a slice of owners,
	// including regular names, names with punctuation, and bot accounts.
	owners := []string{"Samuel Huang", "Ava (Billy) Cathy", "dependabot[bot]"}
	wantAbbrev := []string{"Samuel H", "Ava C", "dependabot[bot]"}

	got := AbbreviateOwners(owners)
	assert.Equal(t, wantAbbrev, got, "AbbreviateOwners should abbreviate all owners correctly")

	// Test that FormatOwners joins the abbreviated owners with commas.
	wantFormat := "Samuel H, Ava C, dependabot[bot]"
	gotFormat := FormatOwners(owners)
	assert.Equal(t, wantFormat, gotFormat, "FormatOwners should join abbreviated owners with commas")
}

func TestOwnersEqual(t *testing.T) {
	// Test order-insensitive equality: same owners in different order should be equal.
	a := []string{"Alice", "Bob", "Carol"}
	b := []string{"Carol", "Alice", "Bob"}
	assert.True(t, OwnersEqual(a, b), "OwnersEqual should treat order-insensitively")

	// Test that different lengths are not equal.
	c := []string{"Alice", "Bob"}
	assert.False(t, OwnersEqual(a, c), "OwnersEqual should return false for different-length slices")
}

func TestGetDefaultWeights(t *testing.T) {
	tests := []struct {
		name string
		mode ScoringMode
	}{
		{"HotMode", HotMode},
		{"RiskMode", RiskMode},
		{"ComplexityMode", ComplexityMode},
		{"InvalidMode defaults to HotMode", ScoringMode("invalid")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weights := GetDefaultWeights(tt.mode)

			// Validate that all weights are non-negative
			for key, weight := range weights {
				assert.GreaterOrEqual(t, weight, 0.0, "weight for %s should be non-negative", key)
			}

			// Validate that we got some weights (not empty map)
			assert.NotEmpty(t, weights, "weights map should not be empty")
		})
	}
}

func TestFileResultGetters(t *testing.T) {
	file := FileResult{
		Path:      "src/main.go",
		ModeScore: 85.5,
		Commits:   Metric(42),
		Churn:     Metric(156),
		Owners:    []string{"Alice", "Bob"},
	}

	// Test GetPath
	assert.Equal(t, "src/main.go", file.GetPath(), "GetPath should return the file path")

	// Test GetScore
	assert.Equal(t, 85.5, file.GetScore(), "GetScore should return the computed score")

	// Test GetCommits
	assert.Equal(t, Metric(42), file.GetCommits(), "GetCommits should return the total commit count")

	// Test GetChurn
	assert.Equal(t, Metric(156), file.GetChurn(), "GetChurn should return the total churn")

	// Test GetOwners with populated owners
	assert.Equal(t, []string{"Alice", "Bob"}, file.GetOwners(), "GetOwners should return the top owners")

	// Test GetOwners with nil owners
	fileNil := FileResult{Owners: nil}
	assert.Equal(t, []string{}, fileNil.GetOwners(), "GetOwners should return empty slice for nil owners")
}

func TestFolderResultGetters(t *testing.T) {
	folder := FolderResult{
		Path:    "src/",
		Score:   72.3,
		Commits: Metric(128),
		Churn:   Metric(543),
		Owners:  []string{"Charlie", "Dana"},
	}

	// Test GetPath
	assert.Equal(t, "src/", folder.GetPath(), "GetPath should return the folder path")

	// Test GetScore
	assert.Equal(t, 72.3, folder.GetScore(), "GetScore should return the computed score")

	// Test GetCommits
	assert.Equal(t, Metric(128), folder.GetCommits(), "GetCommits should return the total commit count")

	// Test GetChurn
	assert.Equal(t, Metric(543), folder.GetChurn(), "GetChurn should return the total churn")

	// Test GetOwners with populated owners
	assert.Equal(t, []string{"Charlie", "Dana"}, folder.GetOwners(), "GetOwners should return the top owners")

	// Test GetOwners with nil owners
	folderNil := FolderResult{Owners: nil}
	assert.Equal(t, []string{}, folderNil.GetOwners(), "GetOwners should return empty slice for nil owners")
}

func TestGetPlainLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"smallest value possible", 0.0, LowValue},
		{"just before moderate", 39.9, LowValue},
		{"exactly moderate", 40.0, ModerateValue},
		{"just before high", 59.9, ModerateValue},
		{"exactly high", 60.0, HighValue},
		{"just before critical", 79.9, HighValue},
		{"exactly critical", 80.0, CriticalValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetPlainLabel(tt.input))
		})
	}
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

func TestParseBoolString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		hasError bool
	}{
		{"yes", "yes", true, false},
		{"YES", "YES", true, false},
		{"no", "no", false, false},
		{"NO", "NO", false, false},
		{"true", "true", true, false},
		{"TRUE", "TRUE", true, false},
		{"false", "false", false, false},
		{"FALSE", "FALSE", false, false},
		{"1", "1", true, false},
		{"0", "0", false, false},
		{"empty", "", false, true},
		{"invalid", "maybe", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseBoolString(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func FuzzShouldIgnore(f *testing.F) {
	seeds := []struct {
		path     string
		excludes string
	}{
		{"main.go", "*.log"},
		{"vendor/package/file.go", "vendor/"},
		{"test_file.min.js", "*.min.js"},
	}
	for _, seed := range seeds {
		f.Add(seed.path, seed.excludes)
	}

	f.Fuzz(func(_ *testing.T, path string, excludesStr string) {
		var excludes []string
		if excludesStr != "" {
			for ex := range strings.SplitSeq(excludesStr, ",") {
				if trimmed := strings.TrimSpace(ex); trimmed != "" {
					excludes = append(excludes, trimmed)
				}
			}
		}
		_ = ShouldIgnore(path, excludes)
	})
}
