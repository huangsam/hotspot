package core

import (
	"context"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRunSingleAnalysisCore_Success(t *testing.T) {
	ctx := WithSuppressHeader(context.Background())
	mockClient := &git.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test
	mockClient.On("GetRemoteURL", mock.Anything, "/test/repo").Return("https://github.com/test/repo", nil).Maybe()
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", "HEAD").Return([]string{"main.go", "core/agg.go"}, nil)
	mockClient.On("GetActivityLog", mock.Anything, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
		Output: config.OutputConfig{
			ResultLimit: 10,
		},
	}

	result, err := runSingleAnalysisCore(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, cfg.Output, cfg.Compare, mockClient, mockMgr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.FileResults)
	assert.NotNil(t, result.AggregateOutput)
	assert.True(t, len(result.FileResults) > 0)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunSingleAnalysisCore_NoFilesFound(t *testing.T) {
	ctx := WithSuppressHeader(context.Background())
	mockClient := &git.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	// Setup mock expectations - return empty file list
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test
	mockClient.On("GetRemoteURL", mock.Anything, "/test/repo").Return("https://github.com/test/repo", nil).Maybe()
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", "HEAD").Return([]string{}, nil)
	mockClient.On("GetActivityLog", mock.Anything, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte(""), nil)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
	}

	result, err := runSingleAnalysisCore(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, cfg.Output, cfg.Compare, mockClient, mockMgr)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no files found")

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunSingleAnalysisCore_AggregationError(t *testing.T) {
	ctx := WithSuppressHeader(context.Background())
	mockClient := &git.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	// Setup mock expectations - aggregation fails
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test
	mockClient.On("GetRemoteURL", mock.Anything, "/test/repo").Return("https://github.com/test/repo", nil).Maybe()
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", "HEAD").Return([]string{"main.go"}, nil)
	mockClient.On("GetActivityLog", mock.Anything, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil, assert.AnError)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
	}

	result, err := runSingleAnalysisCore(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, cfg.Output, cfg.Compare, mockClient, mockMgr)

	assert.Error(t, err)
	assert.Nil(t, result)

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunCompareAnalysisForRef(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "main"
	lookback := 30 * 24 * time.Hour // 30 days
	commitTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("GetRemoteURL", mock.Anything, "/test/repo").Return("https://github.com/test/repo", nil).Maybe()
	mockClient.On("GetCommitTime", mock.Anything, "/test/repo", ref).Return(commitTime, nil)
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", ref).Return([]string{"main.go", "core/agg.go"}, nil)
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", "HEAD").Return([]string{"main.go", "core/agg.go"}, nil) // For aggregateActivity
	mockClient.On("GetActivityLog", mock.Anything, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte("--abc123|Alice|2024-06-01T00:00:00Z\n1\t0\tmain.go\n"), nil)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
		Output: config.OutputConfig{
			ResultLimit: 10,
		},
		Compare: config.CompareConfig{
			Lookback: lookback,
		},
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
	mockClient := &git.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "main"

	// Setup mock expectations - commit time lookup fails
	mockClient.On("GetCommitTime", mock.Anything, "/test/repo", ref).Return(time.Time{}, assert.AnError)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
		Compare: config.CompareConfig{
			Lookback: 30 * 24 * time.Hour,
		},
	}

	result, err := runCompareAnalysisForRef(ctx, cfg, mockClient, ref, mockMgr)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to resolve time window")

	mockClient.AssertExpectations(t)
}

func TestAnalyzeAllFilesAtRef(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "feature-branch"

	// Setup mock expectations
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("GetRemoteURL", mock.Anything, "/test/repo").Return("https://github.com/test/repo", nil).Maybe()
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", ref).Return([]string{"main.go", "core/agg.go", "test_main.go"}, nil)
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", "HEAD").Return([]string{"main.go", "core/agg.go", "test_main.go"}, nil) // For aggregateActivity
	mockClient.On("GetActivityLog", mock.Anything, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte("--abc123|Alice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			Excludes:  []string{"test_*"}, // Should exclude test_main.go
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
	}

	result, err := analyzeAllFilesAtRef(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, mockClient, ref, mockMgr)

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
	mockClient := &git.MockGitClient{}
	mockMgr := &iocache.MockCacheManager{}

	ref := "feature-branch"

	// Setup mock expectations - all files get filtered out
	mockMgr.On("GetActivityStore").Return(nil) // No caching for test
	mockClient.On("GetRemoteURL", mock.Anything, "/test/repo").Return("https://github.com/test/repo", nil).Maybe()
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", ref).Return([]string{"test_main.go", "test_utils.go"}, nil)
	mockClient.On("ListFilesAtRef", mock.Anything, "/test/repo", "HEAD").Return([]string{"test_main.go", "test_utils.go"}, nil)
	mockClient.On("GetActivityLog", mock.Anything, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]byte(""), nil)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			Excludes:  []string{"test_*"}, // Excludes all files
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
	}

	result, err := analyzeAllFilesAtRef(ctx, cfg.Git, cfg.Scoring, cfg.Runtime, mockClient, ref, mockMgr)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(result)) // Should return empty slice, not error

	mockClient.AssertExpectations(t)
	mockMgr.AssertExpectations(t)
}

