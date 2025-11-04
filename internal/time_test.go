package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2025, time.November, 3, 10, 0, 0, 0, time.UTC)

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
