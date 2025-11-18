package agg

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
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

func (m *MockCacheManager) GetActivityStore() contract.CacheStore {
	ret := m.Called()
	val := ret.Get(0)
	if val == nil {
		return nil
	}
	return val.(contract.CacheStore)
}

func (m *MockCacheManager) GetAnalysisStore() contract.AnalysisStore {
	ret := m.Called()
	val := ret.Get(0)
	if val == nil {
		return nil
	}
	return val.(contract.AnalysisStore)
}

func TestCheckCacheHit_CacheHit(t *testing.T) {
	mockStore := &MockCacheStore{}
	result := &schema.AggregateOutput{
		CommitMap: map[string]int{"test.go": 5},
	}
	data, _ := json.Marshal(result)

	// Valid cache entry: current version, recent timestamp
	mockStore.On("Get", "test-key").Return(data, currentCacheVersion, time.Now().Unix(), nil)

	actual := checkCacheHit(mockStore, "test-key")
	assert.NotNil(t, actual)
	assert.Equal(t, 5, actual.CommitMap["test.go"])
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
	mockClient := &contract.MockGitClient{}
	cfg := &contract.Config{
		RepoPath: "/test/repo",
		Mode:     schema.HotMode,
		Lookback: 30 * 24 * time.Hour, // 30 days
	}
	cfg.StartTime = time.Unix(1000000, 0)
	cfg.EndTime = time.Unix(2000000, 0)

	// Mock GetRepoHash for any repo path
	mockClient.On("GetRepoHash", mock.Anything, mock.AnythingOfType("string")).Return("abcd1234", nil)

	key1 := generateCacheKey(context.Background(), cfg, mockClient)

	// Key should be a non-empty SHA256 hash
	assert.NotEmpty(t, key1)
	assert.Len(t, key1, 64) // SHA256 hash length

	// Different config should produce different key
	cfg2 := *cfg
	cfg2.RepoPath = "/different/repo"
	key2 := generateCacheKey(context.Background(), &cfg2, mockClient)
	assert.NotEqual(t, key1, key2)

	mockClient.AssertExpectations(t)
}

func TestGenerateCacheKey_RepoHashError(t *testing.T) {
	mockClient := &contract.MockGitClient{}
	cfg := &contract.Config{
		RepoPath: "/test/repo",
		Mode:     schema.HotMode,
		Lookback: 30 * 24 * time.Hour,
	}
	cfg.StartTime = time.Unix(1000000, 0)
	cfg.EndTime = time.Unix(2000000, 0)

	// Mock GetRepoHash to return error
	mockClient.On("GetRepoHash", mock.Anything, mock.AnythingOfType("string")).Return("", assert.AnError)

	key := generateCacheKey(context.Background(), cfg, mockClient)

	// Key should still be generated (with empty repoHash)
	assert.NotEmpty(t, key)
	assert.Len(t, key, 64) // SHA256 hash length

	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &MockCacheManager{}
	mockStore := &MockCacheStore{}

	// Expected cached result
	expected := &schema.AggregateOutput{
		CommitMap: map[string]int{"test.go": 5},
	}

	data, _ := json.Marshal(expected)

	// Setup mocks
	mockMgr.On("GetActivityStore").Return(mockStore)
	mockStore.On("Get", mock.AnythingOfType("string")).Return(data, currentCacheVersion, time.Now().Unix(), nil)
	mockClient.On("GetRepoHash", ctx, "/test/repo").Return("abcd1234", nil)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Unix(1000000, 0),
		EndTime:   time.Unix(2000000, 0),
	}

	result, err := CachedAggregateActivity(ctx, cfg, mockClient, mockMgr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expected, result)

	mockMgr.AssertExpectations(t)
	mockStore.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_CacheMiss(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &MockCacheManager{}
	mockStore := &MockCacheStore{}

	// Setup for aggregateActivity
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return(strings.Split(strings.TrimSpace(fileListFixture), "\n"), nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(gitLogBasicFixture, nil)
	mockClient.On("GetRepoHash", ctx, "/test/repo").Return("abcd1234", nil)

	// Cache miss
	mockMgr.On("GetActivityStore").Return(mockStore)
	mockStore.On("Get", mock.AnythingOfType("string")).Return([]byte{}, 0, int64(0), assert.AnError)
	mockStore.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), currentCacheVersion, mock.AnythingOfType("int64")).Return(nil)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	result, err := CachedAggregateActivity(ctx, cfg, mockClient, mockMgr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.CommitMap["AGENTS.md"])

	mockMgr.AssertExpectations(t)
	mockStore.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_NoCacheManager(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &MockCacheManager{}

	// Setup for aggregateActivity
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return(strings.Split(strings.TrimSpace(fileListFixture), "\n"), nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(gitLogBasicFixture, nil)

	// No cache manager
	mockMgr.On("GetActivityStore").Return(nil)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	result, err := CachedAggregateActivity(ctx, cfg, mockClient, mockMgr)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.CommitMap["AGENTS.md"])

	mockMgr.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestCachedAggregateActivity_AggregateError(t *testing.T) {
	ctx := context.Background()
	mockClient := &contract.MockGitClient{}
	mockMgr := &MockCacheManager{}
	mockStore := &MockCacheStore{}

	// Setup for error in aggregateActivity
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return(nil, assert.AnError)
	mockClient.On("GetRepoHash", ctx, "/test/repo").Return("abcd1234", nil)

	// Cache miss
	mockMgr.On("GetActivityStore").Return(mockStore)
	mockStore.On("Get", mock.AnythingOfType("string")).Return([]byte{}, 0, int64(0), assert.AnError)

	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	result, err := CachedAggregateActivity(ctx, cfg, mockClient, mockMgr)

	assert.Error(t, err)
	assert.Nil(t, result)

	mockMgr.AssertExpectations(t)
	mockStore.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}
