//go:build basic || database

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/huangsam/hotspot/schema"
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

// runHotspotCommand executes the hotspot binary with the given arguments and returns combined output and error.
// It sets the working directory to the project root.
func runHotspotCommand(t *testing.T, args ...string) ([]byte, error) {
	hotspotPath := getHotspotBinary()
	cmd := exec.Command(hotspotPath, args...)

	// Get repo root for execution
	repoPath, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo root: %w", err)
	}
	repoDir := strings.TrimSpace(string(repoPath))

	cmd.Dir = repoDir

	// Isolate database for integration tests to prevent SQLITE_BUSY errors in parallel runs.
	// We only do this if we're using the SQLite backend (the default).
	cacheBackend := schema.DatabaseBackend(os.Getenv("HOTSPOT_CACHE_BACKEND"))
	analysisBackend := schema.DatabaseBackend(os.Getenv("HOTSPOT_ANALYSIS_BACKEND"))
	if (cacheBackend == "" || cacheBackend == schema.SQLiteBackend) &&
		(analysisBackend == "" || analysisBackend == schema.SQLiteBackend) {
		tempDir := t.TempDir()
		tempCacheDB := filepath.Join(tempDir, "hotspot_cache.db")
		tempAnalysisDB := filepath.Join(tempDir, "hotspot_analysis.db")
		cmd.Env = append(os.Environ(),
			"HOTSPOT_CACHE_DB_CONNECT="+tempCacheDB,
			"HOTSPOT_ANALYSIS_DB_CONNECT="+tempAnalysisDB,
		)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("CLI Command Failed: %s\nError: %v\nOutput: %s", cmd.String(), err, string(output))
	}
	return output, err
}
