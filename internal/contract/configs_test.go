package contract

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Assuming ConfigRawInput is an alias for contract.ConfigRawInput for brevity in the test file.
// If not, the test cases need to be adjusted based on the actual struct definition.
// For now, I'll treat `&ConfigRawInput{...}` as the constructor for the struct being tested.

func TestProcessAndValidate(t *testing.T) {
	tests := []struct {
		name        string
		input       *ConfigRawInput
		expectError bool
		setupMock   func(*MockGitClient, string) // Pass the expected working directory
	}{
		{
			name: "valid minimal config",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid mode",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        "invalid_mode",
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil, // No mock setup needed since validation fails early
		},
		{
			name: "compare mode with both refs",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				BaseRef:     "main",
				TargetRef:   "feature-branch",
				Lookback:    "30 days",
				Color:       "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "compare mode missing base ref",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				TargetRef:   "feature-branch",
				Lookback:    "30 days",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil, // No mock setup needed since validation fails early
		},
		{
			name: "timeseries mode",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Path:        "src/main.go",
				Interval:    "180 days",
				Points:      4,
				Color:       "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid limit (zero)",
			input: &ConfigRawInput{
				Limit:       0,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid limit (negative)",
			input: &ConfigRawInput{
				Limit:       -1,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid limit (too large)",
			input: &ConfigRawInput{
				Limit:       1001,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid workers (zero)",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     0,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid workers (negative)",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     -1,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid precision (zero)",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   0,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid precision (too high)",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   3,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid output format",
			input: &ConfigRawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "invalid_format",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "invalid cache backend",
			input: &ConfigRawInput{
				Limit:        10,
				Workers:      4,
				Mode:         string(schema.HotMode),
				Precision:    1,
				Output:       "text",
				Exclude:      "",
				RepoPathStr:  ".",
				CacheBackend: "invalid_backend",
				Color:        "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "mysql backend without connection string",
			input: &ConfigRawInput{
				Limit:        10,
				Workers:      4,
				Mode:         string(schema.HotMode),
				Precision:    1,
				Output:       "text",
				Exclude:      "",
				RepoPathStr:  ".",
				CacheBackend: string(schema.MySQLBackend),
				Color:        "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "postgresql backend without connection string",
			input: &ConfigRawInput{
				Limit:        10,
				Workers:      4,
				Mode:         string(schema.HotMode),
				Precision:    1,
				Output:       "text",
				Exclude:      "",
				RepoPathStr:  ".",
				CacheBackend: string(schema.PostgreSQLBackend),
				Color:        "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "mysql backend with connection string",
			input: &ConfigRawInput{
				Limit:          10,
				Workers:        4,
				Mode:           string(schema.HotMode),
				Precision:      1,
				Output:         "text",
				Exclude:        "",
				RepoPathStr:    ".",
				CacheBackend:   string(schema.MySQLBackend),
				CacheDBConnect: "user:pass@tcp(localhost:3306)/hotspot",
				Color:          "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "none backend",
			input: &ConfigRawInput{
				Limit:        10,
				Workers:      4,
				Mode:         string(schema.HotMode),
				Precision:    1,
				Output:       "text",
				Exclude:      "",
				RepoPathStr:  ".",
				CacheBackend: string(schema.NoneBackend),
				Color:        "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend sqlite",
			input: &ConfigRawInput{
				Limit:             10,
				Workers:           4,
				Mode:              string(schema.HotMode),
				Precision:         1,
				Output:            "text",
				Exclude:           "",
				RepoPathStr:       ".",
				CacheBackend:      string(schema.SQLiteBackend),
				AnalysisBackend:   string(schema.SQLiteBackend),
				AnalysisDBConnect: "analysis.db",
				Color:             "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend mysql with connection string",
			input: &ConfigRawInput{
				Limit:             10,
				Workers:           4,
				Mode:              string(schema.HotMode),
				Precision:         1,
				Output:            "text",
				Exclude:           "",
				RepoPathStr:       ".",
				CacheBackend:      string(schema.MySQLBackend),
				CacheDBConnect:    "user:pass@tcp(localhost:3306)/cache",
				AnalysisBackend:   string(schema.MySQLBackend),
				AnalysisDBConnect: "user:pass@tcp(localhost:3306)/analysis",
				Color:             "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend same as cache backend with different db",
			input: &ConfigRawInput{
				Limit:             10,
				Workers:           4,
				Mode:              string(schema.HotMode),
				Precision:         1,
				Output:            "text",
				Exclude:           "",
				RepoPathStr:       ".",
				CacheBackend:      string(schema.MySQLBackend),
				CacheDBConnect:    "user:pass@tcp(localhost:3306)/cache",
				AnalysisBackend:   string(schema.MySQLBackend),
				AnalysisDBConnect: "user:pass@tcp(localhost:3306)/analysis",
				Color:             "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend same as cache backend with same db should not fail for mysql",
			input: &ConfigRawInput{
				Limit:             10,
				Workers:           4,
				Mode:              string(schema.HotMode),
				Precision:         1,
				Output:            "text",
				Exclude:           "",
				RepoPathStr:       ".",
				CacheBackend:      string(schema.MySQLBackend),
				CacheDBConnect:    "user:pass@tcp(localhost:3306)/hotspot",
				AnalysisBackend:   string(schema.MySQLBackend),
				AnalysisDBConnect: "user:pass@tcp(localhost:3306)/hotspot",
				Color:             "yes",
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid analysis backend",
			input: &ConfigRawInput{
				Limit:           10,
				Workers:         4,
				Mode:            string(schema.HotMode),
				Precision:       1,
				Output:          "text",
				Exclude:         "",
				RepoPathStr:     ".",
				CacheBackend:    string(schema.SQLiteBackend),
				AnalysisBackend: "invalid_backend",
				Color:           "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
		{
			name: "both sqlite with same explicit database path should fail",
			input: &ConfigRawInput{
				Limit:             10,
				Workers:           4,
				Mode:              string(schema.HotMode),
				Precision:         1,
				Output:            "text",
				Exclude:           "",
				RepoPathStr:       ".",
				CacheBackend:      string(schema.SQLiteBackend),
				CacheDBConnect:    "/tmp/same.db",
				AnalysisBackend:   string(schema.SQLiteBackend),
				AnalysisDBConnect: "/tmp/same.db",
				Color:             "yes",
			},
			expectError: true,
			setupMock:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockGitClient)

			// Dynamically determine the expected working directory
			workDir, err := filepath.Abs(".")
			require.NoError(t, err)

			if tt.setupMock != nil {
				tt.setupMock(mockClient, workDir)
			}

			// Set default cache backend if not specified
			if tt.input.CacheBackend == "" {
				tt.input.CacheBackend = string(schema.SQLiteBackend)
			}

			cfg := &Config{}
			ctx := context.Background()
			err = ProcessAndValidate(ctx, cfg, mockClient, tt.input)

			if tt.expectError {
				assert.Error(t, err, "contract.ProcessAndValidate should return an error for %s", tt.name)
			} else {
				assert.NoError(t, err, "contract.ProcessAndValidate should not return an error for %s", tt.name)
				// Basic validation that config was populated
				assert.Equal(t, tt.input.Limit, cfg.ResultLimit)
				assert.Equal(t, schema.ScoringMode(tt.input.Mode), cfg.Mode)
			}

			if tt.setupMock != nil {
				mockClient.AssertExpectations(t)
			}
		})
	}
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

func TestConfigClone(t *testing.T) {
	original := &Config{
		ResultLimit: 10,
		Workers:     4,
		Mode:        schema.HotMode,
		Precision:   1,
		Output:      schema.TextOut,
		Excludes:    []string{"*.tmp", "*.log"},
		CustomWeights: map[schema.ScoringMode]map[schema.BreakdownKey]float64{
			schema.HotMode: {
				schema.BreakdownCommits: 0.5,
				schema.BreakdownChurn:   0.5,
			},
		},
		PathFilter:         "src/",
		OutputFile:         "output.txt",
		Detail:             true,
		Explain:            true,
		Owner:              true,
		Follow:             true,
		Width:              120,
		StartTime:          time.Now().Add(-24 * time.Hour),
		EndTime:            time.Now(),
		BaseRef:            "main",
		TargetRef:          "feature",
		CompareMode:        true,
		Lookback:           30 * 24 * time.Hour,
		TimeseriesPath:     "src/main.go",
		TimeseriesPoints:   4,
		TimeseriesInterval: 7 * 24 * time.Hour,
		RepoPath:           "/path/to/repo",
	}

	clone := original.Clone()

	// Test that clone is equal but not the same reference
	assert.Equal(t, original, clone)
	assert.NotSame(t, original, clone)

	// Test that slices are deep copied
	assert.NotSame(t, &original.Excludes, &clone.Excludes)
	assert.Equal(t, original.Excludes, clone.Excludes)

	// Test that maps are deep copied
	assert.NotSame(t, &original.CustomWeights, &clone.CustomWeights)
	assert.Equal(t, original.CustomWeights, clone.CustomWeights)

	// Modify original and ensure clone is unaffected
	original.Excludes[0] = "modified.tmp"
	original.CustomWeights[schema.HotMode][schema.BreakdownCommits] = 0.7

	assert.NotEqual(t, original.Excludes[0], clone.Excludes[0])
	assert.NotEqual(t, original.CustomWeights[schema.HotMode][schema.BreakdownCommits], clone.CustomWeights[schema.HotMode][schema.BreakdownCommits])
}

func TestConfigCloneWithTimeWindow(t *testing.T) {
	original := &Config{
		ResultLimit: 10,
		Mode:        schema.HotMode,
		StartTime:   time.Now().Add(-7 * 24 * time.Hour),
		EndTime:     time.Now(),
	}

	newStart := time.Now().Add(-24 * time.Hour)
	newEnd := time.Now()

	clone := original.CloneWithTimeWindow(newStart, newEnd)

	// Test that other fields are preserved
	assert.Equal(t, original.ResultLimit, clone.ResultLimit)
	assert.Equal(t, original.Mode, clone.Mode)

	// Test that times are updated
	assert.WithinDuration(t, newStart, clone.StartTime, time.Millisecond)
	assert.WithinDuration(t, newEnd, clone.EndTime, time.Millisecond)
}

func TestProcessProfilingConfig(t *testing.T) {
	tests := []struct {
		name            string
		profilePrefix   string
		expectedEnabled bool
		expectedPrefix  string
	}{
		{
			name:            "empty prefix disables profiling",
			profilePrefix:   "",
			expectedEnabled: false,
			expectedPrefix:  "",
		},
		{
			name:            "non-empty prefix enables profiling",
			profilePrefix:   "cpu",
			expectedEnabled: true,
			expectedPrefix:  "cpu",
		},
		{
			name:            "prefix with path",
			profilePrefix:   "/tmp/profile",
			expectedEnabled: true,
			expectedPrefix:  "/tmp/profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &ProfileConfig{}
			err := ProcessProfilingConfig(profile, tt.profilePrefix)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEnabled, profile.Enabled)
			assert.Equal(t, tt.expectedPrefix, profile.Prefix)
		})
	}
}

// TestGetAnalysisStartAndEndTime tests the canonical time truncation methods.
func TestGetAnalysisStartAndEndTime(t *testing.T) {
	// Create a time with minutes, seconds, and nanoseconds
	now := time.Date(2024, 1, 15, 14, 30, 45, 123456789, time.UTC)
	startTime := now.AddDate(0, 0, -365)

	cfg := &Config{
		StartTime: startTime,
		EndTime:   now,
	}

	// Test that GetAnalysisStartTime truncates to the hour
	truncatedStart := cfg.GetAnalysisStartTime()
	expectedStart := startTime.Truncate(CacheGranularity)

	assert.Equal(t, expectedStart, truncatedStart, "GetAnalysisStartTime should truncate to hour")
	assert.Equal(t, 0, truncatedStart.Minute(), "Minutes should be 0")
	assert.Equal(t, 0, truncatedStart.Second(), "Seconds should be 0")
	assert.Equal(t, 0, truncatedStart.Nanosecond(), "Nanoseconds should be 0")

	// Test that GetAnalysisEndTime truncates to the hour
	truncatedEnd := cfg.GetAnalysisEndTime()
	expectedEnd := now.Truncate(CacheGranularity)

	assert.Equal(t, expectedEnd, truncatedEnd, "GetAnalysisEndTime should truncate to hour")
	assert.Equal(t, 0, truncatedEnd.Minute(), "Minutes should be 0")
	assert.Equal(t, 0, truncatedEnd.Second(), "Seconds should be 0")
	assert.Equal(t, 0, truncatedEnd.Nanosecond(), "Nanoseconds should be 0")

	// Test that the granularity constant is indeed time.Hour
	assert.Equal(t, time.Hour, CacheGranularity, "CacheGranularity should be time.Hour")
}

func TestValidateDatabaseConnectionString(t *testing.T) {
	tests := []struct {
		name        string
		backend     schema.DatabaseBackend
		connStr     string
		expectError bool
	}{
		// MySQL tests
		{
			name:        "mysql empty string",
			backend:     schema.MySQLBackend,
			connStr:     "",
			expectError: true,
		},
		{
			name:        "mysql valid connection string",
			backend:     schema.MySQLBackend,
			connStr:     "user:pass@tcp(localhost:3306)/hotspot",
			expectError: false,
		},
		{
			name:        "mysql missing @tcp(",
			backend:     schema.MySQLBackend,
			connStr:     "user:pass@localhost:3306/hotspot",
			expectError: true,
		},
		{
			name:        "mysql missing database",
			backend:     schema.MySQLBackend,
			connStr:     "user:pass@tcp(localhost:3306)",
			expectError: true,
		},
		{
			name:        "mysql missing both @tcp( and database",
			backend:     schema.MySQLBackend,
			connStr:     "user:pass@localhost:3306",
			expectError: true,
		},

		// PostgreSQL tests
		{
			name:        "postgresql empty string",
			backend:     schema.PostgreSQLBackend,
			connStr:     "",
			expectError: true,
		},
		{
			name:        "postgresql valid connection string",
			backend:     schema.PostgreSQLBackend,
			connStr:     "host=localhost port=5432 user=postgres password=secret dbname=hotspot",
			expectError: false,
		},
		{
			name:        "postgresql missing host=",
			backend:     schema.PostgreSQLBackend,
			connStr:     "port=5432 user=postgres password=secret dbname=hotspot",
			expectError: true,
		},
		{
			name:        "postgresql missing dbname=",
			backend:     schema.PostgreSQLBackend,
			connStr:     "host=localhost port=5432 user=postgres password=secret",
			expectError: true,
		},
		{
			name:        "postgresql missing both host= and dbname=",
			backend:     schema.PostgreSQLBackend,
			connStr:     "port=5432 user=postgres password=secret",
			expectError: true,
		},
		{
			name:        "postgresql valid with minimal params",
			backend:     schema.PostgreSQLBackend,
			connStr:     "host=localhost dbname=hotspot",
			expectError: false,
		},

		// Other backends
		{
			name:        "sqlite backend with empty string",
			backend:     schema.SQLiteBackend,
			connStr:     "",
			expectError: false,
		},
		{
			name:        "sqlite backend with non-empty string",
			backend:     schema.SQLiteBackend,
			connStr:     "some.db",
			expectError: false,
		},
		{
			name:        "none backend with empty string",
			backend:     schema.NoneBackend,
			connStr:     "",
			expectError: false,
		},
		{
			name:        "none backend with non-empty string",
			backend:     schema.NoneBackend,
			connStr:     "anything",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDatabaseConnectionString(tt.backend, tt.connStr)

			if tt.expectError {
				assert.Error(t, err, "ValidateDatabaseConnectionString should return an error for %s", tt.name)
			} else {
				assert.NoError(t, err, "ValidateDatabaseConnectionString should not return an error for %s", tt.name)
			}
		})
	}
}

func TestRevalidateCompare(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		lookbackStr string
		expectError bool
	}{
		{
			name:        "valid lookback and base ref",
			cfg:         &Config{BaseRef: "main"},
			lookbackStr: "30 days",
			expectError: false,
		},
		{
			name:        "empty lookback but valid base ref",
			cfg:         &Config{BaseRef: "main"},
			lookbackStr: "",
			expectError: false,
		},
		{
			name:        "invalid lookback string",
			cfg:         &Config{BaseRef: "main"},
			lookbackStr: "invalid_duration",
			expectError: true,
		},
		{
			name:        "missing base ref",
			cfg:         &Config{BaseRef: ""},
			lookbackStr: "30 days",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RevalidateCompare(tt.cfg, tt.lookbackStr)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRevalidateTimeseries(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		intervalStr string
		expectError bool
	}{
		{
			name:        "valid interval and valid points",
			cfg:         &Config{TimeseriesPoints: 5},
			intervalStr: "1 week",
			expectError: false,
		},
		{
			name:        "empty interval",
			cfg:         &Config{TimeseriesPoints: 5},
			intervalStr: "",
			expectError: false,
		},
		{
			name:        "invalid interval string",
			cfg:         &Config{TimeseriesPoints: 5},
			intervalStr: "invalid_duration",
			expectError: true,
		},
		{
			name:        "invalid points (<1)",
			cfg:         &Config{TimeseriesPoints: -1},
			intervalStr: "1 week",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RevalidateTimeseries(tt.cfg, tt.intervalStr)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
