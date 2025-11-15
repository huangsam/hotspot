package internal

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/huangsam/hotspot/schema"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

// CacheManager defines the interface for managing cache stores.
// This allows the cache layer to be mocked for testing.
type CacheManager interface {
	GetActivityStore() CacheStore
}

// CacheStore defines the interface for cache data storage.
// This allows mocking the store for testing.
type CacheStore interface {
	Get(key string) ([]byte, int, int64, error)
	Set(key string, value []byte, version int, timestamp int64) error
	Close() error
}

// CacheStoreManager manages multiple CacheStore instances.
type CacheStoreManager struct {
	sync.RWMutex // Protects the store pointers during initialization
	activity     CacheStore
}

var _ CacheManager = &CacheStoreManager{} // Compile-time check

// GetActivityStore returns the activity CacheStore.
func (mgr *CacheStoreManager) GetActivityStore() CacheStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.activity
}

// CacheStoreImpl handles durable storage operations using various database backends.
type CacheStoreImpl struct {
	db         *sql.DB
	tableName  string
	backend    schema.CacheBackend
	driverName string
}

var _ CacheStore = &CacheStoreImpl{} // Compile-time check

// NewCacheStore initializes and returns a new CacheStore based on the backend type.
func NewCacheStore(tableName string, backend schema.CacheBackend, connStr string) (CacheStore, error) {
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
		// connStr should be:
		// user:password@tcp(host:port)/dbname
		driverName = "mysql"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open MySQL database: %w", err)
		}

	case schema.PostgreSQLBackend:
		// connStr should be:
		// host=localhost port=5432 user=postgres password=mysecretpassword dbname=postgres
		driverName = "pgx"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
		}

	case schema.NoneBackend:
		// Return a no-op store for disabled caching
		return &CacheStoreImpl{
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

	return &CacheStoreImpl{
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
				cache_key VARCHAR(255) PRIMARY KEY,
				cache_value BLOB NOT NULL,
				cache_version INT NOT NULL,
				cache_timestamp BIGINT NOT NULL
			);
		`, tableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				cache_key TEXT PRIMARY KEY,
				cache_value BYTEA NOT NULL,
				cache_version INTEGER NOT NULL,
				cache_timestamp BIGINT NOT NULL
			);
		`, tableName)

	default: // SQLite
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				cache_key TEXT PRIMARY KEY,
				cache_value BLOB NOT NULL,
				cache_version INTEGER NOT NULL,
				cache_timestamp INTEGER NOT NULL
			);
		`, tableName)
	}
}

// Get retrieves a value by key from the store.
func (ps *CacheStoreImpl) Get(key string) ([]byte, int, int64, error) {
	// Return not found error for NoneBackend
	if ps.backend == schema.NoneBackend || ps.db == nil {
		return nil, 0, 0, sql.ErrNoRows
	}

	var value []byte
	var version int
	var ts int64

	// Use backend-specific placeholder
	placeholder := ps.getPlaceholder()
	query := fmt.Sprintf(`SELECT cache_value, cache_version, cache_timestamp FROM %s WHERE cache_key = %s`, ps.tableName, placeholder)
	row := ps.db.QueryRow(query, key)

	if err := row.Scan(&value, &version, &ts); err != nil {
		return nil, 0, 0, err
	}
	return value, version, ts, nil
}

// Set inserts or replaces a key/value pair in the store.
func (ps *CacheStoreImpl) Set(key string, value []byte, version int, timestamp int64) error {
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
func (ps *CacheStoreImpl) getPlaceholder() string {
	switch ps.backend {
	case schema.PostgreSQLBackend:
		return "$1"
	default: // SQLite and MySQL
		return "?"
	}
}

// getUpsertQuery returns the UPSERT query for the backend.
func (ps *CacheStoreImpl) getUpsertQuery() string {
	switch ps.backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`INSERT INTO %s (cache_key, cache_value, cache_version, cache_timestamp) VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE cache_value = VALUES(cache_value), cache_version = VALUES(cache_version), cache_timestamp = VALUES(cache_timestamp)`, ps.tableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`INSERT INTO %s (cache_key, cache_value, cache_version, cache_timestamp) VALUES ($1, $2, $3, $4)
			ON CONFLICT (cache_key) DO UPDATE SET cache_value = EXCLUDED.cache_value, cache_version = EXCLUDED.cache_version, cache_timestamp = EXCLUDED.cache_timestamp`, ps.tableName)

	default: // SQLite
		return fmt.Sprintf(`INSERT OR REPLACE INTO %s (cache_key, cache_value, cache_version, cache_timestamp) VALUES (?, ?, ?, ?)`, ps.tableName)
	}
}

// Close closes the underlying DB connection.
func (ps *CacheStoreImpl) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}
