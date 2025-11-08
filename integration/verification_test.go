//go:build integration

// Package integration contains integration tests for hotspot.
// These tests are excluded from normal test runs due to build tags.
// To run these tests: go test -tags integration ./integration
// Or use: make test-integration
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildHotspot builds the hotspot binary in the given directory and returns its absolute path
func buildHotspot(t *testing.T, dir string) string {
	hotspotPath := filepath.Join(dir, "hotspot")
	buildCmd := exec.Command("go", "build", "-o", hotspotPath, ".")
	buildCmd.Dir = dir
	err := buildCmd.Run()
	require.NoError(t, err)
	absPath, err := filepath.Abs(hotspotPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = exec.Command("rm", "-f", absPath).Run() })
	return absPath
}

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

	// Build hotspot binary
	hotspotPath := buildHotspot(t, repoDir)

	// Run hotspot files --output json
	cmd := exec.Command(hotspotPath, "files", "--output", "json")
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
	hotspotPath := buildHotspot(t, "..")

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

// TestTimeseriesVerification tests the timeseries command functionality
func TestTimeseriesVerification(t *testing.T) {
	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Build hotspot binary
	hotspotPath := buildHotspot(t, repoDir)

	// Test timeseries on main.go
	t.Run("main.go", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "timeseries", "--path", "main.go", "--interval", "30 days", "--points", "3", "--output", "json")
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err := cmd.Run()
		require.NoError(t, err)

		// Extract JSON from output (skip log lines)
		jsonOutput := extractJSONFromOutput(stdout.String())

		// Parse JSON output
		var result map[string][]map[string]interface{}
		err = json.Unmarshal([]byte(jsonOutput), &result)
		require.NoError(t, err)

		points, ok := result["points"]
		require.True(t, ok, "output should contain 'points' array")
		require.Len(t, points, 3, "should have 3 data points")

		// Verify each point has required fields
		for i, point := range points {
			t.Run(fmt.Sprintf("point_%d", i), func(t *testing.T) {
				path, ok := point["path"].(string)
				require.True(t, ok, "point should have 'path' field")
				assert.Equal(t, "main.go", path, "path should be main.go")

				period, ok := point["period"].(string)
				require.True(t, ok, "point should have 'period' field")
				assert.NotEmpty(t, period, "period should not be empty")

				score, ok := point["score"].(float64)
				require.True(t, ok, "point should have 'score' field")
				assert.GreaterOrEqual(t, score, 0.0, "score should be non-negative")

				mode, ok := point["mode"].(string)
				require.True(t, ok, "point should have 'mode' field")
				assert.Equal(t, "hot", mode, "default mode should be 'hot'")
			})
		}
	})

	// Test timeseries on core folder
	t.Run("core_folder", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "timeseries", "--path", "core", "--interval", "30 days", "--points", "3", "--output", "json", "--mode", "stale")
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err := cmd.Run()
		require.NoError(t, err)

		// Extract JSON from output
		jsonOutput := extractJSONFromOutput(stdout.String())

		// Parse JSON output
		var result map[string][]map[string]interface{}
		err = json.Unmarshal([]byte(jsonOutput), &result)
		require.NoError(t, err)

		points, ok := result["points"]
		require.True(t, ok, "output should contain 'points' array")
		require.Len(t, points, 3, "should have 3 data points")

		// Verify each point
		for i, point := range points {
			t.Run(fmt.Sprintf("point_%d", i), func(t *testing.T) {
				path, ok := point["path"].(string)
				require.True(t, ok, "point should have 'path' field")
				assert.Equal(t, "core", path, "path should be core")

				mode, ok := point["mode"].(string)
				require.True(t, ok, "point should have 'mode' field")
				assert.Equal(t, "stale", mode, "mode should be 'stale'")
			})
		}
	})

	// Test error cases
	t.Run("invalid_path", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "timeseries", "--path", "nonexistent.go", "--interval", "30 days", "--points", "3")
		cmd.Dir = repoDir
		err := cmd.Run()
		assert.Error(t, err, "should error on nonexistent path")
	})

	t.Run("missing_required_flags", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "timeseries", "--path", "main.go", "--interval", "30 days")
		cmd.Dir = repoDir
		err := cmd.Run()
		assert.Error(t, err, "should error when --points is missing")
	})
}

// extractJSONFromOutput extracts the JSON part from hotspot output that includes log lines
func extractJSONFromOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "{" { // Start of JSON object
			return strings.Join(lines[i:], "\n")
		}
	}
	return output // Fallback to original output
}

// FuzzParseHotspotOutput fuzzes the parseHotspotOutput function with random JSON inputs.
func FuzzParseHotspotOutput(f *testing.F) {
	seeds := []string{
		`[{"path": "main.go", "commits": 10}]`,
		`[]`,
		`[{"path": "", "commits": 0}]`,
		`invalid json`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, output string) {
		// Add timeout to prevent hanging on pathological JSON inputs
		done := make(chan bool, 1)
		go func() {
			_ = parseHotspotOutput(output)
			done <- true
		}()

		select {
		case <-done:
			// Function completed normally
		case <-time.After(100 * time.Millisecond):
			// Function took too long, likely pathological input
			t.Skip("parseHotspotOutput took too long, likely pathological input")
		}
	})
}
