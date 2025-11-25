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
