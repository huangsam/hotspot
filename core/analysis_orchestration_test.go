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

func TestRunSingleAnalysisCore_Success(t *testing.T) {
	ctx := withSuppressHeader(context.Background())
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return([]string{"main.go", "core/agg.go"}, nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil)

	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:        schema.HotMode,
		Workers:     1,
		ResultLimit: 10,
	}

	result, err := runSingleAnalysisCore(ctx, cfg, mockClient, mockMgr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.FileResults)
	assert.NotNil(t, result.AggregateOutput)
	assert.True(t, len(result.FileResults) > 0)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunSingleAnalysisCore_NoFilesFound(t *testing.T) {
	ctx := withSuppressHeader(context.Background())
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	// Setup mock expectations - return empty file list
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return([]string{}, nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte(""), nil)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
	}

	result, err := runSingleAnalysisCore(ctx, cfg, mockClient, mockMgr)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no files found")

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunSingleAnalysisCore_AggregationError(t *testing.T) {
	ctx := withSuppressHeader(context.Background())
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	// Setup mock expectations - aggregation fails
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return([]string{"main.go"}, nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, assert.AnError)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
	}

	result, err := runSingleAnalysisCore(ctx, cfg, mockClient, mockMgr)

	assert.Error(t, err)
	assert.Nil(t, result)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunCompareAnalysisForRef(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "main"
	lookback := 30 * 24 * time.Hour // 30 days
	commitTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("GetCommitTime", ctx, "/test/repo", ref).Return(commitTime, nil)
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", ref).Return([]string{"main.go", "core/agg.go"}, nil)
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return([]string{"main.go", "core/agg.go"}, nil) // For aggregateActivity
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte("--abc123|Alice|2024-06-01T00:00:00Z\n1\t0\tmain.go\n"), nil)

	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		Mode:        schema.HotMode,
		Workers:     1,
		ResultLimit: 10,
		Lookback:    lookback,
	}

	result, err := runCompareAnalysisForRef(ctx, cfg, mockClient, ref, mockMgr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.FileResults)
	assert.NotNil(t, result.FolderResults)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunCompareAnalysisForRef_CommitTimeError(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "main"

	// Setup mock expectations - commit time lookup fails
	mockClient.On("GetCommitTime", ctx, "/test/repo", ref).Return(time.Time{}, assert.AnError)

	cfg := &contract.Config{
		RepoPath: "/test/repo",
		Lookback: 30 * 24 * time.Hour,
	}

	result, err := runCompareAnalysisForRef(ctx, cfg, mockClient, ref, mockMgr)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to resolve time window")

	mockClient.AssertExpectations(t)
}

func TestAnalyzeAllFilesAtRef(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "feature-branch"

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", ref).Return([]string{"main.go", "core/agg.go", "test_main.go"}, nil)
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return([]string{"main.go", "core/agg.go", "test_main.go"}, nil) // For aggregateActivity
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
		Excludes:  []string{"test_*"}, // Should exclude test_main.go
	}

	result, err := analyzeAllFilesAtRef(ctx, cfg, mockClient, ref, mockMgr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, len(result) > 0)

	// Should not include test_main.go due to exclude pattern
	for _, r := range result {
		assert.NotContains(t, r.Path, "test_main.go")
	}

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestAnalyzeAllFilesAtRef_EmptyAfterFiltering(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "feature-branch"

	// Setup mock expectations - all files get filtered out
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", ref).Return([]string{"test_main.go", "test_utils.go"}, nil)
	// No GetActivityLog mock needed since all files are filtered out before aggregation

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   1,
		Excludes:  []string{"test_*"}, // Excludes all files
	}

	result, err := analyzeAllFilesAtRef(ctx, cfg, mockClient, ref, mockMgr)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(result)) // Should return empty slice, not error

	mockClient.AssertExpectations(t)
}

func TestRunFollowPass(t *testing.T) {
	ctx := withSuppressHeader(context.Background())
	mockClient := &contract.MockGitClient{}

	// Create test data
	ranked := []schema.FileResult{
		{Path: "main.go", Score: 10.0},
		{Path: "core/agg.go", Score: 8.0},
	}

	output := &schema.AggregateOutput{
		CommitMap: map[string]int{"main.go": 5, "core/agg.go": 3},
		ChurnMap:  map[string]int{"main.go": 15, "core/agg.go": 9},
		ContribMap: map[string]map[string]int{
			"main.go":     {"Alice": 5},
			"core/agg.go": {"Bob": 3},
		},
		FirstCommitMap: map[string]time.Time{
			"main.go":     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			"core/agg.go": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		StartTime:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:        schema.HotMode,
		Workers:     1,
		ResultLimit: 2,
	}

	// Mock the GetFileActivityLog call that will be made with --follow
	mockClient.On("GetFileActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "main.go", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), true).
		Return([]byte("--follow123|Alice|2024-01-01T00:00:00Z\nDELIMITER_COMMIT_STARTAlice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil)
	mockClient.On("GetFileActivityLog", mock.AnythingOfType("*context.valueCtx"), "/test/repo", "core/agg.go", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), true).
		Return([]byte("--follow456|Bob|2024-01-01T00:00:00Z\nDELIMITER_COMMIT_STARTBob|2024-01-01T00:00:00Z\n1\t0\tcore/agg.go\n"), nil)

	result := runFollowPass(ctx, cfg, mockClient, ranked, output)

	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	// Results should be re-ranked by score (descending)
	assert.GreaterOrEqual(t, result[0].Score, result[1].Score)

	mockClient.AssertExpectations(t)
}

func TestRunFollowPass_EmptyInput(t *testing.T) {
	ctx := withSuppressHeader(context.Background())
	mockClient := &contract.MockGitClient{}

	ranked := []schema.FileResult{}
	output := &schema.AggregateOutput{}

	cfg := &contract.Config{
		ResultLimit: 10,
	}

	result := runFollowPass(ctx, cfg, mockClient, ranked, output)

	assert.Equal(t, ranked, result) // Should return input unchanged

	mockClient.AssertExpectations(t)
}

func TestGetAnalysisWindowForRef(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}

	ref := "v1.0.0"
	lookback := 90 * 24 * time.Hour // 90 days
	commitTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	expectedStart := commitTime.Add(-lookback)

	// Setup mock expectations
	mockClient.On("GetCommitTime", ctx, "/test/repo", ref).Return(commitTime, nil)

	startTime, endTime, err := getAnalysisWindowForRef(ctx, mockClient, "/test/repo", ref, lookback)

	assert.NoError(t, err)
	assert.Equal(t, expectedStart, startTime)
	assert.Equal(t, commitTime, endTime)

	mockClient.AssertExpectations(t)
}

func TestGetAnalysisWindowForRef_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}

	ref := "nonexistent-ref"

	// Setup mock expectations - commit time lookup fails
	mockClient.On("GetCommitTime", ctx, "/test/repo", ref).Return(time.Time{}, assert.AnError)

	startTime, endTime, err := getAnalysisWindowForRef(ctx, mockClient, "/test/repo", ref, 30*24*time.Hour)

	assert.Error(t, err)
	assert.True(t, startTime.IsZero())
	assert.True(t, endTime.IsZero())
	assert.Contains(t, err.Error(), "failed to get analysis window")

	mockClient.AssertExpectations(t)
}
