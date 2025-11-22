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

// TestExecuteHotspotFiles tests the main file analysis entry point.
func TestExecuteHotspotFiles(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}
	mockCacheMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockCacheMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test

	// Create config - this will fail because we're not in a real git repo
	cfg := &contract.Config{
		RepoPath:  "/nonexistent/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotFiles(ctx, cfg, mockCacheMgr)

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)

	// Verify mocks were called
	mockCacheMgr.AssertExpectations(t)
}

// TestExecuteHotspotFolders tests the main folder analysis entry point.
func TestExecuteHotspotFolders(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}
	mockCacheMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockCacheMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test

	// Create config - this will fail because we're not in a real git repo
	cfg := &contract.Config{
		RepoPath:  "/nonexistent/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotFolders(ctx, cfg, mockCacheMgr)

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)

	// Verify mocks were called
	mockCacheMgr.AssertExpectations(t)
}

// TestExecuteHotspotCompare tests the file comparison entry point.
func TestExecuteHotspotCompare(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}

	// Create config with compare mode enabled
	cfg := &contract.Config{
		RepoPath:    "/nonexistent/repo",
		StartTime:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:        schema.HotMode,
		Workers:     1,
		CompareMode: true,
		BaseRef:     "main",
		TargetRef:   "feature",
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotCompare(ctx, cfg, mockCacheMgr)

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)
}

// TestExecuteHotspotCompareFolders tests the folder comparison entry point.
func TestExecuteHotspotCompareFolders(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}

	// Create config with compare mode enabled
	cfg := &contract.Config{
		RepoPath:    "/nonexistent/repo",
		StartTime:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:        schema.HotMode,
		Workers:     1,
		CompareMode: true,
		BaseRef:     "main",
		TargetRef:   "feature",
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotCompareFolders(ctx, cfg, mockCacheMgr)

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)
}

// TestExecuteHotspotTimeseries tests the timeseries analysis entry point.
func TestExecuteHotspotTimeseries(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}

	// Create config with timeseries parameters
	cfg := &contract.Config{
		RepoPath:           "/nonexistent/repo",
		StartTime:          time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:            time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:               schema.HotMode,
		Workers:            1,
		TimeseriesPath:     "main.go",
		TimeseriesInterval: time.Hour * 24 * 30, // 30 days
		TimeseriesPoints:   3,
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotTimeseries(ctx, cfg, mockCacheMgr)

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)
}

// TestExecuteHotspotMetrics tests the metrics display entry point.
func TestExecuteHotspotMetrics(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager (not used for metrics)
	mockCacheMgr := &iocache.MockCacheManager{}

	// Create config
	cfg := &contract.Config{
		Output: schema.TextOut,
	}

	// Execute - should succeed (metrics is static)
	err := ExecuteHotspotMetrics(ctx, cfg, mockCacheMgr)

	// Assert that it succeeds
	assert.NoError(t, err)
}

// TestExecuteHotspotCheck tests the check command execution.
func TestExecuteHotspotCheck(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}

	// Test without compare mode (should fail)
	cfg := &contract.Config{
		RepoPath:    "/nonexistent/repo",
		CompareMode: false,
	}
	err := ExecuteHotspotCheck(ctx, cfg, mockCacheMgr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check requires --base-ref and --target-ref flags")

	// Test with compare mode but non-existent repo (should fail)
	cfg = &contract.Config{
		RepoPath:    "/nonexistent/repo",
		CompareMode: true,
		BaseRef:     "main",
		TargetRef:   "feature",
		Lookback:    time.Hour * 24 * 30,
		RiskThresholds: map[schema.ScoringMode]float64{
			schema.HotMode:        50.0,
			schema.RiskMode:       50.0,
			schema.ComplexityMode: 50.0,
			schema.StaleMode:      50.0,
		},
	}
	err = ExecuteHotspotCheck(ctx, cfg, mockCacheMgr)
	assert.Error(t, err) // Should fail due to non-existent repo
}

// TestRecordFileAnalysis tests the file analysis recording function.
func TestRecordFileAnalysis(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}
	mockCacheMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test

	// Create config
	cfg := &contract.Config{
		Mode: schema.HotMode,
	}

	// Create file result with all scores
	fileResult := &schema.FileResult{
		Path:               "test.go",
		Commits:            100,
		Churn:              500,
		UniqueContributors: 5,
		AgeDays:            365,
		Gini:               0.3,
		Owners:             []string{"alice", "bob"},
		AllScores: map[schema.ScoringMode]float64{
			schema.HotMode:        75.5,
			schema.RiskMode:       80.2,
			schema.ComplexityMode: 65.3,
			schema.StaleMode:      70.1,
		},
	}

	// Set up context with cache manager
	ctx = contextWithCacheManager(ctx, mockCacheMgr)

	// Execute - should not panic
	recordFileAnalysis(ctx, cfg, 1, "test.go", fileResult)

	// Verify mocks were called
	mockCacheMgr.AssertExpectations(t)
}

// TestGetOwnerString tests the owner string conversion.
func TestGetOwnerString(t *testing.T) {
	tests := []struct {
		name     string
		owners   []string
		expected string
	}{
		{
			name:     "empty owners",
			owners:   []string{},
			expected: "",
		},
		{
			name:     "single owner",
			owners:   []string{"alice"},
			expected: "alice",
		},
		{
			name:     "multiple owners",
			owners:   []string{"alice", "bob", "charlie"},
			expected: "alice", // Should return first owner
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOwnerString(tt.owners)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFilterChangedFilesCore tests the file filtering logic (core version).
func TestFilterChangedFilesCore(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		excludes []string
		expected []string
	}{
		{
			name:     "no excludes",
			files:    []string{"main.go", "core/agg.go", "README.md"},
			excludes: []string{},
			expected: []string{"main.go", "core/agg.go", "README.md"},
		},
		{
			name:     "exclude vendor",
			files:    []string{"main.go", "vendor/lib.go", "core/agg.go"},
			excludes: []string{"vendor/"},
			expected: []string{"main.go", "core/agg.go"},
		},
		{
			name:     "exclude multiple patterns",
			files:    []string{"main.go", "vendor/lib.go", "test_main.go", "core/agg.go"},
			excludes: []string{"vendor/", "test_"},
			expected: []string{"main.go", "core/agg.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterChangedFiles(tt.files, tt.excludes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
