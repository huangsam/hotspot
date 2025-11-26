//go:build basic || database

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
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
