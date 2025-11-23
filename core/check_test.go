package core

import (
	"context"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestFilterChangedFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		excludes []string
		expected []string
	}{
		{
			name:     "no excludes",
			files:    []string{"main.go", "core/check.go", "README.md"},
			excludes: []string{},
			expected: []string{"main.go", "core/check.go", "README.md"},
		},
		{
			name:     "exclude by extension",
			files:    []string{"main.go", "core/check.go", "README.md"},
			excludes: []string{".md"},
			expected: []string{"main.go", "core/check.go"},
		},
		{
			name:     "exclude by directory",
			files:    []string{"main.go", "vendor/lib.go", "dist/app.js"},
			excludes: []string{"vendor/", "dist/"},
			expected: []string{"main.go"},
		},
		{
			name:     "all files excluded",
			files:    []string{"README.md", "LICENSE"},
			excludes: []string{".md", "LICENSE"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterChangedFiles(tt.files, tt.excludes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrintCheckResult(t *testing.T) {
	// Test that printCheckResult doesn't panic with various inputs
	tests := []struct {
		name   string
		result schema.CheckResult
	}{
		{
			name: "all passed",
			result: schema.CheckResult{
				Passed:       true,
				FailedFiles:  []schema.CheckFailedFile{},
				TotalFiles:   5,
				CheckedModes: []schema.ScoringMode{schema.HotMode, schema.RiskMode},
				BaseRef:      "main",
				TargetRef:    "HEAD",
				Thresholds: map[schema.ScoringMode]float64{
					schema.HotMode:        50.0,
					schema.RiskMode:       50.0,
					schema.ComplexityMode: 50.0,
					schema.StaleMode:      50.0,
				},
				MaxScores: map[schema.ScoringMode]float64{
					schema.HotMode:        30.0,
					schema.RiskMode:       25.0,
					schema.ComplexityMode: 40.0,
					schema.StaleMode:      20.0,
				},
				Lookback: 180 * 24 * time.Hour,
			},
		},
		{
			name: "some failed",
			result: schema.CheckResult{
				Passed: false,
				FailedFiles: []schema.CheckFailedFile{
					{
						Path:      "main.go",
						Mode:      schema.HotMode,
						Score:     75.5,
						Threshold: 50.0,
					},
				},
				TotalFiles:   5,
				CheckedModes: []schema.ScoringMode{schema.HotMode, schema.RiskMode},
				BaseRef:      "main",
				TargetRef:    "HEAD",
				Thresholds: map[schema.ScoringMode]float64{
					schema.HotMode:        50.0,
					schema.RiskMode:       50.0,
					schema.ComplexityMode: 50.0,
					schema.StaleMode:      50.0,
				},
				MaxScores: map[schema.ScoringMode]float64{
					schema.HotMode:        75.5,
					schema.RiskMode:       45.0,
					schema.ComplexityMode: 60.0,
					schema.StaleMode:      30.0,
				},
				Lookback: 180 * 24 * time.Hour,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure it doesn't panic
			assert.NotPanics(t, func() {
				printCheckResult(&tt.result, time.Second)
			})
		})
	}
}

func TestExecuteHotspotCheck_MissingCompareMode(t *testing.T) {
	ctx := context.Background()

	// Create config without compare mode
	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		CompareMode: false,
	}

	// Create mock cache manager
	mockManager := &iocache.MockCacheManager{}

	// Execute should return error
	err := ExecuteHotspotCheck(ctx, cfg, mockManager)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base-ref and --target-ref")
}
