package internal

import (
	"github.com/stretchr/testify/mock"
)

// MockPersistenceManager is a mock implementation of PersistenceManager for testing.
type MockPersistenceManager struct {
	mock.Mock
}

var _ PersistenceManager = &MockPersistenceManager{} // Compile-time check

// GetActivityStore implements the PersistenceManager interface.
func (m *MockPersistenceManager) GetActivityStore() *PersistStore {
	ret := m.Called()
	store, _ := ret.Get(0).(*PersistStore)
	return store
}

// MockPersistenceStore is a mock implementation of PersistenceStore for testing.
type MockPersistenceStore struct {
	mock.Mock
}

var _ PersistenceStore = &MockPersistenceStore{} // Compile-time check

// Get implements the PersistenceStore interface.
func (m *MockPersistenceStore) Get(key string) ([]byte, int, int64, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Int(1), args.Get(2).(int64), args.Error(3)
}

// Set implements the PersistenceStore interface.
func (m *MockPersistenceStore) Set(key string, data []byte, version int, ts int64) error {
	args := m.Called(key, data, version, ts)
	return args.Error(0)
}
