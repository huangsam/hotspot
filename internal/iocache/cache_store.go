// Package iocache is for caching I/O calls.
package iocache

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3"    // SQLite driver
)

// CacheStoreImpl handles durable storage operations using various database backends.
type CacheStoreImpl struct {
	db         *sql.DB
	tableName  string
	backend    schema.DatabaseBackend
	driverName string
	connStr    string
}

var _ contract.CacheStore = &CacheStoreImpl{} // Compile-time check

// NewCacheStore initializes and returns a new CacheStore based on the backend type.
func NewCacheStore(tableName string, backend schema.DatabaseBackend, connStr string) (contract.CacheStore, error) {
	// Validate table name to prevent SQL injection
	if err := validateTableName(tableName); err != nil {
		return nil, err
	}

	var db *sql.DB
	var err error
	var driverName string

	switch backend {
	case schema.SQLiteBackend:
		driverName = "sqlite3"
		dbPath := connStr
		if dbPath == "" {
			dbPath = GetDBFilePath()
		}
		db, err = sql.Open(driverName, dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SQLite cache at %q: %w. Ensure the directory is writable", dbPath, err)
		}
		// Limit SQLite to a single open connection to avoid "database is locked" errors
		db.SetMaxOpenConns(1)

	case schema.MySQLBackend:
		// connStr should be:
		// user:password@tcp(host:port)/dbname
		driverName = "mysql"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MySQL cache: %w. Check connection format: user:password@tcp(host:port)/dbname", err)
		}

	case schema.PostgreSQLBackend:
		// connStr should be:
		// host=localhost port=5432 user=postgres password=mysecretpassword dbname=postgres
		driverName = "pgx"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL cache: %w. Check connection format: host=localhost port=5432 user=postgres dbname=mydb", err)
		}

	case schema.NoneBackend:
		// Return a no-op store for disabled caching
		return &CacheStoreImpl{
			db:         nil,
			tableName:  tableName,
			backend:    backend,
			driverName: "",
			connStr:    connStr,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported cache backend: %s. Must be sqlite, mysql, postgresql, or none", backend)
	}

	// Ping to verify connection (skip for NoneBackend)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to %s database. Check that the server is running and connection parameters are valid: %w", backend, err)
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
		connStr:    connStr,
	}, nil
}

