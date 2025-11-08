package internal

import (
	"testing"
	"time"
)

// FuzzParseRelativeTime fuzzes the parseRelativeTime function with random inputs.
func FuzzParseRelativeTime(f *testing.F) {
	// Add some seed inputs
	seeds := []string{
		"1 year ago",
		"2 months ago",
		"3 weeks ago",
		"4 days ago",
		"5 hours ago",
		"6 minutes ago",
		"10 years ago",
		"0 years ago", // edge case
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		now := time.Now()
		_, err := parseRelativeTime(input, now)
		// We don't assert on the result, just that it doesn't panic
		_ = err // ignore error, we're testing for crashes
	})
}

// FuzzParseLookbackDuration fuzzes the parseLookbackDuration function.
func FuzzParseLookbackDuration(f *testing.F) {
	seeds := []string{
		"1 year",
		"2 months",
		"3 weeks",
		"4 days",
		"5 hours",
		"6 minutes",
		"10 years",
		"0 years", // edge case
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		_, err := parseLookbackDuration(input)
		_ = err
	})
}
