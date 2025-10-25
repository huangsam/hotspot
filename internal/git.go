package internal

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunGitCommand is a common helper to execute a git command and return
// its combined stdout/stderr output. It provides rich error context.
func RunGitCommand(repoPath string, args ...string) ([]byte, error) {
	// Prefix the arguments with the -C flag to specify the repository path
	// This makes the git command execution consistent.
	fullArgs := append([]string{"-C", repoPath}, args...)

	cmd := exec.Command("git", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		// 1. Try to cast the error to *exec.ExitError to get the stderr
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.Stderr != nil {
			// Extract and clean up the actual error message from Git's stderr
			errMsg := strings.TrimSpace(string(exitErr.Stderr))

			// Wrap the error with context and the Git's stderr output
			return nil, fmt.Errorf("git command '%s' failed: %s: %w", strings.Join(fullArgs, " "), errMsg, err)
		}

		// 2. Handle cases where the command failed to start (e.g., 'git' not found)
		return nil, fmt.Errorf("could not execute git command (is git installed and in PATH?): %w", err)
	}

	return out, nil
}
