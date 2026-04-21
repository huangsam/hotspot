package agg

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCacheStore for testing (alias for MockCacheStore).
type MockCacheStore = iocache.MockCacheStore

// MockCacheManager for testing CachedAggregateActivity.
type MockCacheManager struct {
	mock.Mock
}

func (m *MockCacheManager) GetActivityStore() iocache.CacheStore {
	ret := m.Called()
	val := ret.Get(0)
	if val == nil {
		return nil
	}
	return val.(iocache.CacheStore)
}

func (m *MockCacheManager) GetAnalysisStore() iocache.AnalysisStore {
	ret := m.Called()
	val := ret.Get(0)
	if val == nil {
		return nil
	}
	return val.(iocache.AnalysisStore)
}

func TestCheckCacheHit_CacheHit(t *testing.T) {
	mockStore := &MockCacheStore{}
	result := &schema.AggregateOutput{
		FileStats: map[string]*schema.FileAggregation{
			"test.go": {Commits: 5},
		},
	}
	data, _ := json.Marshal(result)

	// Valid cache entry: current version, recent timestamp
	mockStore.On("Get", "test-key").Return(data, currentCacheVersion, time.Now().Unix(), nil)

	actual := checkCacheHit(mockStore, "test-key")
	assert.NotNil(t, actual)
	assert.Equal(t, schema.Metric(5), actual.FileStats["test.go"].Commits)
	mockStore.AssertExpectations(t)
}

func TestCheckCacheHit_CacheMiss_VersionMismatch(t *testing.T) {
	mockStore := &MockCacheStore{}
	data := []byte("{}")

	// Version mismatch
	mockStore.On("Get", "test-key").Return(data, currentCacheVersion-1, time.Now().Unix(), nil)

	actual := checkCacheHit(mockStore, "test-key")
	assert.Nil(t, actual)
	mockStore.AssertExpectations(t)
}

func TestCheckCacheHit_CacheMiss_Stale(t *testing.T) {
	mockStore := &MockCacheStore{}
	data := []byte("{}")

	// Stale entry (older than 7 days)
	staleTime := time.Now().Add(-8 * 24 * time.Hour).Unix()
	mockStore.On("Get", "test-key").Return(data, currentCacheVersion, staleTime, nil)

	actual := checkCacheHit(mockStore, "test-key")
	assert.Nil(t, actual)
	mockStore.AssertExpectations(t)
}

func TestCheckCacheHit_CacheMiss_Error(t *testing.T) {
	mockStore := &MockCacheStore{}

	// Simulate DB error
	mockStore.On("Get", "test-key").Return([]byte{}, 0, int64(0), assert.AnError)

	actual := checkCacheHit(mockStore, "test-key")
	assert.Nil(t, actual)
	mockStore.AssertExpectations(t)
}

func TestCheckCacheHit_CacheMiss_UnmarshalError(t *testing.T) {
	mockStore := &MockCacheStore{}

	// Invalid JSON data
	mockStore.On("Get", "test-key").Return([]byte("invalid json"), currentCacheVersion, time.Now().Unix(), nil)

	actual := checkCacheHit(mockStore, "test-key")
	assert.Nil(t, actual)
	mockStore.AssertExpectations(t)
}

func TestGenerateCacheKey(t *testing.T) {
	mockClient := &git.MockGitClient{}
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Unix(1000000, 0),
			EndTime:   time.Unix(2000000, 0),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Compare: config.CompareConfig{
			Lookback: 30 * 24 * time.Hour,
		},
	}

	// Mock GetRepoHash for any repo path
	mockClient.On("GetRepoHash", mock.Anything, mock.AnythingOfType("string")).Return("abcd1234", nil)
	mockClient.On("GetRemoteURL", mock.Anything, mock.AnythingOfType("string")).Return("", nil).Maybe()
	mockClient.On("GetRootCommitHash", mock.Anything, mock.AnythingOfType("string")).Return("root123", nil).Maybe()
	mockClient.On("GetRepoRoot", mock.Anything, mock.AnythingOfType("string")).Return("/test/repo", nil).Maybe()

	key1 := generateCacheKey(context.Background(), cfg.Git, cfg.Compare, mockClient, "")

	// Key should be a non-empty SHA256 hash
	assert.NotEmpty(t, key1)
	assert.Len(t, key1, 64) // SHA256 hash length

	// Different config (different URN) should produce different key
	key2 := generateCacheKey(context.Background(), cfg.Git, cfg.Compare, mockClient, "git:github.com/different/repo")
	assert.NotEqual(t, key1, key2)

	mockClient.AssertExpectations(t)
}

