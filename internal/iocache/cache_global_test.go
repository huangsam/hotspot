package iocache

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDBFilePath(t *testing.T) {
	path := GetDBFilePath()

	// Should not be empty
	assert.NotEmpty(t, path)

	// Should contain the database name
	assert.Contains(t, path, ".hotspot_cache.db")

	// Should be in home directory
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, homeDir), "path %s should start with home dir %s", path, homeDir)
}

func TestGetAnalysisDBFilePath(t *testing.T) {
	path := GetAnalysisDBFilePath()

	// Should not be empty
	assert.NotEmpty(t, path)

	// Should contain the database name
	assert.Contains(t, path, ".hotspot_analysis.db")

	// Should be in home directory
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, homeDir), "path %s should start with home dir %s", path, homeDir)
}
