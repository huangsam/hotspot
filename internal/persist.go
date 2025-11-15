package internal

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/huangsam/hotspot/schema"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

// databaseName is the name of the SQLite database file.
const databaseName = ".hotspot_cache.db"

// PersistenceManager defines the interface for managing persistence stores.
// This allows the persistence layer to be mocked for testing.
type PersistenceManager interface {
	GetActivityStore() *PersistStore
}

// PersistenceStore defines the interface for persistence data storage.
// This allows mocking the store for testing.
type PersistenceStore interface {
	Get(key string) ([]byte, int, int64, error)
	Set(key string, value []byte, version int, timestamp int64) error
}

// PersistStoreManager manages multiple PersistStore instances.
type PersistStoreManager struct {
	sync.RWMutex // Protects the store pointers during initialization
	activity     *PersistStore
}

var _ PersistenceManager = &PersistStoreManager{} // Compile-time check

// GetActivityStore returns the activity PersistStore.
func (mgr *PersistStoreManager) GetActivityStore() *PersistStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.activity
}

// PersistStore handles durable storage operations using various database backends.
type PersistStore struct {
	db         *sql.DB
	tableName  string
	backend    schema.CacheBackend
	driverName string
}

var _ PersistenceStore = &PersistStore{} // Compile-time check

// NewPersistStore initializes and returns a new PersistStore based on the backend type.
func NewPersistStore(tableName string, backend schema.CacheBackend, connStr string) (*PersistStore, error) {
	var db *sql.DB
	var err error
	var driverName string

	switch backend {
	case schema.SQLiteBackend:
		driverName = "sqlite3"
		dbPath := GetDBFilePath()
		db, err = sql.Open(driverName, dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database: %w", err)
		}

	case schema.MySQLBackend:
		driverName = "mysql"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open MySQL database: %w", err)
		}

	case schema.PostgreSQLBackend:
		driverName = "postgres"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
		}

	case schema.NoneBackend:
		// Return a no-op store for disabled caching
		return &PersistStore{
			db:         nil,
			tableName:  tableName,
			backend:    backend,
			driverName: "",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported cache backend: %s", backend)
	}

	// Ping to verify connection (skip for NoneBackend)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping %s database: %w", backend, err)
	}

	// Create the table schema
	query := getCreateTableQuery(tableName, backend)
	if _, err := db.Exec(query); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	return &PersistStore{
		db:         db,
		tableName:  tableName,
		backend:    backend,
		driverName: driverName,
	}, nil
}

// getCreateTableQuery returns the CREATE TABLE query for the given backend.
func getCreateTableQuery(tableName string, backend schema.CacheBackend) string {
	switch backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key VARCHAR(255) PRIMARY KEY,
				value BLOB NOT NULL,
				version INT NOT NULL,
				timestamp BIGINT NOT NULL
			);
		`, tableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key TEXT PRIMARY KEY,
				value BYTEA NOT NULL,
				version INTEGER NOT NULL,
				timestamp BIGINT NOT NULL
			);
		`, tableName)

	default: // SQLite
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key TEXT PRIMARY KEY,
				value BLOB NOT NULL,
				version INTEGER NOT NULL,
				timestamp INTEGER NOT NULL
			);
		`, tableName)
	}
}

// GetDBFilePath returns the path to the SQLite DB file.
func GetDBFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, databaseName)
}

// Get retrieves a value by key from the store.
func (ps *PersistStore) Get(key string) ([]byte, int, int64, error) {
	// Return not found error for NoneBackend
	if ps.backend == schema.NoneBackend || ps.db == nil {
		return nil, 0, 0, sql.ErrNoRows
	}

	var value []byte
	var version int
	var ts int64

	// Use backend-specific placeholder
	placeholder := ps.getPlaceholder()
	query := fmt.Sprintf(`SELECT value, version, timestamp FROM %s WHERE key = %s`, ps.tableName, placeholder)
	row := ps.db.QueryRow(query, key)

	if err := row.Scan(&value, &version, &ts); err != nil {
		return nil, 0, 0, err
	}
	return value, version, ts, nil
}

// Set inserts or replaces a key/value pair in the store.
func (ps *PersistStore) Set(key string, value []byte, version int, timestamp int64) error {
	// Skip for NoneBackend
	if ps.backend == schema.NoneBackend || ps.db == nil {
		return nil
	}

	// Use backend-specific UPSERT
	query := ps.getUpsertQuery()
	_, err := ps.db.Exec(query, key, value, version, timestamp)
	return err
}

// getPlaceholder returns the parameter placeholder for the backend.
func (ps *PersistStore) getPlaceholder() string {
	switch ps.backend {
	case schema.PostgreSQLBackend:
		return "$1"
	default:
		return "?"
	}
}

// getUpsertQuery returns the UPSERT query for the backend.
func (ps *PersistStore) getUpsertQuery() string {
	switch ps.backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`INSERT INTO %s (key, value, version, timestamp) VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE value = VALUES(value), version = VALUES(version), timestamp = VALUES(timestamp)`, ps.tableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`INSERT INTO %s (key, value, version, timestamp) VALUES ($1, $2, $3, $4)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, version = EXCLUDED.version, timestamp = EXCLUDED.timestamp`, ps.tableName)

	default: // SQLite
		return fmt.Sprintf(`INSERT OR REPLACE INTO %s (key, value, version, timestamp) VALUES (?, ?, ?, ?)`, ps.tableName)
	}
}

// Close closes the underlying DB connection.
func (ps *PersistStore) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}