// getCreateTableQuery returns the CREATE TABLE query for the given backend.
func getCreateTableQuery(tableName string, backend schema.DatabaseBackend) string {
	quotedTableName := quoteTableName(tableName, backend)
	switch backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				cache_key VARCHAR(255) PRIMARY KEY,
				cache_value BLOB NOT NULL,
				cache_version INT NOT NULL,
				cache_timestamp BIGINT NOT NULL
			);
		`, quotedTableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				cache_key TEXT PRIMARY KEY,
				cache_value BYTEA NOT NULL,
				cache_version INTEGER NOT NULL,
				cache_timestamp BIGINT NOT NULL
			);
		`, quotedTableName)

	default: // SQLite
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				cache_key TEXT PRIMARY KEY,
				cache_value BLOB NOT NULL,
				cache_version INTEGER NOT NULL,
				cache_timestamp INTEGER NOT NULL
			);
		`, quotedTableName)
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
	quotedTableName := quoteTableName(ps.tableName, ps.backend)
	placeholder := ps.getPlaceholder()
	query := fmt.Sprintf(`SELECT cache_value, cache_version, cache_timestamp FROM %s WHERE cache_key = %s`, quotedTableName, placeholder)
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
	quotedTableName := quoteTableName(ps.tableName, ps.backend)
	switch ps.backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`INSERT INTO %s (cache_key, cache_value, cache_version, cache_timestamp) VALUES (?, ?, ?, ?) AS new
			ON DUPLICATE KEY UPDATE cache_value = new.cache_value, cache_version = new.cache_version, cache_timestamp = new.cache_timestamp`, quotedTableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`INSERT INTO %s (cache_key, cache_value, cache_version, cache_timestamp) VALUES ($1, $2, $3, $4)
			ON CONFLICT (cache_key) DO UPDATE SET cache_value = EXCLUDED.cache_value, cache_version = EXCLUDED.cache_version, cache_timestamp = EXCLUDED.cache_timestamp`, quotedTableName)

	default: // SQLite
		return fmt.Sprintf(`INSERT OR REPLACE INTO %s (cache_key, cache_value, cache_version, cache_timestamp) VALUES (?, ?, ?, ?)`, quotedTableName)
	}
}

// Close closes the underlying DB connection.
func (ps *CacheStoreImpl) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}

// GetStatus returns status information about the cache store.
func (ps *CacheStoreImpl) GetStatus() (schema.CacheStatus, error) {
	status := schema.CacheStatus{
		Backend:   string(ps.backend),
		Connected: ps.db != nil,
	}

	if ps.backend == schema.NoneBackend || ps.db == nil {
		return status, nil
	}

	quotedTableName := quoteTableName(ps.tableName, ps.backend)

	// Get total entries
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTableName)
	row := ps.db.QueryRow(countQuery)
	if err := row.Scan(&status.TotalEntries); err != nil {
		return status, fmt.Errorf("failed to get total entries: %w", err)
	}

	if status.TotalEntries == 0 {
		return status, nil
	}

	// Get last entry time
	lastQuery := fmt.Sprintf("SELECT MAX(cache_timestamp) FROM %s", quotedTableName)
	row = ps.db.QueryRow(lastQuery)
	var lastTs int64
	if err := row.Scan(&lastTs); err != nil {
		return status, fmt.Errorf("failed to get last entry time: %w", err)
	}
	status.LastEntryTime = time.Unix(lastTs, 0)

	// Get oldest entry time
	oldestQuery := fmt.Sprintf("SELECT MIN(cache_timestamp) FROM %s", quotedTableName)
	row = ps.db.QueryRow(oldestQuery)
	var oldestTs int64
	if err := row.Scan(&oldestTs); err != nil {
		return status, fmt.Errorf("failed to get oldest entry time: %w", err)
	}
	status.OldestEntryTime = time.Unix(oldestTs, 0)

	// Estimate table size (approximate)
	// For SQLite, use page_count * page_size
	// For others, estimate based on row count (rough approximation)
	if ps.backend == schema.SQLiteBackend {
		sizeQuery := "SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()"
		row = ps.db.QueryRow(sizeQuery)
		if err := row.Scan(&status.TableSizeBytes); err != nil {
			// If pragma fails, skip size
			status.TableSizeBytes = 0
		}
	} else {
		// For MySQL/PostgreSQL, use database-specific size queries
		var sizeQuery string
		switch ps.backend {
		case schema.MySQLBackend:
			// Fallback rough estimate if information_schema query fails
			status.TableSizeBytes = int64(status.TotalEntries) * 1000

			// Use information_schema for MySQL
			cfg, err := mysql.ParseDSN(ps.connStr)
			if err != nil {
				break
			}
			dbName := cfg.DBName
			if dbName == "" {
				break
			}
			sizeQuery := "SELECT data_length + index_length FROM information_schema.tables WHERE table_schema = ? AND table_name = ?"
			row := ps.db.QueryRow(sizeQuery, dbName, ps.tableName)
			if err := row.Scan(&status.TableSizeBytes); err != nil {
				// Fallback if the query or scanning fails
				status.TableSizeBytes = int64(status.TotalEntries) * 1000
			}
		case schema.PostgreSQLBackend:
			// Use pg_total_relation_size for PostgreSQL
			sizeQuery = "SELECT pg_total_relation_size($1)"
			row = ps.db.QueryRow(sizeQuery, ps.tableName)
			if err := row.Scan(&status.TableSizeBytes); err != nil {
				status.TableSizeBytes = int64(status.TotalEntries) * 1000 // Fallback rough estimate
			}
		default:
			status.TableSizeBytes = int64(status.TotalEntries) * 1000 // Rough estimate
		}
	}

	return status, nil
}
