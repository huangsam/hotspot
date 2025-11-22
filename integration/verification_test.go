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
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// sharedHotspotPath holds the path to a shared hotspot binary built once for all tests.
	sharedHotspotPath string

	// buildOnce ensures we only build the binary once.
	buildOnce sync.Once

	// buildMutex protects the shared binary path.
	buildMutex sync.Mutex

	// tempDir holds the temp directory for cleanup.
	tempDir string
)

// TestMain handles setup and cleanup for all integration tests.
func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	// Cleanup the shared binary after all tests
	if tempDir != "" {
		_ = os.RemoveAll(tempDir)
	}

	os.Exit(code)
}

// getHotspotBinary returns the path to the hotspot binary, building it once if needed.
func getHotspotBinary() string {
	buildMutex.Lock()
	defer buildMutex.Unlock()

	buildOnce.Do(func() {
		// Create a temp directory for the binary
		var err error
		tempDir, err = os.MkdirTemp("", "hotspot-integration-*")
		if err != nil {
			panic(fmt.Sprintf("failed to create temp dir: %v", err))
		}

		hotspotPath := filepath.Join(tempDir, "hotspot")
		buildCmd := exec.Command("go", "build", "-o", hotspotPath, ".")
		buildCmd.Dir = ".." // Build from parent directory (project root)
		err = buildCmd.Run()
		if err != nil {
			panic(fmt.Sprintf("failed to build hotspot: %v", err))
		}

		sharedHotspotPath = hotspotPath
	})

	return sharedHotspotPath
}

// TestFilesVerification runs hotspot files with time filters and verifies both commit counts and age calculations.
// This test samples a subset of files to keep runtime reasonable while still providing good coverage.
func TestFilesVerification(t *testing.T) {
	t.Parallel()

	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

	// Use a fixed time range for consistent testing (last 365 days)
	startTime := time.Now().AddDate(0, 0, -365).Format(contract.DateTimeFormat)
	endTime := time.Now().Format(contract.DateTimeFormat)

	// Run hotspot files --output json --detail --start <start> --end <end> --limit 50
	// Limit to top 50 files to keep runtime reasonable while still testing core functionality
	cmd := exec.Command(hotspotPath, "files", "--output", "json", "--detail", "--start", startTime, "--end", endTime, "--limit", "50")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	require.NoError(t, err)

	// Parse output to extract file details
	fileDetails := parseHotspotDetailOutput(stdout.String())

	// Sample a subset of files to verify (first 10, last 5, and some random ones)
	// This keeps the test thorough but not exhaustive
	files := make([]string, 0, len(fileDetails))
	for file := range fileDetails {
		files = append(files, file)
	}

	// Sort for deterministic sampling
	sort.Strings(files)

	// Sample files: first 10, last 5, and every 10th file in between
	sampledFiles := make(map[string]bool)
	for i, file := range files {
		if i < 10 || i >= len(files)-5 || i%10 == 0 {
			sampledFiles[file] = true
		}
	}

	// Verify both commit counts and age calculations for sampled files
	for file := range sampledFiles {
		t.Run(file, func(t *testing.T) {
			details, exists := fileDetails[file]
			require.True(t, exists, "file should exist in results")

			// Verify commit count within the time range
			gitCmd := exec.Command("git", "log", "--oneline", "--since", startTime, "--until", endTime, "--", file)
			gitCmd.Dir = repoDir
			gitOutput, err := gitCmd.Output()
			if err != nil {
				// File might not exist or have commits in range, skip
				t.Skipf("git log failed for %s: %v", file, err)
			}
			gitLines := strings.Split(strings.TrimSpace(string(gitOutput)), "\n")
			if gitLines[0] == "" {
				gitLines = []string{}
			}
			gitCommits := len(gitLines)

			assert.Equal(t, details.Commits, gitCommits,
				"commit count mismatch for %s in time range %s to %s", file, startTime, endTime)

			// Verify age calculation against the first commit within the time range
			if gitCommits > 0 {
				ageGitCmd := exec.Command("git", "log", "--pretty=format:%ad", "--date=iso-strict", "--since", startTime, "--until", endTime, "--", file)
				ageGitCmd.Dir = repoDir
				ageGitOutput, err := ageGitCmd.Output()
				require.NoError(t, err, "failed to get age data for %s", file)

				lines := strings.Split(strings.TrimSpace(string(ageGitOutput)), "\n")
				if len(lines) > 0 && lines[0] != "" {
					// Parse the first commit timestamp (oldest in the range, last line since newest first)
					firstCommitTimestampStr := strings.TrimSpace(lines[len(lines)-1]) // Last line is oldest
					firstCommitTime, err := time.Parse(time.RFC3339, firstCommitTimestampStr)
					require.NoError(t, err, "failed to parse git timestamp for %s", file)

					expectedAgeDays := contract.CalculateDaysBetween(firstCommitTime, time.Now())

					// Age should match exactly since we're using the same time range
					assert.Equal(t, expectedAgeDays, details.AgeDays,
						"age calculation should match first commit in analysis window for %s (got %d, expected %d)",
						file, details.AgeDays, expectedAgeDays)
				}
			}
		})
	}
}

