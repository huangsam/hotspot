package iocache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateAnalysis_NoneBackend(t *testing.T) {
	err := MigrateAnalysis(schema.NoneBackend, "", targetLatestVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migrations are not supported for 'none' backend")
}

func TestMigrateAnalysis_SQLite(t *testing.T) {
	// Create a temporary database file for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_migration.db")

	// Run migration to latest version (should go to version 1)
	err := MigrateAnalysis(schema.SQLiteBackend, dbPath, targetLatestVersion)
	require.NoError(t, err)

	// Verify migration was successful by checking the database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Run migration again (should be a no-op)
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, targetLatestVersion)
	assert.NoError(t, err)

	// Run migration to a specific version (version 1)
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, 1)
	assert.NoError(t, err)

	// Rollback to version 0
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, targetInitialVersion)
	assert.NoError(t, err)

	// Migrate back up to version 1
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, 1)
	assert.NoError(t, err)
}

func TestMigrateAnalysis_SQLiteInvalidPath(t *testing.T) {
	// Test with an invalid database path
	invalidPath := "/invalid/path/to/db.sqlite"
	err := MigrateAnalysis(schema.SQLiteBackend, invalidPath, targetLatestVersion)
	assert.Error(t, err)
}

func TestMigrateAnalysis_InvalidVersion(t *testing.T) {
	// Create a temporary database file for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_invalid_version.db")

	// Run migration with an invalid version number
	err := MigrateAnalysis(schema.SQLiteBackend, dbPath, -2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target version")
}

func TestMigrateAnalysis_SQLiteInMemory(t *testing.T) {
	// Test with in-memory database
	err := MigrateAnalysis(schema.SQLiteBackend, ":memory:", targetLatestVersion)
	require.NoError(t, err)
}
