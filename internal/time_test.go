package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2025, time.November, 3, 10, 0, 0, 0, time.UTC)

// TestParseRelativeTimeUnit covers various valid and invalid cases.
func TestParseRelativeTimeUnit(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Time
		expectError bool
	}{
		// Valid tests: Ensure units and casing are parsed correctly relative to fixedNow
		{
			name:        "valid plural months (mixed case)",
			input:       "3 MoNtHs AgO",
			expected:    fixedNow.AddDate(0, -3, 0), // 3 months before fixedNow
			expectError: false,
		},
		{
			name:        "valid singular week (capitalized)",
			input:       "1 Week Ago",
			expected:    fixedNow.Add(time.Duration(-1) * 7 * 24 * time.Hour), // 1 week before fixedNow
			expectError: false,
		},
		{
			name:        "valid 10 days (upper case)",
			input:       "10 DAYS AGO",
			expected:    fixedNow.Add(time.Duration(-10) * 24 * time.Hour), // 10 days before fixedNow
			expectError: false,
		},
		// Invalid tests: Ensure only supported formats/units are accepted
		{
			name:        "invalid missing ago",
			input:       "2 years",
			expectError: true,
		},
		{
			name:        "invalid bad unit (decades)",
			input:       "4 decades ago",
			expectError: true,
		},
		{
			name:        "invalid non-numeric value",
			input:       "one year ago",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tResult, err := parseRelativeTime(tt.input, fixedNow)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.Round(time.Second), tResult.Round(time.Second), "Parsed time mismatch")
			}
		})
	}
}

// TestParseLookbackDuration covers various valid and invalid lookback strings,
// including singular/plural forms and the month/year approximations.
func TestParseLookbackDuration(t *testing.T) {
	// Define expected durations based on the approximations used in the implementation:
	// 1 Month = 30 Days
	// 1 Year = 365 Days
	const day = 24 * time.Hour

	tests := []struct {
		name      string
		input     string
		want      time.Duration
		expectErr bool
	}{
		// --- Fixed Unit Tests (Exact duration) ---
		{"1 minute", "1 minute", time.Minute, false},
		{"5 minutes", "5 minutes", 5 * time.Minute, false},
		{"1 hour", "1 hour", time.Hour, false},
		{"3 hours", "3 hours", 3 * time.Hour, false},
		{"1 day", "1 day", day, false},
		{"7 days", "7 days", 7 * day, false},
		{"1 week", "1 week", 7 * day, false},
		{"4 weeks", "4 weeks", 4 * 7 * day, false},

		// --- Variable Unit Tests (Approximation) ---
		{"1 month approx", "1 month", 30 * day, false},
		{"6 months approx", "6 months", 6 * 30 * day, false},
		{"1 year approx", "1 year", 365 * day, false},
		{"2 years approx", "2 years", 2 * 365 * day, false},

		// --- Case/Spacing Tolerance Tests ---
		{"mixed case", "3 MoNtHs", 3 * 30 * day, false},
		{"extra space", " 1  day ", day, false},

		// --- Error/Invalid Tests ---
		{"invalid format (missing value)", "months", 0, true},
		{"invalid format (missing unit)", "3", 0, true},
		{"invalid unit", "3 decades", 0, true},
		// NOTE: Assuming 0 quantity will be caught by validation or cause an error in the implementation
		{"zero quantity", "0 days", 0, true},
		{"non-integer quantity", "1.5 days", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLookbackDuration(tt.input)

			if tt.expectErr {
				// Assert that an error occurred
				assert.Error(t, err, "Expected an error for input: %q", tt.input)
			} else if assert.NoError(t, err, "Did not expect an error for input: %q", tt.input) {
				// Assert that no error occurred, and then check the value
				assert.Equal(t, tt.want, got, "Duration mismatch for input: %q", tt.input)
			}
		})
	}
}
