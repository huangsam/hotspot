// Package iocache is for caching I/O calls.
package iocache

import (
	"sync"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// CacheManager defines the interface for managing cache stores.
// This allows the cache layer to be mocked for testing.
type CacheManager interface {
	GetActivityStore() CacheStore
	GetAnalysisStore() AnalysisStore
}

// CacheStore defines the interface for cache data storage.
// This allows mocking the store for testing.
type CacheStore interface {
	Get(key string) ([]byte, int, int64, error)
	Set(key string, value []byte, version int, timestamp int64) error
	GetStatus() (schema.CacheStatus, error)
	Close() error
}

// AnalysisStore defines the interface for tracking analysis runs and storing metrics.
type AnalysisStore interface {
	// BeginAnalysis creates a new analysis run and returns its unique ID
	BeginAnalysis(urn string, startTime time.Time, configParams map[string]any) (int64, error)

	// EndAnalysis updates the analysis run with completion data
	EndAnalysis(analysisID int64, endTime time.Time, totalFiles int) error

	// RecordFileMetricsAndScores stores both raw git metrics and final scores for a file in one operation
	RecordFileMetricsAndScores(analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error

	// RecordFileResultsBatch stores multiple file metrics and scores in a single batch operation
	RecordFileResultsBatch(analysisID int64, results []schema.BatchFileResult) error

	// GetAllAnalysisRuns retrieves all analysis runs from the store
	GetAllAnalysisRuns() ([]schema.AnalysisRunRecord, error)

	// GetAnalysisRuns retrieves analysis runs with optional filtering and pagination
	GetAnalysisRuns(filter schema.AnalysisQueryFilter) ([]schema.AnalysisRunRecord, error)

	// GetAllFileScoresMetrics retrieves all file scores and metrics from the store
	GetAllFileScoresMetrics() ([]schema.FileScoresMetricsRecord, error)

	// GetFileScoresMetrics retrieves file scores and metrics with optional filtering and pagination
	GetFileScoresMetrics(filter schema.AnalysisQueryFilter) ([]schema.FileScoresMetricsRecord, error)

	// GetStatus returns status information about the analysis store
	GetStatus() (schema.AnalysisStatus, error)

	// PruneOrphanedRuns removes analysis runs that never completed (total_files_analyzed is NULL)
	// and are older than the specified duration.
	PruneOrphanedRuns(maxAge time.Duration) error

	// Close closes the underlying connection
	Close() error
}

// CacheStoreManager manages multiple CacheStore instances.
type CacheStoreManager struct {
	sync.RWMutex // Protects the store pointers during initialization
	activity     CacheStore
	analysis     AnalysisStore
}

var _ CacheManager = &CacheStoreManager{} // Compile-time check

// GetActivityStore returns the activity CacheStore.
func (mgr *CacheStoreManager) GetActivityStore() CacheStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.activity
}

// GetAnalysisStore returns the analysis AnalysisStore.
func (mgr *CacheStoreManager) GetAnalysisStore() AnalysisStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.analysis
}
