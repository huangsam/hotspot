package core

import (
	"context"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/outwriter"
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
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/nonexistent/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotFiles(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr, outwriter.NewOutWriter())

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
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/nonexistent/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotFolders(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr, outwriter.NewOutWriter())

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
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/nonexistent/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
		Compare: config.CompareConfig{
			Enabled:   true,
			BaseRef:   "main",
			TargetRef: "feature",
		},
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotCompare(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr, outwriter.NewOutWriter())

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)
}

// TestExecuteHotspotCompareFolders tests the folder comparison entry point.
func TestExecuteHotspotCompareFolders(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}

	// Create config with compare mode enabled
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/nonexistent/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
		Compare: config.CompareConfig{
			Enabled:   true,
			BaseRef:   "main",
			TargetRef: "feature",
		},
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotCompareFolders(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr, outwriter.NewOutWriter())

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)
}

// TestExecuteHotspotTimeseries tests the timeseries analysis entry point.
func TestExecuteHotspotTimeseries(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}

	// Create config with timeseries parameters
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/nonexistent/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
		Timeseries: config.TimeseriesConfig{
			Path:     "main.go",
			Interval: time.Hour * 24 * 30, // 30 days
			Points:   3,
		},
	}

	// Execute - should fail due to non-existent repo
	err := ExecuteHotspotTimeseries(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr, outwriter.NewOutWriter())

	// Assert that we get an error (expected since repo doesn't exist)
	assert.Error(t, err)
}

// TestExecuteHotspotMetrics tests the metrics display entry point.
func TestExecuteHotspotMetrics(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager (not used for metrics)
	mockCacheMgr := &iocache.MockCacheManager{}

	// Create config
	cfg := &config.Config{
		Output: config.OutputConfig{
			Format: schema.TextOut,
		},
	}

	// Execute - should succeed (metrics is static)
	err := ExecuteHotspotMetrics(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr, outwriter.NewOutWriter())

	// Assert that it succeeds
	assert.NoError(t, err)
}

// TestExecuteHotspotCheck tests the check command execution.
func TestExecuteHotspotCheck(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}

	// Test without compare mode (should fail)
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/nonexistent/repo",
		},
		Compare: config.CompareConfig{
			Enabled: false,
		},
	}
	err := ExecuteHotspotCheck(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "check command requires --base-ref and --target-ref flags")

	// Test with compare mode but non-existent repo (should fail)
	cfg = &config.Config{
		Git: config.GitConfig{
			RepoPath: "/nonexistent/repo",
		},
		Compare: config.CompareConfig{
			Enabled:   true,
			BaseRef:   "main",
			TargetRef: "feature",
			Lookback:  time.Hour * 24 * 30,
		},
		Scoring: config.ScoringConfig{
			RiskThresholds: map[schema.ScoringMode]float64{
				schema.HotMode:        50.0,
				schema.RiskMode:       50.0,
				schema.ComplexityMode: 50.0,
			},
		},
	}
	err = ExecuteHotspotCheck(ctx, cfg, git.NewLocalGitClient(), mockCacheMgr)
	assert.Error(t, err) // Should fail due to non-existent repo
}

func TestExecuteHotspotCheck_MissingCompareMode(t *testing.T) {
	ctx := context.Background()

	// Create config without compare mode
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
		Compare: config.CompareConfig{
			Enabled: false,
		},
	}

	// Create mock cache manager
	mockManager := &iocache.MockCacheManager{}

	// Execute should return error
	err := ExecuteHotspotCheck(ctx, cfg, git.NewLocalGitClient(), mockManager)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base-ref and --target-ref")
}