func TestGenerateCacheKey_RepoHashError(t *testing.T) {
	mockClient := &git.MockGitClient{}
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Unix(1000000, 0),
			EndTime:   time.Unix(2000000, 0),
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
		Compare: config.CompareConfig{
			Lookback: 30 * 24 * time.Hour,
		},
	}

	// Mock GetRepoHash to return error
	mockClient.On("GetRepoHash", mock.Anything, mock.AnythingOfType("string")).Return("", assert.AnError)
	mockClient.On("GetRemoteURL", mock.Anything, mock.AnythingOfType("string")).Return("", nil).Maybe()
	mockClient.On("GetRootCommitHash", mock.Anything, mock.AnythingOfType("string")).Return("root123", nil).Maybe()
	mockClient.On("GetRepoRoot", mock.Anything, mock.AnythingOfType("string")).Return("/test/repo", nil).Maybe()

	key := generateCacheKey(context.Background(), cfg.Git, cfg.Compare, mockClient, "")

	// Key should still be generated (with empty repoHash)
	assert.NotEmpty(t, key)
	assert.Len(t, key, 64) // SHA256 hash length

	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}
	mockMgr := &MockCacheManager{}
	mockStore := &MockCacheStore{}

	// Expected cached result
	expected := &schema.AggregateOutput{
		FileStats: map[string]*schema.FileAggregation{
			"test.go": {Commits: 5},
		},
	}

	data, _ := json.Marshal(expected)

	// Setup mocks
	mockMgr.On("GetActivityStore").Return(mockStore)
	mockStore.On("Get", mock.AnythingOfType("string")).Return(data, currentCacheVersion, time.Now().Unix(), nil)
	mockClient.On("GetRepoHash", ctx, "/test/repo").Return("abcd1234", nil)
	mockClient.On("GetRemoteURL", mock.Anything, mock.AnythingOfType("string")).Return("", nil).Maybe()
	mockClient.On("GetRootCommitHash", mock.Anything, mock.AnythingOfType("string")).Return("root123", nil).Maybe()
	mockClient.On("GetRepoRoot", mock.Anything, mock.AnythingOfType("string")).Return("/test/repo", nil).Maybe()

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Unix(1000000, 0),
			EndTime:   time.Unix(2000000, 0),
		},
	}

	result, err := CachedAggregateActivity(ctx, cfg.Git, cfg.Compare, mockClient, mockMgr, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expected, result)

	mockMgr.AssertExpectations(t)
	mockStore.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_CacheMiss(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}
	mockMgr := &MockCacheManager{}
	mockStore := &MockCacheStore{}

	// Setup for aggregateActivity
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return(strings.Split(strings.TrimSpace(fileListFixture), "\n"), nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(gitLogBasicFixture, nil)
	mockClient.On("GetRepoHash", ctx, "/test/repo").Return("abcd1234", nil)
	mockClient.On("GetRemoteURL", mock.Anything, mock.AnythingOfType("string")).Return("", nil).Maybe()
	mockClient.On("GetRootCommitHash", mock.Anything, mock.AnythingOfType("string")).Return("root123", nil).Maybe()
	mockClient.On("GetRepoRoot", mock.Anything, mock.AnythingOfType("string")).Return("/test/repo", nil).Maybe()

	// Cache miss
	mockMgr.On("GetActivityStore").Return(mockStore)
	mockStore.On("Get", mock.AnythingOfType("string")).Return([]byte{}, 0, int64(0), assert.AnError)
	mockStore.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), currentCacheVersion, mock.AnythingOfType("int64")).Return(nil)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	result, err := CachedAggregateActivity(ctx, cfg.Git, cfg.Compare, mockClient, mockMgr, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, schema.Metric(1), result.FileStats["AGENTS.md"].Commits)

	mockMgr.AssertExpectations(t)
	mockStore.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_NoCacheManager(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}
	mockMgr := &MockCacheManager{}

	// Setup for aggregateActivity
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return(strings.Split(strings.TrimSpace(fileListFixture), "\n"), nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(gitLogBasicFixture, nil)

	// No cache manager
	mockMgr.On("GetActivityStore").Return(nil)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	result, err := CachedAggregateActivity(ctx, cfg.Git, cfg.Compare, mockClient, mockMgr, "")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, schema.Metric(1), result.FileStats["AGENTS.md"].Commits)

	mockMgr.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_AggregateError(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}
	mockMgr := &MockCacheManager{}
	mockStore := &MockCacheStore{}

	// Setup for error in aggregateActivity
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return(nil, assert.AnError)
	mockClient.On("GetRepoHash", ctx, "/test/repo").Return("abcd1234", nil)
	mockClient.On("GetRemoteURL", mock.Anything, mock.AnythingOfType("string")).Return("", nil).Maybe()
	mockClient.On("GetRootCommitHash", mock.Anything, mock.AnythingOfType("string")).Return("root123", nil).Maybe()
	mockClient.On("GetRepoRoot", mock.Anything, mock.AnythingOfType("string")).Return("/test/repo", nil).Maybe()

	// Cache miss
	mockMgr.On("GetActivityStore").Return(mockStore)
	mockStore.On("Get", mock.AnythingOfType("string")).Return([]byte{}, 0, int64(0), assert.AnError)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath:  "/test/repo",
			StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	result, err := CachedAggregateActivity(ctx, cfg.Git, cfg.Compare, mockClient, mockMgr, "")

	assert.Error(t, err)
	assert.Nil(t, result)

	mockMgr.AssertExpectations(t)
	mockStore.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}
