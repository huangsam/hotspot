package contract

import (
	"context"
	"path/filepath"
	"testing"

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
			},
			expectError: false,
			setupMock: func(mock *MockGitClient, workDir string) {
				ctx := context.Background()
				mock.On("GetRepoRoot", ctx, workDir).Return("/mock/repo/root", nil)
			},
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
