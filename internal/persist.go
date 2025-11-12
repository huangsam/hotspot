package internal

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
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

// PersistStore handles durable storage operations using SQLite.
type PersistStore struct {
	db        *sql.DB
	tableName string
}

var _ PersistenceStore = &PersistStore{} // Compile-time check

// NewPersistStore initializes and returns a new PersistStore.
func NewPersistStore(tableName string) (*PersistStore, error) {
	dbPath := GetDBFilePath()

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

// GetDBFilePath returns the path to the SQLite DB file.
func GetDBFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, databaseName)
}

// Get retrieves a value by key from the store.
func (ps *PersistStore) Get(key string) ([]byte, int, int64, error) {
	var value []byte
	var version int
	var ts int64
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
