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
	err := MigrateAnalysis(schema.NoneBackend, "", -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migrations are not supported for NoneBackend")
}

func TestMigrateAnalysis_SQLite(t *testing.T) {
	// Create a temporary database file for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_migration.db")

	// Run migration to latest version (should go to version 1)
	err := MigrateAnalysis(schema.SQLiteBackend, dbPath, -1)
	require.NoError(t, err)

	// Verify migration was successful by checking the database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Run migration again (should be a no-op)
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, -1)
	assert.NoError(t, err)

	// Run migration to a specific version (version 1)
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, 1)
	assert.NoError(t, err)

	// Rollback to version 0
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, 0)
	assert.NoError(t, err)

	// Migrate back up to version 1
	err = MigrateAnalysis(schema.SQLiteBackend, dbPath, 1)
	assert.NoError(t, err)
}

func TestMigrateAnalysis_SQLiteInMemory(t *testing.T) {
	// Test with in-memory database
	err := MigrateAnalysis(schema.SQLiteBackend, ":memory:", -1)
	require.NoError(t, err)
}
