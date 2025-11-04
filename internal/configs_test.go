package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSimpleInputs(t *testing.T) {
	t.Run("success minimal", func(t *testing.T) {
		cfg := &Config{}
		input := &ConfigRawInput{
			ResultLimit: 50,
			Workers:     4,
			Mode:        "hot",
			Precision:   1,
			Output:      "text",
			ExcludeStr:  "",
		}

		err := validateSimpleInputs(cfg, input)
		require.NoError(t, err, "validateSimpleInputs() failed unexpectedly: %v", err)
		assert.Equal(t, 50, cfg.ResultLimit, "ResultLimit was not set correctly, got %d, want 50", cfg.ResultLimit)
		assert.Equal(t, "hot", cfg.Mode, "Mode was not set correctly, got %s, want hot", cfg.Mode)
		assert.NotEmpty(t, cfg.Excludes, "Excludes list was unexpectedly empty")
	})

	t.Run("failure invalid mode", func(t *testing.T) {
		cfg := &Config{}
		input := &ConfigRawInput{
			ResultLimit: 50,
			Workers:     4,
			Mode:        "unknown_mode", // This is the error trigger
			Precision:   1,
			Output:      "text",
			ExcludeStr:  "",
		}

		err := validateSimpleInputs(cfg, input)
		require.Error(t, err, "validateSimpleInputs() expected an error for invalid mode, but got nil")
	})
}

func TestProcessTimeRange(t *testing.T) {
	tests := []struct {
		name        string
		input       *ConfigRawInput
		expectError bool
	}{
		// --- Existing Absolute Time Tests ---
		{
			name: "valid explicit range",
			input: &ConfigRawInput{
				StartTimeStr: "2024-01-01T00:00:00Z",
				EndTimeStr:   "2024-02-01T00:00:00Z",
			},
			expectError: false,
		},
		{
			name: "invalid start time format (absolute)",
			input: &ConfigRawInput{
				StartTimeStr: "01/01/2024", // Invalid format
				EndTimeStr:   "",
			},
			expectError: true,
		},
		{
			name: "invalid end time format (absolute)",
			input: &ConfigRawInput{
				StartTimeStr: "",
				EndTimeStr:   "01/01/2024", // Invalid format
			},
			expectError: true,
		},
		{
			name: "start time after end time (absolute)",
			input: &ConfigRawInput{
				StartTimeStr: "2024-02-01T00:00:00Z",
				EndTimeStr:   "2024-01-01T00:00:00Z",
			},
			expectError: true,
		},
		// --- Revised Relative Time Tests (Testing Case-Insensitivity) ---
		{
			name: "valid relative start time (plural, mixed case)",
			input: &ConfigRawInput{
				StartTimeStr: "3 MoNtHs AgO", // Testing mixed case input
				EndTimeStr:   "",
			},
			expectError: false,
		},
		{
			name: "valid relative start time (singular, capitalized)",
			input: &ConfigRawInput{
				StartTimeStr: "1 Week Ago", // Testing capitalized words input
				EndTimeStr:   "",
			},
			expectError: false,
		},
		{
			name: "valid relative start time (day, upper case)",
			input: &ConfigRawInput{
				StartTimeStr: "10 DAYS AGO", // Testing all caps input
				EndTimeStr:   "",
			},
			expectError: false,
		},
		{
			name: "valid relative start time (hour, mixed case)",
			input: &ConfigRawInput{
				StartTimeStr: "5 Hours ago",
				EndTimeStr:   "",
			},
			expectError: false,
		},
		{
			name: "valid relative start time (minute, mixed case)",
			input: &ConfigRawInput{
				StartTimeStr: "30 Minutes AgO",
				EndTimeStr:   "",
			},
			expectError: false,
		},
		// --- Failed/Validation Tests ---
		{
			name: "invalid relative time format (missing ago)",
			input: &ConfigRawInput{
				StartTimeStr: "2 years",
				EndTimeStr:   "",
			},
			expectError: true,
		},
		{
			name: "invalid relative time format (bad unit)",
			input: &ConfigRawInput{
				StartTimeStr: "4 decades ago",
				EndTimeStr:   "",
			},
			expectError: true,
		},
		{
			name: "invalid relative time format (non-numeric)",
			input: &ConfigRawInput{
				StartTimeStr: "one year ago",
				EndTimeStr:   "",
			},
			expectError: true,
		},
		{
			name: "relative start time after explicit end time",
			input: &ConfigRawInput{
				StartTimeStr: "1 minute ago",
				EndTimeStr:   "1990-01-01T00:00:00Z",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize cfg to a zero state (the function will set defaults internally if strings are empty)
			cfg := &Config{}
			err := processTimeRange(cfg, tt.input)

			if tt.expectError {
				require.Error(t, err, "processTimeRange() expected an error, but got nil")
			} else {
				require.NoError(t, err, "processTimeRange() unexpected error: %v", err)
			}
		})
	}
}
