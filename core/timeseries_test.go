package core

import (
	"context"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunTimeseriesAnalysis_Success(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	interval := 30 * 24 * time.Hour // 30 days
	numPoints := 3
	path := "main.go"

	// Setup mock expectations for each time point
	// Point 0: current time window
	mockClient.On("GetOldestCommitDateForPath", ctx, "/test/repo", path, now, minCommits, maxSearchDuration).
		Return(now.Add(-60*24*time.Hour), nil) // 60 days ago

	// Point 1: 30 days ago
	point1End := now.Add(-interval)
	mockClient.On("GetOldestCommitDateForPath", ctx, "/test/repo", path, point1End, minCommits, maxSearchDuration).
		Return(point1End.Add(-45*24*time.Hour), nil) // 45 days before point1End

	// Point 2: 60 days ago
	point2End := now.Add(-2 * interval)
	mockClient.On("GetOldestCommitDateForPath", ctx, "/test/repo", path, point2End, minCommits, maxSearchDuration).
		Return(point2End.Add(-30*24*time.Hour), nil) // 30 days before point2End

	// Mock the analysis calls for each point (simplified - just return success)
	mockMgr.On("GetActivityStore").Return(nil).Maybe() // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil).Maybe() // No analysis tracking for test
	mockClient.On("ListFilesAtRef", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "HEAD").Return([]string{path}, nil).Maybe()
	mockClient.On("GetActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil).Maybe()

	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		Mode:        schema.HotMode,
		Workers:     1,
		ResultLimit: 10,
	}

	result := runTimeseriesAnalysis(ctx, cfg, mockClient, path, false, now, interval, numPoints, mockMgr)

	assert.Len(t, result, numPoints)

	// Check period labels
	assert.Equal(t, "0-30d ago", result[0].Period)
	assert.Equal(t, "30-60d ago", result[1].Period)
	assert.Equal(t, "60-90d ago", result[2].Period)

	// Check that all points have valid structure (scores may be 0 due to mock data)
	for _, point := range result {
		assert.True(t, point.Score >= 0) // Score should be non-negative
		assert.Equal(t, path, point.Path)
		assert.Equal(t, schema.HotMode, point.Mode)
		assert.True(t, point.Start.Before(point.End))
	}

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunTimeseriesAnalysis_GetOldestCommitError(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	interval := 30 * 24 * time.Hour
	numPoints := 2
	path := "main.go"

	// Setup mock expectations - GetOldestCommitDateForPath fails
	mockMgr.On("GetActivityStore").Return(nil).Maybe() // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil).Maybe() // No analysis tracking for test
	mockClient.On("GetOldestCommitDateForPath", ctx, "/test/repo", path, mock.AnythingOfType("time.Time"), minCommits, maxSearchDuration).
		Return(time.Time{}, assert.AnError).Maybe()
	mockClient.On("ListFilesAtRef", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "HEAD").Return([]string{path}, nil).Maybe()
	mockClient.On("GetActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]byte(""), nil).Maybe() // Empty log for fallback case

	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		Mode:        schema.HotMode,
		Workers:     1,
		ResultLimit: 10,
	}

	result := runTimeseriesAnalysis(ctx, cfg, mockClient, path, false, now, interval, numPoints, mockMgr)

	assert.Len(t, result, numPoints)

	// Should still create points but with fallback lookback
	for _, point := range result {
		assert.Equal(t, path, point.Path)
		assert.True(t, point.Lookback >= minLookback)
	}

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestAnalyzeTimeseriesPoint_File(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	path := "main.go"

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test
	mockClient.On("ListFilesAtRef", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "HEAD").Return([]string{path}, nil)
	mockClient.On("GetActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil)

	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:        schema.HotMode,
		Workers:     1,
		ResultLimit: 10,
	}

	score, owners := analyzeTimeseriesPoint(ctx, cfg, mockClient, path, false, mockMgr)

	assert.True(t, score >= 0 && score <= 100)
	assert.NotNil(t, owners)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestAnalyzeTimeseriesPoint_Folder(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	path := "src/"

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test
	mockClient.On("ListFilesAtRef", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "HEAD").Return([]string{"src/main.go", "src/utils.go"}, nil)
	mockClient.On("GetActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tsrc/main.go\n2\t1\tsrc/utils.go\n"), nil)

	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:        schema.HotMode,
		Workers:     1,
		ResultLimit: 10,
		PathFilter:  "src/", // Only analyze files in src/
	}

	score, owners := analyzeTimeseriesPoint(ctx, cfg, mockClient, path, true, mockMgr)

	assert.True(t, score >= 0 && score <= 100)
	assert.NotNil(t, owners)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestAnalyzeTimeseriesPoint_NoData(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	path := "nonexistent.go"

	// Setup mock expectations - no files found
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test
	mockClient.On("ListFilesAtRef", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "HEAD").Return([]string{}, nil)
	mockClient.On("GetActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]byte(""), nil)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
	}

	score, owners := analyzeTimeseriesPoint(ctx, cfg, mockClient, path, false, mockMgr)

	assert.Equal(t, 0.0, score)
	assert.Empty(t, owners)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestAnalyzeTimeseriesPoint_PathNotFound(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	path := "missing.go"

	// Setup mock expectations - file exists but path not found in results
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test
	mockClient.On("ListFilesAtRef", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "HEAD").Return([]string{"other.go"}, nil)
	mockClient.On("GetActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tother.go\n"), nil)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
	}

	score, owners := analyzeTimeseriesPoint(ctx, cfg, mockClient, path, false, mockMgr)

	assert.Equal(t, 0.0, score)
	assert.Empty(t, owners)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}