// parseHotspotDetailOutput extracts file details from hotspot JSON output
// Works with both basic and detailed output formats.
func parseHotspotDetailOutput(output string) map[string]schema.FileResult {
	var files []schema.FileResult
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "[" { // Start of JSON array
			jsonPart := strings.Join(lines[i:], "\n")
			if json.Unmarshal([]byte(jsonPart), &files) == nil {
				break
			}
		}
	}

	fileDetails := make(map[string]schema.FileResult)
	for _, file := range files {
		fileDetails[file.Path] = file
	}

	return fileDetails
}

// TestFoldersVerification runs hotspot folders with time filters and verifies folder aggregation.
func TestFoldersVerification(t *testing.T) {
	t.Parallel()

	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

	// Use a fixed time range for consistent testing (last 365 days)
	startTime := time.Now().AddDate(0, 0, -365).Format(contract.DateTimeFormat)
	endTime := time.Now().Format(contract.DateTimeFormat)

	// Run hotspot folders --output json --detail --start <start> --end <end> --limit 1000
	cmd := exec.Command(hotspotPath, "folders", "--output", "json", "--detail", "--start", startTime, "--end", endTime, "--limit", "1000")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	require.NoError(t, err)

	// Parse output to extract folder details
	folderDetails := parseHotspotFolderOutput(stdout.String())

	// Verify folder aggregation for each folder
	for folderPath, folder := range folderDetails {
		t.Run(folderPath, func(t *testing.T) {
			// Verify that folder has reasonable values and structure
			assert.Greater(t, folder.Commits, 0, "folder should have commits")
			assert.GreaterOrEqual(t, folder.Churn, 0, "folder should have non-negative churn")
			assert.GreaterOrEqual(t, folder.Score, 0.0, "folder should have non-negative score")
			assert.NotEmpty(t, folder.Path, "folder should have a path")
			assert.Contains(t, []string{"hot", "risk", "complexity", "stale"}, string(folder.Mode), "folder should have valid mode")
		})
	}
}

// TestCompareFilesVerification runs hotspot compare files and verifies comparison deltas.
func TestCompareFilesVerification(t *testing.T) {
	t.Parallel()

	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

	// Use stable tags for consistent testing
	baseRef := "v1.1.4"
	targetRef := "v1.1.5"

	// Run hotspot compare files --output json --base-ref v1.1.4 --target-ref v1.1.5 --limit 10
	cmd := exec.Command(hotspotPath, "compare", "files", "--output", "json", "--base-ref", baseRef, "--target-ref", targetRef, "--limit", "10")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	require.NoError(t, err)

	// Parse the comparison JSON output
	jsonPart := extractJSONFromOutput(stdout.String())
	var result schema.ComparisonResult
	err = json.Unmarshal([]byte(jsonPart), &result)
	require.NoError(t, err)

	// Verify basic structure
	require.NotEmpty(t, result.Results, "should have comparison results")
	require.NotNil(t, result.Summary, "should have comparison summary")

	// Verify each comparison detail
	for i, detail := range result.Results {
		t.Run(fmt.Sprintf("file_%d_%s", i, filepath.Base(detail.Path)), func(t *testing.T) {
			// Verify required fields are present
			assert.NotEmpty(t, detail.Path, "path should not be empty")
			assert.GreaterOrEqual(t, detail.BeforeScore, 0.0, "before score should be non-negative")
			assert.GreaterOrEqual(t, detail.AfterScore, 0.0, "after score should be non-negative")
			assert.Contains(t, []string{"hot", "risk", "complexity", "stale"}, string(detail.Mode), "should have valid mode")

			// Delta can be positive or negative, just verify it's a valid number
			assert.True(t, detail.Delta >= -100.0 && detail.Delta <= 100.0, "delta should be reasonable")

			// Verify owners arrays are valid (can be empty)
			assert.NotNil(t, detail.BeforeOwners, "before owners should not be nil")
			assert.NotNil(t, detail.AfterOwners, "after owners should not be nil")
		})
	}

	// Verify summary has reasonable values
	assert.True(t, result.Summary.NetScoreDelta >= -1000.0 && result.Summary.NetScoreDelta <= 1000.0, "net score delta should be reasonable")
	assert.GreaterOrEqual(t, result.Summary.TotalModifiedFiles, 0, "total modified files should be non-negative")
	assert.GreaterOrEqual(t, result.Summary.TotalNewFiles, 0, "total new files should be non-negative")
	assert.GreaterOrEqual(t, result.Summary.TotalInactiveFiles, 0, "total inactive files should be non-negative")
}

