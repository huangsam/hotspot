package schema

import (
	"reflect"
	"testing"
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
			if got != tt.want {
				t.Fatalf("AbbreviateName(%q) = %q; want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestAbbreviateOwnersAndFormat(t *testing.T) {
	// Test that AbbreviateOwners correctly abbreviates a slice of owners,
	// including regular names, names with punctuation, and bot accounts.
	owners := []string{"Samuel Huang", "Ava (Billy) Cathy", "dependabot[bot]"}
	wantAbbrev := []string{"Samuel H", "Ava C", "dependabot[bot]"}

	got := AbbreviateOwners(owners)
	if !reflect.DeepEqual(got, wantAbbrev) {
		t.Fatalf("AbbreviateOwners = %v; want %v", got, wantAbbrev)
	}

	// Test that FormatOwners joins the abbreviated owners with commas.
	wantFormat := "Samuel H, Ava C, dependabot[bot]"
	gotFormat := FormatOwners(owners)
	if gotFormat != wantFormat {
		t.Fatalf("FormatOwners = %q; want %q", gotFormat, wantFormat)
	}
}

func TestOwnersEqual(t *testing.T) {
	// Test order-insensitive equality: same owners in different order should be equal.
	a := []string{"Alice", "Bob", "Carol"}
	b := []string{"Carol", "Alice", "Bob"}
	if !OwnersEqual(a, b) {
		t.Fatalf("OwnersEqual should treat order-insensitively but returned false for %v vs %v", a, b)
	}

	// Test that different lengths are not equal.
	c := []string{"Alice", "Bob"}
	if OwnersEqual(a, c) {
		t.Fatalf("OwnersEqual returned true for different-length slices %v vs %v", a, c)
	}
}
