package agg

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCacheStore for testing (alias for MockCacheStore)
type MockCacheStore = iocache.MockCacheStore

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
