package config

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/git"
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
		input       *RawInput
		expectError bool
		setupMock   func(*git.MockGitClient, string) // Pass the expected working directory
	}{
		{
			name: "valid minimal config",
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid mode",
			input: &RawInput{
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
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "compare mode missing base ref",
			input: &RawInput{
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
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "default limit (zero)",
			input: &RawInput{
				Limit:       0,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: false,
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid limit (negative)",
			input: &RawInput{
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
			input: &RawInput{
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
			name: "default workers (zero)",
			input: &RawInput{
				Limit:       10,
				Workers:     0,
				Mode:        string(schema.HotMode),
				Precision:   1,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: false,
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid workers (negative)",
			input: &RawInput{
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
			name: "default precision (zero)",
			input: &RawInput{
				Limit:       10,
				Workers:     4,
				Mode:        string(schema.HotMode),
				Precision:   0,
				Output:      "text",
				Exclude:     "",
				RepoPathStr: ".",
				Color:       "yes",
			},
			expectError: false,
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid precision (too high)",
			input: &RawInput{
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
			input: &RawInput{
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
			input: &RawInput{
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
			input: &RawInput{
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
			input: &RawInput{
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
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "none backend",
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend sqlite",
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend mysql with connection string",
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend same as cache backend with different db",
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "analysis backend same as cache backend with same db should not fail for mysql",
			input: &RawInput{
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
			setupMock: func(mock *git.MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
		},
		{
			name: "invalid analysis backend",
			input: &RawInput{
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
			input: &RawInput{
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
			mockClient := new(git.MockGitClient)

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
				if tt.input.Limit > 0 {
					assert.Equal(t, tt.input.Limit, cfg.Output.ResultLimit)
				} else {
					assert.NotZero(t, cfg.Output.ResultLimit)
				}

				if tt.input.Mode != "" {
					assert.Equal(t, schema.ScoringMode(tt.input.Mode), cfg.Scoring.Mode)
				} else {
					assert.Equal(t, schema.HotMode, cfg.Scoring.Mode)
				}
			}

			if tt.setupMock != nil {
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestExcludes(t *testing.T) {
	tests := []struct {
		name     string
		input    *RawInput
		expected []string
	}{
		{
			name: "defaults when no exclude provided",
			input: &RawInput{
				Exclude: "",
			},
			expected: func() []string {
				var expected []string
				for p := range strings.SplitSeq(schema.DefaultExclude, ",") {
					trimmedP := strings.TrimSpace(p)
					if trimmedP != "" {
						expected = append(expected, trimmedP)
					}
				}
				return expected
			}(),
		},
		{
			name: "additive with custom exclude provided",
			input: &RawInput{
				Exclude: "my_custom_dir/, another_dir/",
			},
			expected: func() []string {
				var expected []string
				for p := range strings.SplitSeq(schema.DefaultExclude, ",") {
					trimmedP := strings.TrimSpace(p)
					if trimmedP != "" {
						expected = append(expected, trimmedP)
					}
				}
				expected = append(expected, "my_custom_dir/", "another_dir/")
				return expected
			}(),
		},
		{
			name: "additive with single custom item",
			input: &RawInput{
				Exclude: "*.special_log",
			},
			expected: func() []string {
				var expected []string
				for p := range strings.SplitSeq(schema.DefaultExclude, ",") {
					trimmedP := strings.TrimSpace(p)
					if trimmedP != "" {
						expected = append(expected, trimmedP)
					}
				}
				expected = append(expected, "*.special_log")
				return expected
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			err := validateSimpleInputs(cfg, tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Git.Excludes)
		})
	}
}

func TestProcessTimeRange(t *testing.T) {
	tests := []struct {
		name        string
		input       *RawInput
		expectError bool
	}{
		// --- Absolute Time Range Tests ---
		{
			name: "valid explicit range",
			input: &RawInput{
				Start: "2024-01-01T00:00:00Z", // Changed from StartTimeStr
				End:   "2024-02-01T00:00:00Z", // Changed from EndTimeStr
			},
			expectError: false,
		},
		{
			name: "invalid start time format (absolute)",
			input: &RawInput{
				Start: "01/01/2024", // Changed from StartTimeStr
				End:   "",           // Changed from EndTimeStr
			},
			expectError: true,
		},
		{
			name: "start time after end time (absolute)",
			input: &RawInput{
				Start: "2024-02-01T00:00:00Z", // Changed from StartTimeStr
				End:   "2024-01-01T00:00:00Z", // Changed from EndTimeStr
			},
			expectError: true,
		},
		// --- Relative Time Usage/Validation Tests (Focusing on flow, not grammar) ---
		{
			name: "valid relative start time (plural)",
			input: &RawInput{
				Start: "3 months ago", // Changed from StartTimeStr
				End:   "",             // Changed from EndTimeStr
			},
			expectError: false,
		},
		{
			name: "valid relative end time (explicit start)",
			input: &RawInput{
				Start: "2024-01-01T00:00:00Z", // Changed from StartTimeStr
				End:   "10 days ago",          // Changed from EndTimeStr
			},
			expectError: false,
		},
		{
			name: "invalid relative end time format (bad unit)",
			input: &RawInput{
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
			input: &RawInput{
				Start: "1 minute ago", // Changed from StartTimeStr
				End:   "1 day ago",    // Changed from EndTimeStr
			},
			expectError: true,
		},
		{
			name: "relative start time after explicit end time",
			input: &RawInput{
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
		input       *RawInput
		expectError bool
		expected    map[schema.ScoringMode]map[schema.BreakdownKey]float64
	}{
		{
			name: "valid custom weights for hot mode",
			input: &RawInput{
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
			input: &RawInput{
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
						LowRecent:       &[]float64{0.04}[0],
						LOC:             &[]float64{0.06}[0],
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
					schema.BreakdownLowRecent:  0.04,
					schema.BreakdownLOC:        0.06,
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
			input: &RawInput{
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
			input: &RawInput{
				Weights: WeightsRawInput{},
			},
			expectError: false,
			expected:    map[schema.ScoringMode]map[schema.BreakdownKey]float64{},
		},
		{
			name: "weights that don't sum to 1.0 should fail",
			input: &RawInput{
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
			input: &RawInput{
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
			input: &RawInput{
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
				assert.Equal(t, tt.expected, cfg.Scoring.CustomWeights, "CustomWeights mismatch")
			}
		})
	}
}

func TestConfigClone(t *testing.T) {
	original := &Config{
		Output: OutputConfig{
			ResultLimit: 10,
			Precision:   1,
			Format:      schema.TextOut,
			OutputFile:  "output.txt",
			Detail:      true,
			Explain:     true,
			Owner:       true,
			Width:       120,
		},
		Runtime: RuntimeConfig{
			Workers: 4,
		},
		Scoring: ScoringConfig{
			Mode: schema.HotMode,
			CustomWeights: map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				schema.HotMode: {
					schema.BreakdownCommits: 0.5,
					schema.BreakdownChurn:   0.5,
				},
			},
		},
		Git: GitConfig{
			Excludes:   []string{"*.tmp", "*.log"},
			PathFilter: "src/",
			StartTime:  time.Now().Add(-24 * time.Hour),
			EndTime:    time.Now(),
			Follow:     true,
			RepoPath:   "/path/to/repo",
		},
		Compare: CompareConfig{
			BaseRef:   "main",
			TargetRef: "feature",
			Enabled:   true,
			Lookback:  30 * 24 * time.Hour,
		},
		Timeseries: TimeseriesConfig{
			Path:     "src/main.go",
			Points:   4,
			Interval: 7 * 24 * time.Hour,
		},
	}

	clone := original.Clone()

	// Test that clone is equal but not the same reference
	assert.Equal(t, original, clone)
	assert.NotSame(t, original, clone)

	// Test that slices are deep copied
	assert.NotSame(t, &original.Git.Excludes, &clone.Git.Excludes)
	assert.Equal(t, original.Git.Excludes, clone.Git.Excludes)

	// Test that maps are deep copied
	assert.NotSame(t, &original.Scoring.CustomWeights, &clone.Scoring.CustomWeights)
	assert.Equal(t, original.Scoring.CustomWeights, clone.Scoring.CustomWeights)

	// Modify original and ensure clone is unaffected
	original.Git.Excludes[0] = "modified.tmp"
	original.Scoring.CustomWeights[schema.HotMode][schema.BreakdownCommits] = 0.7

	assert.NotEqual(t, original.Git.Excludes[0], clone.Git.Excludes[0])
	assert.NotEqual(t, original.Scoring.CustomWeights[schema.HotMode][schema.BreakdownCommits], clone.Scoring.CustomWeights[schema.HotMode][schema.BreakdownCommits])
}

func TestConfigCloneWithTimeWindow(t *testing.T) {
	original := &Config{
		Output: OutputConfig{
			ResultLimit: 10,
		},
		Scoring: ScoringConfig{
			Mode: schema.HotMode,
		},
		Git: GitConfig{
			StartTime: time.Now().Add(-7 * 24 * time.Hour),
			EndTime:   time.Now(),
		},
	}

	newStart := time.Now().Add(-24 * time.Hour)
	newEnd := time.Now()

	clone := original.CloneWithTimeWindow(newStart, newEnd)

	// Test that other fields are preserved
	assert.Equal(t, original.Output.ResultLimit, clone.Output.ResultLimit)
	assert.Equal(t, original.Scoring.Mode, clone.Scoring.Mode)

	// Test that times are updated
	assert.WithinDuration(t, newStart, clone.Git.StartTime, time.Millisecond)
	assert.WithinDuration(t, newEnd, clone.Git.EndTime, time.Millisecond)
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
		Git: GitConfig{
			StartTime: startTime,
			EndTime:   now,
		},
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
			cfg:         &Config{Compare: CompareConfig{BaseRef: "main"}},
			lookbackStr: "30 days",
			expectError: false,
		},
		{
			name:        "empty lookback but valid base ref",
			cfg:         &Config{Compare: CompareConfig{BaseRef: "main"}},
			lookbackStr: "",
			expectError: false,
		},
		{
			name:        "invalid lookback string",
			cfg:         &Config{Compare: CompareConfig{BaseRef: "main"}},
			lookbackStr: "invalid_duration",
			expectError: true,
		},
		{
			name:        "missing base ref",
			cfg:         &Config{Compare: CompareConfig{BaseRef: ""}},
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
			cfg:         &Config{Timeseries: TimeseriesConfig{Points: 5}},
			intervalStr: "1 week",
			expectError: false,
		},
		{
			name:        "empty interval",
			cfg:         &Config{Timeseries: TimeseriesConfig{Points: 5}},
			intervalStr: "",
			expectError: false,
		},
		{
			name:        "invalid interval string",
			cfg:         &Config{Timeseries: TimeseriesConfig{Points: 5}},
			intervalStr: "invalid_duration",
			expectError: true,
		},
		{
			name:        "invalid points (<1)",
			cfg:         &Config{Timeseries: TimeseriesConfig{Points: -1}},
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
