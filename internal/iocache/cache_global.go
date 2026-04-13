package iocache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/schema"
)

// activityTable is the name of the table for activity caching.
const activityTable = "activity_cache"

// Global Manager instance for main logic.
var (
	Manager   = &CacheStoreManager{}
	initOnce  sync.Once
	closeOnce sync.Once
)

// GetDBFilePath returns the path to the SQLite DB file for cache storage.
func GetDBFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".hotspot_cache.db"
	}
	return filepath.Join(homeDir, ".hotspot_cache.db")
}

// GetAnalysisDBFilePath returns the path to the SQLite DB file for analysis storage.
func GetAnalysisDBFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".hotspot_analysis.db"
	}
	return filepath.Join(homeDir, ".hotspot_analysis.db")
}

// InitStores initializes the global cache manager with separate cache and analysis stores.
// cacheBackend and cacheConnStr can be empty to disable cache initialization.
// analysisBackend and analysisConnStr can be empty to disable analysis tracking.
func InitStores(cacheBackend schema.DatabaseBackend, cacheConnStr string, analysisBackend schema.DatabaseBackend, analysisConnStr string, client git.Client) error {
	var initErr error

	initOnce.Do(func() {
		// This function body runs exactly once, even with concurrent calls.
		var err error

		// Initialize Activity Cache Store only if backend is configured
		var activityCacheStore CacheStore
		if cacheBackend != "" {
			activityCacheStore, err = NewCacheStore(activityTable, cacheBackend, cacheConnStr)
			if err != nil {
				initErr = fmt.Errorf("failed to initialize activity caching: %w", err)
				return
			}
		}

		// Initialize Analysis Store only if backend is configured
		var analysisStore AnalysisStore
		if analysisBackend != "" {
			analysisStore, err = NewAnalysisStore(analysisBackend, analysisConnStr, client)
			if err != nil {
				if activityCacheStore != nil {
					_ = activityCacheStore.Close()
				}
				initErr = fmt.Errorf("failed to initialize analysis store: %w", err)
				return
			}
		}

		// Assign to global manager
		Manager.activity = activityCacheStore
		Manager.analysis = analysisStore
	})

	// After once.Do, initErr will contain any error from the initialization block.
	return initErr
}

// CloseCaching should be called on application shutdown.
func CloseCaching() { // called in main defer
	closeOnce.Do(func() {
		Manager.Lock()
		defer Manager.Unlock()
		if Manager.activity != nil {
			_ = Manager.activity.Close()
		}
		if Manager.analysis != nil {
			_ = Manager.analysis.Close()
		}
	})
}

// ClearCache clears the cache for the specified backend.
// For SQLite, it deletes the database file.
// For SQL backends (MySQL/PostgreSQL), it drops the table.
// For NoneBackend, it does nothing.
func ClearCache(backend schema.DatabaseBackend, dbFilePath, connStr string) error {
	switch backend {
	case schema.SQLiteBackend:
		if dbFilePath == "" {
			return fmt.Errorf("dbFilePath cannot be empty for SQLite backend")
		}
		// Remove the file; ignore if it doesn't exist
		if err := os.Remove(dbFilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove SQLite database file %s: %w", dbFilePath, err)
		}
		return nil

	case schema.MySQLBackend:
		return clearSQLTable("mysql", connStr, activityTable)

	case schema.PostgreSQLBackend:
		return clearSQLTable("pgx", connStr, activityTable)

	case schema.NoneBackend:
		return nil

	default:
		return fmt.Errorf("unsupported cache backend for clearing: %s", backend)
	}
}

// ClearAnalysis clears the analysis data for the specified backend.
// For SQLite, it deletes the database file.
// For SQL backends (MySQL/PostgreSQL), it drops the analysis tables.
// For NoneBackend, it does nothing.
func ClearAnalysis(backend schema.DatabaseBackend, dbFilePath, connStr string) error {
	switch backend {
	case schema.SQLiteBackend:
		if dbFilePath == "" {
			return fmt.Errorf("dbFilePath cannot be empty for SQLite backend")
		}
		// Remove the file; ignore if it doesn't exist
		if err := os.Remove(dbFilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove SQLite database file %s: %w", dbFilePath, err)
		}
		return nil

	case schema.MySQLBackend:
		// Clear analysis tables
		tables := []string{analysisRunsTable, fileScoresMetricsTable}
		for _, table := range tables {
			if err := clearSQLTable("mysql", connStr, table); err != nil {
				return err
			}
		}
		return nil

	case schema.PostgreSQLBackend:
		// Clear analysis tables
		tables := []string{analysisRunsTable, fileScoresMetricsTable}
		for _, table := range tables {
			if err := clearSQLTable("pgx", connStr, table); err != nil {
				return err
			}
		}
		return nil

	case schema.NoneBackend:
		return nil

	default:
		return fmt.Errorf("unsupported analysis backend for clearing: %s", backend)
	}
}

// clearSQLTable connects to the SQL database and deletes all rows from the table.
func clearSQLTable(driverName, connStr, tableName string) error {
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s database: %w", driverName, err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping %s database: %w", driverName, err)
	}

	// Use DELETE instead of DROP TABLE to preserve the schema (which is now
	// managed by migrations).  DROP would leave the schema_migrations table
	// at the latest version while the data tables no longer exist.
	// Ignore errors from non-existent tables (fresh database before first migration).
	query := fmt.Sprintf("DELETE FROM %s", tableName)
	_, _ = db.Exec(query)

	return nil
}