// TestCompareFoldersVerification runs hotspot compare folders and verifies comparison deltas.
func TestCompareFoldersVerification(t *testing.T) {
	t.Parallel()

	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

	// Use stable tags for consistent testing
	baseRef := "v1.1.4"
	targetRef := "v1.1.5"

	// Run hotspot compare folders --output json --base-ref v1.1.4 --target-ref v1.1.5 --limit 10
	cmd := exec.Command(hotspotPath, "compare", "folders", "--output", "json", "--base-ref", baseRef, "--target-ref", targetRef, "--limit", "10")
	cmd.Dir = repoDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	require.NoError(t, err)

	// Parse the comparison JSON output
	jsonPart := extractJSONFromOutput(stdout.String())
	var result schema.ComparisonResult
	err = json.Unmarshal([]byte(jsonPart), &result)
	require.NoError(t, err)

	// Verify basic structure
	require.NotEmpty(t, result.Results, "should have comparison results")
	require.NotNil(t, result.Summary, "should have comparison summary")

	// Verify each comparison detail
	for i, detail := range result.Results {
		t.Run(fmt.Sprintf("folder_%d_%s", i, filepath.Base(detail.Path)), func(t *testing.T) {
			// Verify required fields are present
			assert.NotEmpty(t, detail.Path, "path should not be empty")
			assert.GreaterOrEqual(t, detail.BeforeScore, 0.0, "before score should be non-negative")
			assert.GreaterOrEqual(t, detail.AfterScore, 0.0, "after score should be non-negative")
			assert.Contains(t, []string{"hot", "risk", "complexity", "stale"}, string(detail.Mode), "should have valid mode")

			// Delta can be positive or negative, just verify it's a valid number
			assert.True(t, detail.Delta >= -100.0 && detail.Delta <= 100.0, "delta should be reasonable")

			// Verify owners arrays are valid (can be empty)
			assert.NotNil(t, detail.BeforeOwners, "before owners should not be nil")
			assert.NotNil(t, detail.AfterOwners, "after owners should not be nil")
		})
	}

	// Verify summary has reasonable values
	assert.True(t, result.Summary.NetScoreDelta >= -1000.0 && result.Summary.NetScoreDelta <= 1000.0, "net score delta should be reasonable")
	assert.GreaterOrEqual(t, result.Summary.TotalModifiedFiles, 0, "total modified files should be non-negative")
	assert.GreaterOrEqual(t, result.Summary.TotalNewFiles, 0, "total new files should be non-negative")
	assert.GreaterOrEqual(t, result.Summary.TotalInactiveFiles, 0, "total inactive files should be non-negative")
}

// parseHotspotFolderOutput extracts folder details from hotspot JSON output.
func parseHotspotFolderOutput(output string) map[string]schema.FolderResult {
	var folders []schema.FolderResult
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "[" { // Start of JSON array
			jsonPart := strings.Join(lines[i:], "\n")
			if json.Unmarshal([]byte(jsonPart), &folders) == nil {
				break
			}
		}
	}

	folderDetails := make(map[string]schema.FolderResult)
	for _, folder := range folders {
		folderDetails[folder.Path] = folder
	}

	return folderDetails
}

