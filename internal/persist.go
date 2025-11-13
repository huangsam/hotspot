package internal

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// dbFileName is the name of the SQLite database file.
const dbFileName = ".hotspot_cache.db"

// PersistStoreManager manages multiple PersistStore instances.
type PersistStoreManager struct {
	sync.RWMutex // Protects the store pointers during initialization
	activity     *PersistStore
	fileStats    *PersistStore
}

// GetActivityStore returns the activity PersistStore.
func (mgr *PersistStoreManager) GetActivityStore() *PersistStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.activity
}

// GetFileStatsStore returns the file stats PersistStore.
func (mgr *PersistStoreManager) GetFileStatsStore() *PersistStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.fileStats
}

// PersistStore handles durable storage operations using SQLite.
type PersistStore struct {
	db        *sql.DB
	tableName string
}

// NewPersistStore initializes and returns a new PersistStore.
func NewPersistStore(tableName string) (*PersistStore, error) {
	dbPath := getDBFilePath()

	db, err := sql.Open("sqlite3", dbPath) // Replace with actual SQLite driver
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	// Set up the table with a key, a BLOB for the data, and a timestamp/version
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key TEXT PRIMARY KEY,
			value BLOB NOT NULL,
			version INTEGER NOT NULL,
			timestamp INTEGER NOT NULL
		);
	`, tableName)

	if _, err := db.Exec(query); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	return &PersistStore{db: db, tableName: tableName}, nil
}

// getDBFilePath returns the path to the SQLite DB file.
func getDBFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, dbFileName)
}

// Get retrieves a value by key from the store. Returns sql.ErrNoRows when missing.
func (ps *PersistStore) Get(key string) ([]byte, int, int64, error) {
	var value []byte
	var version int
	var ts int64
	// Note: Using `strings.TrimSpace(output)` for git hash in key generation, maybe use strings.TrimSpace here too
	row := ps.db.QueryRow(fmt.Sprintf(`SELECT value, version, timestamp FROM %s WHERE key = ?`, ps.tableName), key)
	if err := row.Scan(&value, &version, &ts); err != nil {
		return nil, 0, 0, err
	}
	return value, version, ts, nil
}

// Set inserts or replaces a key/value pair in the store.
func (ps *PersistStore) Set(key string, value []byte, version int, timestamp int64) error {
	query := fmt.Sprintf(`INSERT OR REPLACE INTO %s (key, value, version, timestamp) VALUES (?, ?, ?, ?)`, ps.tableName)
	_, err := ps.db.Exec(query, key, value, version, timestamp)
	return err
}

// Close closes the underlying DB connection.
func (ps *PersistStore) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}
