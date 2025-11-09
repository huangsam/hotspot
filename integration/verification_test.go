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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
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

// HotspotFileDetail represents a file entry with detailed metadata from hotspot JSON output
// Also works for basic output (extra fields are ignored during unmarshaling)
type HotspotFileDetail struct {
	Path               string  `json:"path"`
	Score              float64 `json:"score"`
	UniqueContributors int     `json:"unique_contributors"`
	Commits            int     `json:"commits"`
	LinesOfCode        int     `json:"lines_of_code"`
	Churn              int     `json:"churn"`
	AgeDays            int     `json:"age_days"`
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
	fileDetails := parseHotspotDetailOutput(stdout.String())
	fileCommits := make(map[string]int)
	for _, detail := range fileDetails {
		fileCommits[detail.Path] = detail.Commits
	}

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

// TestHotspotFilesAgeVerification runs hotspot files --detail with explicit time filters
// and verifies age calculations against the first commit within that time range
func TestHotspotFilesAgeVerification(t *testing.T) {
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

	// Use a fixed time range for consistent testing (last 365 days)
	startTime := time.Now().AddDate(0, 0, -365).Format(internal.DateTimeFormat)
	endTime := time.Now().Format(internal.DateTimeFormat)

	// Run hotspot files --output json --detail --start <start> --end <end>
	cmd := exec.Command(hotspotPath, "files", "--output", "json", "--detail", "--start", startTime, "--end", endTime)
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	require.NoError(t, err)

	// Parse output to extract file details
	fileDetails := parseHotspotDetailOutput(stdout.String())

	// For each file, verify age calculation against the first commit within the time range
	for file, details := range fileDetails {
		t.Run(file, func(t *testing.T) {
			// Get the first commit timestamp for this file within the same time range as hotspot
			gitCmd := exec.Command("git", "log", "--pretty=format:%ct", "--since", startTime, "--until", endTime, "--", file)
			gitCmd.Dir = repoDir
			gitOutput, err := gitCmd.Output()
			if err != nil {
				// File might not exist or have commits in range, skip
				t.Skipf("git log failed for %s: %v", file, err)
			}

			lines := strings.Split(strings.TrimSpace(string(gitOutput)), "\n")
			if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
				t.Skipf("no commits found for %s in time range", file)
			}

			// Parse the first commit timestamp (oldest in the range due to reverse chronological order)
			firstCommitTimestampStr := strings.TrimSpace(lines[len(lines)-1]) // Last line is oldest
			firstCommitTimestamp, err := strconv.ParseInt(firstCommitTimestampStr, 10, 64)
			require.NoError(t, err, "failed to parse git timestamp")

			firstCommitTime := time.Unix(firstCommitTimestamp, 0)
			expectedAgeDays := int(time.Since(firstCommitTime).Hours() / 24)

			// Age should match exactly since we're using the same time range
			assert.Equal(t, expectedAgeDays, details.AgeDays,
				"age calculation should match first commit in analysis window for %s (got %d, expected %d)",
				file, details.AgeDays, expectedAgeDays)
		})
	}
}

// parseHotspotDetailOutput extracts file details from hotspot JSON output
// Works with both basic and detailed output formats
func parseHotspotDetailOutput(output string) map[string]HotspotFileDetail {
	var files []HotspotFileDetail
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "[" { // Start of JSON array
			jsonPart := strings.Join(lines[i:], "\n")
			if json.Unmarshal([]byte(jsonPart), &files) == nil {
				break
			}
		}
	}

	fileDetails := make(map[string]HotspotFileDetail)
	for _, file := range files {
		fileDetails[file.Path] = file
	}

	return fileDetails
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
	fileDetails := parseHotspotDetailOutput(stdout.String())
	fileCommits := make(map[string]int)
	for _, detail := range fileDetails {
		fileCommits[detail.Path] = detail.Commits
	}

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
		var result map[string][]map[string]any
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
				assert.Equal(t, schema.HotMode, schema.ScoringMode(mode), "default mode should be 'hot'")
			})
		}
	})

	// Test timeseries on core folder
	t.Run("core_folder", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "timeseries", "--path", "core", "--interval", "30 days", "--points", "3", "--output", "json", "--mode", string(schema.StaleMode))
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err := cmd.Run()
		require.NoError(t, err)

		// Extract JSON from output
		jsonOutput := extractJSONFromOutput(stdout.String())

		// Parse JSON output
		var result map[string][]map[string]any
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
				assert.Equal(t, schema.StaleMode, schema.ScoringMode(mode), "mode should be 'stale'")
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
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") { // Start of JSON object or array
			return strings.Join(lines[i:], "\n")
		}
	}
	return output // Fallback to original output
}

// TestCustomWeightsIntegration tests the full CLI with custom weights configuration
func TestCustomWeightsIntegration(t *testing.T) {
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

	// Create a temporary config file with custom weights
	configContent := `
mode: hot
limit: 5
weights:
  hot:
    commits: 0.8
    churn: 0.1
    age: 0.05
    contrib: 0.04
    size: 0.01
`
	configFile := filepath.Join(repoDir, ".hotspot.yml")
	err = os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(configFile) })

	// Run hotspot files with custom config
	cmd := exec.Command(hotspotPath, "files", "--output", "json")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	require.NoError(t, err)

	// Parse output
	var files []HotspotFileDetail
	jsonPart := extractJSONFromOutput(stdout.String())
	err = json.Unmarshal([]byte(jsonPart), &files)
	require.NoError(t, err)

	// Should have some results
	assert.NotEmpty(t, files, "Should have file results")

	// Test with different custom weights for comparison
	configContent2 := `
mode: hot
limit: 5
weights:
  hot:
    commits: 0.1
    churn: 0.8
    age: 0.05
    contrib: 0.04
    size: 0.01
`
	err = os.WriteFile(configFile, []byte(configContent2), 0o644)
	require.NoError(t, err)

	// Run hotspot files with different custom config
	cmd2 := exec.Command(hotspotPath, "files", "--output", "json")
	cmd2.Dir = repoDir
	var stdout2 bytes.Buffer
	cmd2.Stdout = &stdout2
	err = cmd2.Run()
	require.NoError(t, err)

	// Parse second output
	var files2 []HotspotFileDetail
	jsonPart2 := extractJSONFromOutput(stdout2.String())
	err = json.Unmarshal([]byte(jsonPart2), &files2)
	require.NoError(t, err)

	// Results should be different due to different weights
	// (Note: we can't guarantee the scores will be different for all files,
	// but the configuration should be accepted and processed)
	assert.NotEmpty(t, files2, "Should have file results with different weights")
}

// TestCustomWeightsValidationIntegration tests that invalid weights are rejected
func TestCustomWeightsValidationIntegration(t *testing.T) {
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

	// Create a temporary config file with invalid weights (don't sum to 1.0)
	configContent := `
mode: hot
limit: 5
weights:
  hot:
    commits: 0.5
    churn: 0.3
    age: 0.3  # 0.5 + 0.3 + 0.3 = 1.1
`
	configFile := filepath.Join(repoDir, ".hotspot.yml")
	err = os.WriteFile(configFile, []byte(configContent), 0o644)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(configFile) })

	// Run hotspot files - should fail due to invalid weights
	cmd := exec.Command(hotspotPath, "files")
	cmd.Dir = repoDir
	err = cmd.Run()
	assert.Error(t, err, "Should fail with invalid weights that don't sum to 1.0")
}
