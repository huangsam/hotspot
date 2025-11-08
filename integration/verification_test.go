//go:build integration

// Package integration contains integration tests for hotspot.
// These tests are excluded from normal test runs due to build tags.
// To run these tests: go test -tags integration ./integration
// Or use: make test-integration
package integration

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HotspotFile represents a file entry in hotspot JSON output
type HotspotFile struct {
	Path    string `json:"path"`
	Commits int    `json:"commits"`
}

// TestHotspotFilesVerification runs hotspot files --detail and verifies commit counts against git log
func TestHotspotFilesVerification(t *testing.T) {
	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Run hotspot files --output json
	cmd := exec.Command("./hotspot", "files", "--output", "json")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	require.NoError(t, err)

	// Parse output to extract file -> commits map
	fileCommits := parseHotspotOutput(stdout.String())

	// For each file, verify against git log --oneline -- <file>
	for file, hotspotCommits := range fileCommits {
		t.Run(file, func(t *testing.T) {
			gitCmd := exec.Command("git", "log", "--oneline", "--", file)
			gitCmd.Dir = repoDir
			gitOutput, err := gitCmd.Output()
			if err != nil {
				// File might not exist or have commits, skip
				t.Skipf("git log failed for %s: %v", file, err)
			}
			gitLines := strings.Split(strings.TrimSpace(string(gitOutput)), "\n")
			if gitLines[0] == "" {
				gitLines = []string{}
			}
			gitCommits := len(gitLines)

			assert.Equal(t, hotspotCommits, gitCommits,
				"commit count mismatch for %s", file)
		})
	}
}

// parseHotspotOutput extracts file paths and commit counts from hotspot JSON output
func parseHotspotOutput(output string) map[string]int {
	var files []HotspotFile
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "[" { // Start of JSON array
			jsonPart := strings.Join(lines[i:], "\n")
			if json.Unmarshal([]byte(jsonPart), &files) == nil {
				break
			}
		}
	}

	fileCommits := make(map[string]int)
	for _, file := range files {
		fileCommits[file.Path] = file.Commits
	}

	return fileCommits
}

// TestExternalRepoVerification clones multiple small public repos and runs verification
func TestExternalRepoVerification(t *testing.T) {
	// Test repos with different characteristics for better coverage
	testRepos := []struct {
		url  string
		name string
	}{
		{"https://github.com/mitchellh/go-homedir", "go-homedir"},          // Small Go library
		{"https://github.com/go-yaml/yaml", "go-yaml"},                     // Medium Go library with CGO
		{"https://github.com/urfave/cli", "urfave-cli"},                    // Popular Go CLI library
		{"https://github.com/huangsam/ultimate-python", "ultimate-python"}, // Medium Python repo
	}

	// Build hotspot binary once
	hotspotPath, err := filepath.Abs("test-repos/hotspot")
	require.NoError(t, err)
	buildCmd := exec.Command("go", "build", "-o", hotspotPath, ".")
	buildCmd.Dir = ".." // Project root
	err = buildCmd.Run()
	require.NoError(t, err)
	defer func() { _ = exec.Command("rm", "-f", hotspotPath).Run() }()

	for _, repo := range testRepos {
		t.Run(repo.name, func(t *testing.T) {
			testRepoDir := "test-repos/" + repo.name

			// Clean up any existing dir
			_ = exec.Command("rm", "-rf", testRepoDir).Run()

			// Clone the repo
			cloneCmd := exec.Command("git", "clone", "--depth=1", repo.url, testRepoDir)
			err := cloneCmd.Run()
			if err != nil {
				t.Skipf("failed to clone test repo %s: %v", repo.name, err)
			}
			defer func() { _ = exec.Command("rm", "-rf", testRepoDir).Run() }() // Clean up

			// Run verification in the test repo
			verifyRepo(t, testRepoDir, hotspotPath)
		})
	}
}

// verifyRepo runs hotspot and verifies against git for a given repo
func verifyRepo(t *testing.T, repoDir, hotspotPath string) {
	// Run hotspot files --output json --start 2000-01-01T00:00:00Z
	cmd := exec.Command(hotspotPath, "files", "--output", "json", "--start", "2000-01-01T00:00:00Z")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	require.NoError(t, err)

	// Parse output
	fileCommits := parseHotspotOutput(stdout.String())

	// Verify each file
	for file, hotspotCommits := range fileCommits {
		t.Run(file, func(t *testing.T) {
			gitCmd := exec.Command("git", "log", "--oneline", "--since", "2000-01-01T00:00:00Z", "--", file)
			gitCmd.Dir = repoDir
			gitOutput, err := gitCmd.Output()
			if err != nil {
				t.Skipf("git log failed for %s: %v", file, err)
			}
			gitLines := strings.Split(strings.TrimSpace(string(gitOutput)), "\n")
			if gitLines[0] == "" {
				gitLines = []string{}
			}
			gitCommits := len(gitLines)

			assert.Equal(t, hotspotCommits, gitCommits,
				"commit count mismatch for %s", file)
		})
	}
}