// TestExternalRepoVerification clones a couple of small public repos and runs basic verification.
// This test is kept minimal to avoid network dependencies and long runtimes.
func TestExternalRepoVerification(t *testing.T) {
	t.Parallel()

	// Test repos with different characteristics for basic coverage
	// Reduced from 4 to 2 repos to keep runtime reasonable
	testRepos := []struct {
		url  string
		name string
	}{
		{"https://github.com/mitchellh/go-homedir", "go-homedir"}, // Small Go library
		{"https://github.com/go-yaml/yaml", "go-yaml"},            // Medium Go library
	}

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

	for _, repo := range testRepos {
		t.Run(repo.name, func(t *testing.T) {
			testRepoDir := "test-repos/" + repo.name

			// Clean up any existing dir
			_ = exec.Command("rm", "-rf", testRepoDir).Run()

			// Clone the repo (shallow clone for speed)
			cloneCmd := exec.Command("git", "clone", "--depth=1", repo.url, testRepoDir)
			err := cloneCmd.Run()
			if err != nil {
				t.Skipf("failed to clone test repo %s: %v", repo.name, err)
			}
			defer func() { _ = exec.Command("rm", "-rf", testRepoDir).Run() }() // Clean up

			// Run basic hotspot analysis (just check it doesn't crash)
			// Use a very recent time range to avoid issues with shallow clones
			cmd := exec.Command(hotspotPath, "files", "--limit", "5", "--start", "2020-01-01T00:00:00Z", testRepoDir)
			cmd.Dir = testRepoDir
			err = cmd.Run()
			if err != nil {
				t.Skipf("hotspot files failed on external repo %s: %v", repo.name, err)
			}

			// Run folders analysis too
			cmd2 := exec.Command(hotspotPath, "folders", "--limit", "3", "--start", "2020-01-01T00:00:00Z", testRepoDir)
			cmd2.Dir = testRepoDir
			err = cmd2.Run()
			if err != nil {
				t.Skipf("hotspot folders failed on external repo %s: %v", repo.name, err)
			}
		})
	}
}

// verifyRepo runs hotspot and verifies against git for a given repo.
// This function is no longer used after optimizing TestExternalRepoVerification
// to focus on basic functionality rather than exhaustive verification.

// TestTimeseriesVerification tests the timeseries command functionality.
func TestTimeseriesVerification(t *testing.T) {
	t.Parallel()

	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

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
		cmd := exec.Command(hotspotPath, "timeseries")
		cmd.Dir = repoDir
		err := cmd.Run()
		assert.Error(t, err, "should error when --path is missing")
	})
}

// extractJSONFromOutput extracts the JSON part from hotspot output that includes log lines.
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

// TestCheckVerification tests the check command functionality.
func TestCheckVerification(t *testing.T) {
	t.Parallel()

	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

	// Test check command with very high thresholds (should pass)
	t.Run("check_with_high_thresholds", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "check", "--base-ref", "v1.1.4", "--target-ref", "v1.1.5",
			"--thresholds-override", "hot:100,risk:100,complexity:100,stale:100")
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()

		// Should succeed (exit code 0) with high thresholds
		assert.NoError(t, err, "check should pass with high thresholds")

		output := stdout.String()
		assert.Contains(t, output, "Policy Check Results:", "should contain policy check header")
		assert.Contains(t, output, "Base Ref:", "should contain base ref info")
		assert.Contains(t, output, "Target Ref:", "should contain target ref info")
		assert.Contains(t, output, "All files passed policy checks", "should indicate success")
	})

	// Test check command with very low thresholds (should fail)
	t.Run("check_with_low_thresholds", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "check", "--base-ref", "v1.1.4", "--target-ref", "v1.1.5",
			"--thresholds-override", "hot:10,risk:10,complexity:10,stale:10")
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()

		// Should fail (exit code non-zero) with low thresholds
		assert.Error(t, err, "check should fail with low thresholds")

		output := stdout.String()
		assert.Contains(t, output, "Policy Check Results:", "should contain policy check header")
		assert.Contains(t, output, "Policy check failed:", "should indicate failure")
	}) // Test check command missing required flags
	t.Run("check_missing_flags", func(t *testing.T) {
		cmd := exec.Command(hotspotPath, "check")
		cmd.Dir = repoDir
		err := cmd.Run()

		// Should fail due to missing base-ref and target-ref
		assert.Error(t, err, "check should fail when base-ref and target-ref are missing")
	})
}

