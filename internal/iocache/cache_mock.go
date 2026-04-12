package iocache

import (
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/mock"
)

// MockCacheManager is a mock implementation of CacheManager for testing.
type MockCacheManager struct {
	mock.Mock
}

var _ CacheManager = &MockCacheManager{} // Compile-time check

// GetActivityStore implements the CacheManager interface.
func (m *MockCacheManager) GetActivityStore() CacheStore {
	ret := m.Called()
	store, _ := ret.Get(0).(CacheStore)
	return store
}

// GetAnalysisStore implements the CacheManager interface.
func (m *MockCacheManager) GetAnalysisStore() AnalysisStore {
	ret := m.Called()
	store, _ := ret.Get(0).(AnalysisStore)
	return store
}

// MockCacheStore is a mock implementation of CacheStore for testing.
type MockCacheStore struct {
	mock.Mock
}

var _ CacheStore = &MockCacheStore{} // Compile-time check

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

// GetStatus implements the CacheStore interface.
func (m *MockCacheStore) GetStatus() (schema.CacheStatus, error) {
	args := m.Called()
	return args.Get(0).(schema.CacheStatus), args.Error(1)
}

// MockAnalysisStore is a mock implementation of AnalysisStore for testing.
type MockAnalysisStore struct {
	mock.Mock
}

var _ AnalysisStore = &MockAnalysisStore{} // Compile-time check

// BeginAnalysis implements the AnalysisStore interface.
func (m *MockAnalysisStore) BeginAnalysis(urn string, startTime time.Time, configParams map[string]any) (int64, error) {
	args := m.Called(urn, startTime, configParams)
	return args.Get(0).(int64), args.Error(1)
}

// EndAnalysis implements the AnalysisStore interface.
func (m *MockAnalysisStore) EndAnalysis(analysisID int64, endTime time.Time, totalFiles int) error {
	args := m.Called(analysisID, endTime, totalFiles)
	return args.Error(0)
}

// RecordFileMetricsAndScores implements the AnalysisStore interface.
func (m *MockAnalysisStore) RecordFileMetricsAndScores(analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error {
	args := m.Called(analysisID, filePath, metrics, scores)
	return args.Error(0)
}

// Close implements the AnalysisStore interface.
func (m *MockAnalysisStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

// GetStatus implements the AnalysisStore interface.
func (m *MockAnalysisStore) GetStatus() (schema.AnalysisStatus, error) {
	args := m.Called()
	return args.Get(0).(schema.AnalysisStatus), args.Error(1)
}

// GetAllAnalysisRuns implements the AnalysisStore interface.
func (m *MockAnalysisStore) GetAllAnalysisRuns() ([]schema.AnalysisRunRecord, error) {
	args := m.Called()
	return args.Get(0).([]schema.AnalysisRunRecord), args.Error(1)
}

// GetAnalysisRuns implements the AnalysisStore interface.
func (m *MockAnalysisStore) GetAnalysisRuns(filter schema.AnalysisQueryFilter) ([]schema.AnalysisRunRecord, error) {
	args := m.Called(filter)
	return args.Get(0).([]schema.AnalysisRunRecord), args.Error(1)
}

// GetAllFileScoresMetrics implements the AnalysisStore interface.
func (m *MockAnalysisStore) GetAllFileScoresMetrics() ([]schema.FileScoresMetricsRecord, error) {
	args := m.Called()
	return args.Get(0).([]schema.FileScoresMetricsRecord), args.Error(1)
}

// GetFileScoresMetrics implements the AnalysisStore interface.
func (m *MockAnalysisStore) GetFileScoresMetrics(filter schema.AnalysisQueryFilter) ([]schema.FileScoresMetricsRecord, error) {
	args := m.Called(filter)
	return args.Get(0).([]schema.FileScoresMetricsRecord), args.Error(1)
}
