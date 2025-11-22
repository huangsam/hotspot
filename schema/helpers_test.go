package schema

import (
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
		{"StaleMode", StaleMode},
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
		Commits:   42,
		Churn:     156,
		Owners:    []string{"Alice", "Bob"},
	}

	// Test GetPath
	assert.Equal(t, "src/main.go", file.GetPath(), "GetPath should return the file path")

	// Test GetScore
	assert.Equal(t, 85.5, file.GetScore(), "GetScore should return the computed score")

	// Test GetCommits
	assert.Equal(t, 42, file.GetCommits(), "GetCommits should return the total commit count")

	// Test GetChurn
	assert.Equal(t, 156, file.GetChurn(), "GetChurn should return the total churn")

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
		Commits: 128,
		Churn:   543,
		Owners:  []string{"Charlie", "Dana"},
	}

	// Test GetPath
	assert.Equal(t, "src/", folder.GetPath(), "GetPath should return the folder path")

	// Test GetScore
	assert.Equal(t, 72.3, folder.GetScore(), "GetScore should return the computed score")

	// Test GetCommits
	assert.Equal(t, 128, folder.GetCommits(), "GetCommits should return the total commit count")

	// Test GetChurn
	assert.Equal(t, 543, folder.GetChurn(), "GetChurn should return the total churn")

	// Test GetOwners with populated owners
	assert.Equal(t, []string{"Charlie", "Dana"}, folder.GetOwners(), "GetOwners should return the top owners")

	// Test GetOwners with nil owners
	folderNil := FolderResult{Owners: nil}
	assert.Equal(t, []string{}, folderNil.GetOwners(), "GetOwners should return empty slice for nil owners")
}
