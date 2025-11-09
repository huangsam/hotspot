package internal

import (
	"context"
	"time"
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
}
