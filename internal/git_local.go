package internal

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
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
		return nil, fmt.Errorf("git '%v' exit: %s", strings.Join(fullArgs, " "), strings.TrimSpace(string(exitErr.Stderr)))
	} else if err != nil {
		return nil, fmt.Errorf("git '%v' unknown: %w", strings.Join(fullArgs, " "), err)
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