func TestRunFollowPass(t *testing.T) {
	ctx := WithSuppressHeader(context.Background())
	mockClient := &git.MockGitClient{}

	// Create test data
	ranked := []schema.FileResult{
		{Path: "main.go", ModeScore: 10.0},
		{Path: "core/agg.go", ModeScore: 8.0},
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

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Runtime: config.RuntimeConfig{
			Workers: 1,
		},
		Output: config.OutputConfig{
			ResultLimit: 2,
		},
	}

	// Mock the GetFileActivityLog call that will be made with --follow
	mockClient.On("GetFileActivityLog", mock.Anything, "/test/repo", "main.go", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), true).
		Return([]byte("--follow123|Alice|2024-01-01T00:00:00Z\nDELIMITER_COMMIT_STARTAlice|2024-01-01T00:00:00Z\n1\t0\tmain.go\n"), nil)
	mockClient.On("GetFileActivityLog", mock.Anything, "/test/repo", "core/agg.go", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), true).
		Return([]byte("--follow456|Bob|2024-01-01T00:00:00Z\nDELIMITER_COMMIT_STARTBob|2024-01-01T00:00:00Z\n1\t0\tcore/agg.go\n"), nil)

	result := runFollowPass(ctx, cfg.Git, cfg.Scoring, cfg.Output, mockClient, ranked, output)

	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	// Results should be re-ranked by score (descending)
	assert.GreaterOrEqual(t, result[0].ModeScore, result[1].ModeScore)

	mockClient.AssertExpectations(t)
}

func TestRunFollowPass_EmptyInput(t *testing.T) {
	ctx := WithSuppressHeader(context.Background())
	mockClient := &git.MockGitClient{}

	var ranked []schema.FileResult
	output := &schema.AggregateOutput{}

	cfg := &config.Config{
		Output: config.OutputConfig{
			ResultLimit: 10,
		},
	}

	result := runFollowPass(ctx, cfg.Git, cfg.Scoring, cfg.Output, mockClient, ranked, output)

	assert.Equal(t, ranked, result) // Should return input unchanged

	mockClient.AssertExpectations(t)
}

func TestGetAnalysisWindowForRef(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}

	ref := "v1.0.0"
	lookback := 90 * 24 * time.Hour // 90 days
	commitTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	expectedStart := commitTime.Add(-lookback)

	// Setup mock expectations
	mockClient.On("GetCommitTime", mock.Anything, "/test/repo", ref).Return(commitTime, nil)

	startTime, endTime, err := getAnalysisWindowForRef(ctx, mockClient, "/test/repo", ref, lookback)

	assert.NoError(t, err)
	assert.Equal(t, expectedStart, startTime)
	assert.Equal(t, commitTime, endTime)

	mockClient.AssertExpectations(t)
}

func TestGetAnalysisWindowForRef_Error(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}

	ref := "nonexistent-ref"

	// Setup mock expectations - commit time lookup fails
	mockClient.On("GetCommitTime", mock.Anything, "/test/repo", ref).Return(time.Time{}, assert.AnError)

	startTime, endTime, err := getAnalysisWindowForRef(ctx, mockClient, "/test/repo", ref, 30*24*time.Hour)

	assert.Error(t, err)
	assert.True(t, startTime.IsZero())
	assert.True(t, endTime.IsZero())
	assert.Contains(t, err.Error(), "failed to get analysis window")

	mockClient.AssertExpectations(t)
}
