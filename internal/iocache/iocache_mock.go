package iocache

import (
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/stretchr/testify/mock"
)

// MockCacheManager is a mock implementation of CacheManager for testing.
type MockCacheManager struct {
	mock.Mock
}

var _ contract.CacheManager = &MockCacheManager{} // Compile-time check

// GetActivityStore implements the CacheManager interface.
func (m *MockCacheManager) GetActivityStore() contract.CacheStore {
	ret := m.Called()
	store, _ := ret.Get(0).(contract.CacheStore)
	return store
}

// MockCacheStore is a mock implementation of CacheStore for testing.
type MockCacheStore struct {
	mock.Mock
}

var _ contract.CacheStore = &MockCacheStore{} // Compile-time check

// Get implements the CacheStore interface.
func (m *MockCacheStore) Get(key string) ([]byte, int, int64, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Int(1), args.Get(2).(int64), args.Error(3)
}

// Set implements the CacheStore interface.
func (m *MockCacheStore) Set(key string, data []byte, version int, ts int64) error {
	args := m.Called(key, data, version, ts)
	return args.Error(0)
}

// Close implements the CacheStore interface.
func (m *MockCacheStore) Close() error {
	args := m.Called()
	return args.Error(0)
}
