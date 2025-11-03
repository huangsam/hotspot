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
		{
			name: "Valid Explicit Range",
			input: &ConfigRawInput{
				StartTimeStr: "2024-01-01T00:00:00Z",
				EndTimeStr:   "2024-02-01T00:00:00Z",
			},
			expectError: false,
		},
		{
			name: "Invalid Start Time Format",
			input: &ConfigRawInput{
				StartTimeStr: "01/01/2024", // Invalid format
				EndTimeStr:   "",
			},
			expectError: true,
		},
		{
			name: "Invalid End Time Format",
			input: &ConfigRawInput{
				StartTimeStr: "",
				EndTimeStr:   "01/01/2024", // Invalid format
			},
			expectError: true,
		},
		{
			name: "Start Time After End Time",
			input: &ConfigRawInput{
				StartTimeStr: "2024-02-01T00:00:00Z",
				EndTimeStr:   "2024-01-01T00:00:00Z",
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
