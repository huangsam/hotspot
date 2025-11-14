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
