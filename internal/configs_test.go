package internal

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSimpleInputs(t *testing.T) {
	t.Run("success minimal", func(t *testing.T) {
		cfg := &Config{}
		input := &ConfigRawInput{
			Limit:     50, // Changed from ResultLimit
			Workers:   4,
			Mode:      string(schema.HotMode),
			Precision: 1,
			Output:    "text",
			Exclude:   "", // Changed from ExcludeStr
		}

		err := validateSimpleInputs(cfg, input)
		require.NoError(t, err, "validateSimpleInputs() failed unexpectedly: %v", err)
		assert.Equal(t, 50, cfg.ResultLimit, "ResultLimit was not set correctly, got %d, want 50", cfg.ResultLimit)
		assert.Equal(t, schema.HotMode, cfg.Mode, "Mode was not set correctly, got %s, want hot", cfg.Mode)
		assert.NotEmpty(t, cfg.Excludes, "Excludes list was unexpectedly empty")
	})

	t.Run("failure invalid mode", func(t *testing.T) {
		cfg := &Config{}
		input := &ConfigRawInput{
			Limit:     50, // Changed from ResultLimit
			Workers:   4,
			Mode:      "unknown_mode", // This is the error trigger
			Precision: 1,
			Output:    "text",
			Exclude:   "", // Changed from ExcludeStr
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
		// --- Absolute Time Range Tests ---
		{
			name: "valid explicit range",
			input: &ConfigRawInput{
				Start: "2024-01-01T00:00:00Z", // Changed from StartTimeStr
				End:   "2024-02-01T00:00:00Z", // Changed from EndTimeStr
			},
			expectError: false,
		},
		{
			name: "invalid start time format (absolute)",
			input: &ConfigRawInput{
				Start: "01/01/2024", // Changed from StartTimeStr
				End:   "",           // Changed from EndTimeStr
			},
			expectError: true,
		},
		{
			name: "start time after end time (absolute)",
			input: &ConfigRawInput{
				Start: "2024-02-01T00:00:00Z", // Changed from StartTimeStr
				End:   "2024-01-01T00:00:00Z", // Changed from EndTimeStr
			},
			expectError: true,
		},
		// --- Relative Time Usage/Validation Tests (Focusing on flow, not grammar) ---
		{
			name: "valid relative start time (plural)",
			input: &ConfigRawInput{
				Start: "3 months ago", // Changed from StartTimeStr
				End:   "",             // Changed from EndTimeStr
			},
			expectError: false,
		},
		{
			name: "valid relative end time (explicit start)",
			input: &ConfigRawInput{
				Start: "2024-01-01T00:00:00Z", // Changed from StartTimeStr
				End:   "10 days ago",          // Changed from EndTimeStr
			},
			expectError: false,
		},
		{
			name: "invalid relative end time format (bad unit)",
			input: &ConfigRawInput{
				Start: "2024-01-01T00:00:00Z", // Changed from StartTimeStr
				End:   "2 badunit ago",        // Changed from EndTimeStr
			},
			// This test assumes your (un-provided) parseRelativeTime
			// will fail on "2 badunit ago"
			expectError: true,
		},
		// --- Critical Cross-Validation Tests ---
		{
			name: "relative start time after relative end time",
			input: &ConfigRawInput{
				Start: "1 minute ago", // Changed from StartTimeStr
				End:   "1 day ago",    // Changed from EndTimeStr
			},
			expectError: true,
		},
		{
			name: "relative start time after explicit end time",
			input: &ConfigRawInput{
				Start: "1 minute ago", // Changed from StartTimeStr
				End:   "1990-01-01T00:00:00Z",
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

func TestProcessCustomWeights(t *testing.T) {
	tests := []struct {
		name        string
		input       *ConfigRawInput
		expectError bool
		expected    map[schema.ScoringMode]map[schema.BreakdownKey]float64
	}{
		{
			name: "valid custom weights for hot mode",
			input: &ConfigRawInput{
				Weights: WeightsRawInput{
					Hot: &ModeWeightsRaw{
						Commits: &[]float64{0.5}[0],
						Churn:   &[]float64{0.3}[0],
						Age:     &[]float64{0.2}[0],
					},
				},
			},
			expectError: false,
			expected: map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				schema.HotMode: {
					schema.BreakdownCommits: 0.5,
					schema.BreakdownChurn:   0.3,
					schema.BreakdownAge:     0.2,
				},
			},
		},
		{
			name: "valid custom weights for all modes",
			input: &ConfigRawInput{
				Weights: WeightsRawInput{
					Hot: &ModeWeightsRaw{
						Commits:      &[]float64{0.4}[0],
						Churn:        &[]float64{0.4}[0],
						Age:          &[]float64{0.1}[0],
						Contributors: &[]float64{0.05}[0],
						Size:         &[]float64{0.05}[0],
					},
					Risk: &ModeWeightsRaw{
						InvContributors: &[]float64{0.3}[0],
						Gini:            &[]float64{0.26}[0],
						Age:             &[]float64{0.16}[0],
						Size:            &[]float64{0.12}[0],
						Churn:           &[]float64{0.06}[0],
						Commits:         &[]float64{0.04}[0],
						LOC:             &[]float64{0.06}[0],
					},
					Stale: &ModeWeightsRaw{
						InvRecent:    &[]float64{0.35}[0],
						Size:         &[]float64{0.25}[0],
						Age:          &[]float64{0.20}[0],
						Commits:      &[]float64{0.15}[0],
						Contributors: &[]float64{0.05}[0],
					},
					Complexity: &ModeWeightsRaw{
						Age:             &[]float64{0.30}[0],
						Churn:           &[]float64{0.30}[0],
						LOC:             &[]float64{0.20}[0],
						Commits:         &[]float64{0.10}[0],
						Size:            &[]float64{0.05}[0],
						InvContributors: &[]float64{0.05}[0],
					},
				},
			},
			expectError: false,
			expected: map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				schema.HotMode: {
					schema.BreakdownCommits: 0.4,
					schema.BreakdownChurn:   0.4,
					schema.BreakdownAge:     0.1,
					schema.BreakdownContrib: 0.05,
					schema.BreakdownSize:    0.05,
				},
				schema.RiskMode: {
					schema.BreakdownInvContrib: 0.3,
					schema.BreakdownGini:       0.26,
					schema.BreakdownAge:        0.16,
					schema.BreakdownSize:       0.12,
					schema.BreakdownChurn:      0.06,
					schema.BreakdownCommits:    0.04,
					schema.BreakdownLOC:        0.06,
				},
				schema.StaleMode: {
					schema.BreakdownInvRecent: 0.35,
					schema.BreakdownSize:      0.25,
					schema.BreakdownAge:       0.20,
					schema.BreakdownCommits:   0.15,
					schema.BreakdownContrib:   0.05,
				},
				schema.ComplexityMode: {
					schema.BreakdownAge:        0.30,
					schema.BreakdownChurn:      0.30,
					schema.BreakdownLOC:        0.20,
					schema.BreakdownCommits:    0.10,
					schema.BreakdownSize:       0.05,
					schema.BreakdownInvContrib: 0.05,
				},
			},
		},
		{
			name: "partial custom weights",
			input: &ConfigRawInput{
				Weights: WeightsRawInput{
					Hot: &ModeWeightsRaw{
						Commits: &[]float64{0.7}[0],
						Churn:   &[]float64{0.3}[0],
					},
				},
			},
			expectError: false,
			expected: map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				schema.HotMode: {
					schema.BreakdownCommits: 0.7,
					schema.BreakdownChurn:   0.3,
				},
			},
		},
		{
			name: "empty weights should not set anything",
			input: &ConfigRawInput{
				Weights: WeightsRawInput{},
			},
			expectError: false,
			expected:    map[schema.ScoringMode]map[schema.BreakdownKey]float64{},
		},
		{
			name: "weights that don't sum to 1.0 should fail",
			input: &ConfigRawInput{
				Weights: WeightsRawInput{
					Hot: &ModeWeightsRaw{
						Commits: &[]float64{0.5}[0],
						Churn:   &[]float64{0.3}[0],
						Age:     &[]float64{0.3}[0], // 0.5 + 0.3 + 0.3 = 1.1
					},
				},
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "weights that sum to less than 1.0 should fail",
			input: &ConfigRawInput{
				Weights: WeightsRawInput{
					Hot: &ModeWeightsRaw{
						Commits: &[]float64{0.3}[0],
						Churn:   &[]float64{0.3}[0],
					},
				},
			},
			expectError: true,
			expected:    nil,
		},
		{
			name: "negative weights should still be validated for sum",
			input: &ConfigRawInput{
				Weights: WeightsRawInput{
					Hot: &ModeWeightsRaw{
						Commits: &[]float64{0.5}[0],
						Churn:   &[]float64{-0.2}[0],
						Age:     &[]float64{0.7}[0], // 0.5 - 0.2 + 0.7 = 1.0
					},
				},
			},
			expectError: false,
			expected: map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				schema.HotMode: {
					schema.BreakdownCommits: 0.5,
					schema.BreakdownChurn:   -0.2,
					schema.BreakdownAge:     0.7,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			err := processCustomWeights(cfg, tt.input)

			if tt.expectError {
				require.Error(t, err, "processCustomWeights() expected an error, but got nil")
			} else {
				require.NoError(t, err, "processCustomWeights() unexpected error: %v", err)
				assert.Equal(t, tt.expected, cfg.CustomWeights, "CustomWeights mismatch")
			}
		})
	}
}