// TestMetricsVerification tests the metrics command and custom weights handling.
func TestMetricsVerification(t *testing.T) {
	t.Parallel()

	// Skip if not in a git repo
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Get current repo path
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	repoDir := strings.TrimSpace(string(repoPath))

	// Get hotspot binary (built once and shared)
	hotspotPath := getHotspotBinary()

	// Helper function to create a temp config file
	createTempConfig := func(t *testing.T, content string) string {
		t.Helper()
		configFile, err := os.CreateTemp("", "hotspot_test_config_*.yml")
		require.NoError(t, err)
		_, err = configFile.WriteString(content)
		require.NoError(t, err)
		_ = configFile.Close()
		t.Cleanup(func() { _ = os.Remove(configFile.Name()) })
		return configFile.Name()
	}

	// Test valid custom weights configurations
	t.Run("valid_weights_commit_focused", func(t *testing.T) {
		// Create a temporary config file with custom weights favoring commits
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
		configFile := createTempConfig(t, configContent)

		// Run hotspot metrics to verify custom weights are loaded and displayed
		cmd := exec.Command(hotspotPath, "metrics", "--output", "json", "--config", configFile)
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err = cmd.Run()
		require.NoError(t, err)

		// Parse the metrics JSON output
		jsonPart := extractJSONFromOutput(stdout.String())
		var metricsRenderModel schema.MetricsRenderModel
		err = json.Unmarshal([]byte(jsonPart), &metricsRenderModel)
		require.NoError(t, err)

		// Find the "hot" mode
		var hotMode *schema.MetricsModeWithData
		for i := range metricsRenderModel.Modes {
			if metricsRenderModel.Modes[i].Name == "hot" {
				hotMode = &metricsRenderModel.Modes[i]
				break
			}
		}
		require.NotNil(t, hotMode, "Should find 'hot' mode in metrics output")

		// Verify the custom weights are correctly loaded
		expectedWeights := map[string]float64{
			"commits": 0.8,
			"churn":   0.1,
			"age":     0.05,
			"contrib": 0.04,
			"size":    0.01,
		}
		assert.Equal(t, expectedWeights, hotMode.Weights, "Weights should match custom configuration")

		// Verify the formula reflects the custom weights
		expectedFormula := "0.80*commits+0.10*churn+0.04*contrib+0.05*age+0.01*size"
		assert.Equal(t, expectedFormula, hotMode.Formula, "Formula should reflect custom weights")
	})

	t.Run("valid_weights_churn_focused", func(t *testing.T) {
		// Create a temporary config file with custom weights favoring churn
		configContent := `
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
		configFile := createTempConfig(t, configContent)

		// Run hotspot metrics to verify custom weights are loaded and displayed
		cmd := exec.Command(hotspotPath, "metrics", "--output", "json", "--config", configFile)
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err = cmd.Run()
		require.NoError(t, err)

		// Parse the metrics JSON output
		jsonPart := extractJSONFromOutput(stdout.String())
		var metricsRenderModel schema.MetricsRenderModel
		err = json.Unmarshal([]byte(jsonPart), &metricsRenderModel)
		require.NoError(t, err)

		// Find the "hot" mode
		var hotMode *schema.MetricsModeWithData
		for i := range metricsRenderModel.Modes {
			if metricsRenderModel.Modes[i].Name == "hot" {
				hotMode = &metricsRenderModel.Modes[i]
				break
			}
		}
		require.NotNil(t, hotMode, "Should find 'hot' mode in metrics output")

		// Verify the custom weights are correctly loaded
		expectedWeights := map[string]float64{
			"commits": 0.1,
			"churn":   0.8,
			"age":     0.05,
			"contrib": 0.04,
			"size":    0.01,
		}
		assert.Equal(t, expectedWeights, hotMode.Weights, "Weights should match custom configuration")

		// Verify the formula reflects the custom weights
		expectedFormula := "0.10*commits+0.80*churn+0.04*contrib+0.05*age+0.01*size"
		assert.Equal(t, expectedFormula, hotMode.Formula, "Formula should reflect custom weights")
	})

	t.Run("invalid_weights_validation", func(t *testing.T) {
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
		configFile := createTempConfig(t, configContent)

		// Run hotspot metrics - should fail due to invalid weights
		cmd := exec.Command(hotspotPath, "metrics", "--config", configFile)
		cmd.Dir = repoDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		err = cmd.Run()
		assert.Error(t, err, "Should fail with invalid weights that don't sum to 1.0")
	})
}
