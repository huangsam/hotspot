package contract

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// LocalGitClient implements the GitClient interface by executing the
// local 'git' binary installed on the machine.
type LocalGitClient struct{}

var _ GitClient = &LocalGitClient{} // Compile-time check

// NewLocalGitClient creates a new instance of the local Git client.
func NewLocalGitClient() *LocalGitClient {
	return &LocalGitClient{}
}

// Run executes a git command and returns its combined stdout/stderr output.
func (c *LocalGitClient) Run(_ context.Context, repoPath string, args ...string) ([]byte, error) {
	fullArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", fullArgs...)
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		stderr := strings.TrimSpace(string(exitErr.Stderr))
		return nil, fmt.Errorf("git command failed in %q: %s. If this is not a Git repository, verify the path or run 'git init'", repoPath, stderr)
	} else if err != nil {
		return nil, fmt.Errorf("git command failed: %w. Ensure Git is installed and available on your PATH", err)
	}
	return out, nil
}

// GetActivityLog implements the GitClient interface.
func (c *LocalGitClient) GetActivityLog(ctx context.Context, repoPath string, startTime, endTime time.Time) ([]byte, error) {
	args := []string{
		"log",
		"--numstat",
		"--pretty=format:'--%H|%an|%ad'",
		"--date=iso-strict",
	}
	if !startTime.IsZero() {
		args = append(args, fmt.Sprintf("--since=%s", startTime.Format(DateTimeFormat)))
	}
	if !endTime.IsZero() {
		args = append(args, fmt.Sprintf("--until=%s", endTime.Format(DateTimeFormat)))
	}
	return c.Run(ctx, repoPath, args...)
}

// GetCommitTime implements the GitClient interface.
func (c *LocalGitClient) GetCommitTime(ctx context.Context, repoPath string, ref string) (time.Time, error) {
	args := []string{
		"log", "-n", "1",
		"--pretty=format:%ad",
		"--date=iso-strict",
		ref,
	}
	out, err := c.Run(ctx, repoPath, args...)
	if err != nil {
		return time.Time{}, err
	}
	dateStr := strings.TrimSpace(string(out))
	return time.Parse(time.RFC3339, dateStr)
}

// GetRepoHash implements the GitClient interface.
func (c *LocalGitClient) GetRepoHash(ctx context.Context, repoPath string) (string, error) {
	out, err := c.Run(ctx, repoPath, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetFileActivityLog implements the GitClient interface.
func (c *LocalGitClient) GetFileActivityLog(ctx context.Context, repoPath string, path string, startTime, endTime time.Time, follow bool) ([]byte, error) {
	args := []string{
		"log",
		"--pretty=format:DELIMITER_COMMIT_START%an|%ad",
		"--date=iso-strict",
		"--numstat",
	}
	if follow {
		args = append(args, "--follow")
		// When using --follow, we want complete history, not time-filtered
	} else {
		// Only apply time filters when not using --follow
		if !startTime.IsZero() {
			args = append(args, "--since="+startTime.Format(DateTimeFormat))
		}
		if !endTime.IsZero() {
			args = append(args, "--until="+endTime.Format(DateTimeFormat))
		}
	}
	args = append(args, "--", path)
	return c.Run(ctx, repoPath, args...)
}

// GetRepoRoot implements the GitClient interface.
func (c *LocalGitClient) GetRepoRoot(ctx context.Context, contextPath string) (string, error) {
	out, err := c.Run(ctx, contextPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ListFilesAtRef implements the GitClient interface.
func (c *LocalGitClient) ListFilesAtRef(ctx context.Context, repoPath string, ref string) ([]string, error) {
	args := []string{
		"ls-tree", "-r", "--name-only",
		ref,
	}
	out, err := c.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

// GetChangedFilesBetweenRefs implements the GitClient interface.
// It returns files that have changed between baseRef and targetRef.
// Uses Git's ".." (two-dot) range syntax which shows commits reachable from
// targetRef but not from baseRef. This is appropriate for comparing branches
// that share a common ancestor (e.g., feature branch vs main).
func (c *LocalGitClient) GetChangedFilesBetweenRefs(ctx context.Context, repoPath string, baseRef string, targetRef string) ([]string, error) {
	args := []string{
		"diff", "--name-only",
		baseRef + ".." + targetRef,
	}
	out, err := c.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}
	return files, nil
}

// GetOldestCommitDateForPath implements the GitClient interface.
func (c *LocalGitClient) GetOldestCommitDateForPath(ctx context.Context, repoPath string, path string, before time.Time, numCommits int, maxSearchDuration time.Duration) (time.Time, error) {
	afterTime := before.Add(-maxSearchDuration)
	args := []string{
		"log",
		fmt.Sprintf("-n%d", numCommits),
		"--pretty=format:%ad",
		"--date=iso-strict",
		"--before=" + before.Format(time.RFC3339),
		"--after=" + afterTime.Format(time.RFC3339),
		"--",
		path,
	}
	out, err := c.Run(ctx, repoPath, args...)
	if err != nil {
		return time.Time{}, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return time.Time{}, errors.New("no commits found for path")
	}

	// The last line has the oldest commit's date
	oldestDateStr := lines[len(lines)-1]
	return time.Parse(time.RFC3339, oldestDateStr)
}
