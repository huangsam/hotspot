// Package git provides Git client implementations and the core GitClient interface.
package git

import (
	"context"
	"time"
)

// Client defines the necessary operations for complex Git analysis.
// This allows the core analysis logic to be tested without needing a real git executable.
type Client interface {
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
	// If path is non-empty, the log is restricted to that subdirectory or file.
	GetActivityLog(ctx context.Context, repoPath string, path string, startTime, endTime time.Time) ([]byte, error)

	// GetFileActivityLog returns the raw commit log output for a specific file path (supports --follow).
	GetFileActivityLog(ctx context.Context, repoPath string, path string, startTime, endTime time.Time, follow bool) ([]byte, error)

	// --- File State / Content ---

	// ListFilesAtRef returns a list of all trackable files in the repository at a specific reference.
	ListFilesAtRef(ctx context.Context, repoPath string, ref string) ([]string, error)

	// GetChangedFilesBetweenRefs returns a list of files that changed between two Git references.
	GetChangedFilesBetweenRefs(ctx context.Context, repoPath string, baseRef string, targetRef string) ([]string, error)

	// GetOldestCommitDateForPath retrieves the commit date of the Nth oldest commit for a path.
	GetOldestCommitDateForPath(ctx context.Context, repoPath string, path string, before time.Time, numCommits int, maxSearchDuration time.Duration) (time.Time, error)

	// GetRootCommitHash returns the hash of the oldest/initial commit in the repository.
	GetRootCommitHash(ctx context.Context, repoPath string) (string, error)

	// GetRemoteURL returns the URL of the 'origin' remote for the repository.
	GetRemoteURL(ctx context.Context, repoPath string) (string, error)

	// GetTags returns a list of version tags sorted by descending semantic version order.
	// limit controls the maximum number of tags returned (0 = no limit).
	GetTags(ctx context.Context, repoPath string, limit int) ([]string, error)
}

// ResolveURN returns a canonical repository identifier.
// It prioritizes the remote 'origin' URL but falls back to the root commit hash or absolute local path.
func ResolveURN(ctx context.Context, client Client, repoPath string) string {
	if url, err := client.GetRemoteURL(ctx, repoPath); err == nil && url != "" {
		return "git:" + url
	}

	// Fallback to the root commit hash of the local repository.
	// This provides a stable identifier even if the repo directory is moved.
	if rootHash, err := client.GetRootCommitHash(ctx, repoPath); err == nil && rootHash != "" {
		return "local:" + rootHash
	}

	// Last resort fallback: local path if no remote origin and no commits exist yet
	absPath, _ := client.GetRepoRoot(ctx, repoPath)
	if absPath == "" {
		absPath = repoPath
	}
	return "local:" + absPath
}
