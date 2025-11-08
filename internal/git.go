package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/mock"
)

// --- GitClient Interface Definition ---

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

// --- LocalGitClient Implementation ---

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
	if err != nil {
		return nil, fmt.Errorf("git command failed (%s): %w", strings.Join(fullArgs, " "), err)
	}
	return out, nil
}

// GetActivityLog implements the GitClient interface.
func (c *LocalGitClient) GetActivityLog(ctx context.Context, repoPath string, startTime, endTime time.Time) ([]byte, error) {
	args := []string{
		"log",
		"--numstat",
		"--pretty=format:'--%H|%an'",
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
		"--pretty=format:%ct",
		ref,
	}
	out, err := c.Run(ctx, repoPath, args...)
	if err != nil {
		return time.Time{}, err
	}
	timestampStr := strings.TrimSpace(string(out))
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse commit time '%s': %w", timestampStr, err)
	}
	return time.Unix(timestamp, 0), nil
}

// GetFileActivityLog implements the GitClient interface for fetching single-file metrics.
func (c *LocalGitClient) GetFileActivityLog(ctx context.Context, repoPath string, path string, startTime, endTime time.Time, follow bool) ([]byte, error) {
	args := []string{
		"log",
		"--pretty=format:DELIMITER_COMMIT_START%an,%ad",
		"--date=iso",
		"--numstat",
	}
	if follow {
		args = append(args, "--follow")
	}
	if !startTime.IsZero() {
		args = append(args, "--since="+startTime.Format(DateTimeFormat))
	}
	if !endTime.IsZero() {
		args = append(args, "--until="+endTime.Format(DateTimeFormat))
	}
	args = append(args, "--", path)
	return c.Run(ctx, repoPath, args...)
}

// GetRepoRoot implements the GitClient interface by executing 'git rev-parse --show-toplevel'.
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

// --- MockGitClient Implementation ---

// MockGitClient is an autogenerated mock type for the GitClient type.
type MockGitClient struct {
	mock.Mock
}

var _ GitClient = &MockGitClient{} // Compile-time check

// Run implements the core.GitClient interface.
func (m *MockGitClient) Run(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	var mockArgs []interface{}
	mockArgs = append(mockArgs, ctx, repoPath)
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}
	ret := m.Called(mockArgs...)
	output, _ := ret.Get(0).([]byte)
	return output, ret.Error(1)
}

// GetActivityLog implements the core.GitClient interface.
func (m *MockGitClient) GetActivityLog(ctx context.Context, repoPath string, startTime, endTime time.Time) ([]byte, error) {
	ret := m.Called(ctx, repoPath, startTime, endTime)
	output, _ := ret.Get(0).([]byte)
	return output, ret.Error(1)
}

// GetCommitTime implements the core.GitClient interface.
func (m *MockGitClient) GetCommitTime(ctx context.Context, repoPath string, ref string) (time.Time, error) {
	ret := m.Called(ctx, repoPath, ref)
	t, _ := ret.Get(0).(time.Time)
	return t, ret.Error(1)
}

// GetFileActivityLog implements the core.GitClient interface.
func (m *MockGitClient) GetFileActivityLog(ctx context.Context, repoPath string, path string, startTime, endTime time.Time, follow bool) ([]byte, error) {
	ret := m.Called(ctx, repoPath, path, startTime, endTime, follow)
	content, _ := ret.Get(0).([]byte)
	return content, ret.Error(1)
}

// GetRepoRoot implements the core.GitClient interface.
func (m *MockGitClient) GetRepoRoot(ctx context.Context, contextPath string) (string, error) {
	ret := m.Called(ctx, contextPath)
	root, _ := ret.Get(0).(string)
	return root, ret.Error(1)
}

// ListFilesAtRef implements the core.GitClient interface.
func (m *MockGitClient) ListFilesAtRef(ctx context.Context, repoPath string, ref string) ([]string, error) {
	ret := m.Called(ctx, repoPath, ref)
	files, _ := ret.Get(0).([]string)
	return files, ret.Error(1)
}
