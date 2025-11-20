// Package contract provides interfaces and shared utilities for internal architecture.
package contract

import (
	"context"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// GitClient defines the necessary operations for complex Git analysis.
// This allows the core analysis logic to be tested without needing a real git executable.
type GitClient interface {
	// --- Generic / Low-Level ---

	// Run executes a git command and returns the combined output.
	// Its use should be minimized in favor of the explicit methods below.
	Run(ctx context.Context, repoPath string, args ...string) ([]byte, error)

	// --- Time / Reference Resolution ---

	// GetCommitTime returns the time of the specified Git reference (e.g., commit hash, tag, branch name).
	GetCommitTime(ctx context.Context, repoPath string, ref string) (time.Time, error)

	// GetRepoHash returns the current HEAD commit hash of the repository.
	GetRepoHash(ctx context.Context, repoPath string) (string, error)

	// GetRepoRoot returns the absolute path to the root of the Git repository
	// containing the given context path.
	GetRepoRoot(ctx context.Context, contextPath string) (string, error)

	// --- Activity / Churn Logs ---

	// GetActivityLog returns the raw commit log output needed for repository-wide aggregation.
	GetActivityLog(ctx context.Context, repoPath string, startTime, endTime time.Time) ([]byte, error)

	// GetFileActivityLog returns the raw commit log output for a specific file path (supports --follow).
	GetFileActivityLog(ctx context.Context, repoPath string, path string, startTime, endTime time.Time, follow bool) ([]byte, error)

	// --- File State / Content ---

	// ListFilesAtRef returns a list of all trackable files in the repository at a specific reference.
	ListFilesAtRef(ctx context.Context, repoPath string, ref string) ([]string, error)

	// GetOldestCommitDateForPath retrieves the commit date of the Nth oldest commit for a path.
	GetOldestCommitDateForPath(ctx context.Context, repoPath string, path string, before time.Time, numCommits int, maxSearchDuration time.Duration) (time.Time, error)
}

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
	BeginAnalysis(startTime time.Time, configParams map[string]any) (int64, error)

	// EndAnalysis updates the analysis run with completion data
	EndAnalysis(analysisID int64, endTime time.Time, totalFiles int) error

	// RecordFileMetrics stores raw git metrics for a file
	RecordFileMetrics(analysisID int64, filePath string, metrics schema.FileMetrics) error

	// RecordFileScores stores final scores for a file
	RecordFileScores(analysisID int64, filePath string, scores schema.FileScores) error

	// GetStatus returns status information about the analysis store
	GetStatus() (schema.AnalysisStatus, error)

	// Close closes the underlying connection
	Close() error
}
