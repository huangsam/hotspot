package iocache

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/huangsam/hotspot/schema"
)

func TestCaching(t *testing.T) {
	t.Run("single setup", func(t *testing.T) {
		// Clean up any existing test database
		testDBPath := GetDBFilePath()
		defer func() { _ = os.Remove(testDBPath) }()
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Test initialization with SQLite backend
		err := InitCaching(schema.SQLiteBackend, "")
		if err != nil {
			t.Fatalf("Failed to initialize persistence: %v", err)
		}

		// Test that Manager is accessible
		if Manager == nil {
			t.Fatal("Manager is nil")
		}

		// Test that stores are accessible
		if Manager.GetActivityStore() == nil {
			t.Fatal("Activity store is nil")
		}

		// Test cleanup
		CloseCaching()

		// Verify database file was created
		if _, err := os.Stat(testDBPath); os.IsNotExist(err) {
			t.Fatal("Database file was not created")
		}
	})

	t.Run("idempotent setup", func(t *testing.T) {
		// Clean up any existing test database
		testDBPath := GetDBFilePath()
		defer func() { _ = os.Remove(testDBPath) }()
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Multiple initializations should be safe (sync.Once)
		err1 := InitCaching(schema.SQLiteBackend, "")
		err2 := InitCaching(schema.SQLiteBackend, "")
		err3 := InitCaching(schema.SQLiteBackend, "")

		if err1 != nil {
			t.Fatalf("First init failed: %v", err1)
		}
		if err2 != nil {
			t.Fatalf("Second init failed: %v", err2)
		}
		if err3 != nil {
			t.Fatalf("Third init failed: %v", err3)
		}

		// Multiple closes should be safe (sync.Once)
		CloseCaching()
		CloseCaching()
		CloseCaching()
	})

	t.Run("none backend", func(t *testing.T) {
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Test initialization with None backend (no database)
		err := InitCaching(schema.NoneBackend, "")
		if err != nil {
			t.Fatalf("Failed to initialize persistence with none backend: %v", err)
		}

		// Test that Manager is accessible
		if Manager == nil {
			t.Fatal("Manager is nil")
		}

		// Test that stores are accessible
		store := Manager.GetActivityStore()
		if store == nil {
			t.Fatal("Activity store is nil")
		}

		// Test cleanup (should be safe even with no DB)
		CloseCaching()
	})

	t.Run("none backend operations", func(t *testing.T) {
		// Create a none backend store directly
		store, err := NewCacheStore("test_table", schema.NoneBackend, "")
		if err != nil {
			t.Fatalf("Failed to create none backend store: %v", err)
		}

		// Test Get returns error (no data)
		_, _, _, err = store.Get("test_key")
		if err == nil {
			t.Fatal("Expected error from Get on none backend")
		}

		// Test Set is no-op (no error)
		err = store.Set("test_key", []byte("test_value"), 1, 123456789)
		if err != nil {
			t.Fatalf("Set should not error on none backend: %v", err)
		}

		// Verify Get still returns error after Set (no-op)
		_, _, _, err = store.Get("test_key")
		if err == nil {
			t.Fatal("Expected error from Get after Set on none backend")
		}

		// Close is safe
		err = store.Close()
		if err != nil {
			t.Fatalf("Close should not error on none backend: %v", err)
		}
	})
}

// TestValidateTableName tests the validateTableName function with various inputs.
func TestValidateTableName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		wantErr   bool
	}{
		{
			name:      "valid simple name",
			tableName: "test_table",
			wantErr:   false,
		},
		{
			name:      "valid name with numbers",
			tableName: "test_table_123",
			wantErr:   false,
		},
		{
			name:      "valid name starting with underscore",
			tableName: "_test_table",
			wantErr:   false,
		},
		{
			name:      "valid uppercase name",
			tableName: "TEST_TABLE",
			wantErr:   false,
		},
		{
			name:      "valid mixed case",
			tableName: "TestTable_123",
			wantErr:   false,
		},
		{
			name:      "empty name",
			tableName: "",
			wantErr:   true,
		},
		{
			name:      "starts with number",
			tableName: "123_table",
			wantErr:   true,
		},
		{
			name:      "contains dash",
			tableName: "test-table",
			wantErr:   true,
		},
		{
			name:      "contains space",
			tableName: "test table",
			wantErr:   true,
		},
		{
			name:      "contains special chars",
			tableName: "test@table",
			wantErr:   true,
		},
		{
			name:      "sql injection attempt",
			tableName: "test'; DROP TABLE users; --",
			wantErr:   true,
		},
		{
			name:      "contains dot",
			tableName: "test.table",
			wantErr:   true,
		},
		{
			name:      "contains semicolon",
			tableName: "test;table",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTableName(tt.tableName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTableName(%q) error = %v, wantErr %v", tt.tableName, err, tt.wantErr)
			}
		})
	}
}

// TestQuoteTableName tests the quoteTableName function for all backends.
func TestQuoteTableName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		backend   schema.CacheBackend
		want      string
	}{
		{
			name:      "SQLite backend",
			tableName: "test_table",
			backend:   schema.SQLiteBackend,
			want:      `"test_table"`,
		},
		{
			name:      "MySQL backend",
			tableName: "test_table",
			backend:   schema.MySQLBackend,
			want:      "`test_table`",
		},
		{
			name:      "PostgreSQL backend",
			tableName: "test_table",
			backend:   schema.PostgreSQLBackend,
			want:      `"test_table"`,
		},
		{
			name:      "None backend defaults to SQLite style",
			tableName: "test_table",
			backend:   schema.NoneBackend,
			want:      `"test_table"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteTableName(tt.tableName, tt.backend)
			if got != tt.want {
				t.Errorf("quoteTableName(%q, %q) = %q, want %q", tt.tableName, tt.backend, got, tt.want)
			}
		})
	}
}

// TestSQLiteBackendOperations tests the full lifecycle of SQLite backend operations.
func TestSQLiteBackendOperations(t *testing.T) {
	// Use the default database path for tests
	dbPath := GetDBFilePath()
	defer func() { _ = os.Remove(dbPath) }() // Clean up after all subtests

	t.Run("set and get operations", func(t *testing.T) {
		// Clean up before test
		_ = os.Remove(dbPath)

		store, err := NewCacheStore("test_table", schema.SQLiteBackend, "")
		if err != nil {
			t.Fatalf("Failed to create SQLite store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Test Set operation
		testKey := "test_key"
		testValue := []byte("test_value_data")
		testVersion := 1
		testTimestamp := int64(1234567890)

		err = store.Set(testKey, testValue, testVersion, testTimestamp)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Test Get operation
		value, version, timestamp, err := store.Get(testKey)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if string(value) != string(testValue) {
			t.Errorf("Get value = %q, want %q", string(value), string(testValue))
		}
		if version != testVersion {
			t.Errorf("Get version = %d, want %d", version, testVersion)
		}
		if timestamp != testTimestamp {
			t.Errorf("Get timestamp = %d, want %d", timestamp, testTimestamp)
		}
	})

	t.Run("upsert behavior", func(t *testing.T) {
		// Clean up before test
		_ = os.Remove(dbPath)

		store, err := NewCacheStore("test_table", schema.SQLiteBackend, "")
		if err != nil {
			t.Fatalf("Failed to create SQLite store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Insert initial value
		testKey := "upsert_key"
		err = store.Set(testKey, []byte("initial_value"), 1, 1000)
		if err != nil {
			t.Fatalf("Initial Set failed: %v", err)
		}

		// Update with new value
		err = store.Set(testKey, []byte("updated_value"), 2, 2000)
		if err != nil {
			t.Fatalf("Update Set failed: %v", err)
		}

		// Verify updated value
		value, version, timestamp, err := store.Get(testKey)
		if err != nil {
			t.Fatalf("Get after update failed: %v", err)
		}

		if string(value) != "updated_value" {
			t.Errorf("After upsert, value = %q, want %q", string(value), "updated_value")
		}
		if version != 2 {
			t.Errorf("After upsert, version = %d, want %d", version, 2)
		}
		if timestamp != 2000 {
			t.Errorf("After upsert, timestamp = %d, want %d", timestamp, 2000)
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		// Clean up before test
		_ = os.Remove(dbPath)

		store, err := NewCacheStore("test_table", schema.SQLiteBackend, "")
		if err != nil {
			t.Fatalf("Failed to create SQLite store: %v", err)
		}
		defer func() { _ = store.Close() }()

		_, _, _, err = store.Get("non_existent_key")
		if err != sql.ErrNoRows {
			t.Errorf("Get non-existent key error = %v, want %v", err, sql.ErrNoRows)
		}
	})

	t.Run("multiple keys", func(t *testing.T) {
		// Clean up before test
		_ = os.Remove(dbPath)

		store, err := NewCacheStore("test_table", schema.SQLiteBackend, "")
		if err != nil {
			t.Fatalf("Failed to create SQLite store: %v", err)
		}
		defer func() { _ = store.Close() }()

		// Set multiple keys
		keys := []string{"key1", "key2", "key3"}
		for i, key := range keys {
			err := store.Set(key, []byte("value"+key), i+1, int64(1000+i))
			if err != nil {
				t.Fatalf("Set %s failed: %v", key, err)
			}
		}

		// Verify all keys can be retrieved
		for i, key := range keys {
			value, version, timestamp, err := store.Get(key)
			if err != nil {
				t.Fatalf("Get %s failed: %v", key, err)
			}
			expectedValue := "value" + key
			if string(value) != expectedValue {
				t.Errorf("Get %s value = %q, want %q", key, string(value), expectedValue)
			}
			if version != i+1 {
				t.Errorf("Get %s version = %d, want %d", key, version, i+1)
			}
			if timestamp != int64(1000+i) {
				t.Errorf("Get %s timestamp = %d, want %d", key, timestamp, int64(1000+i))
			}
		}
	})
}

// TestGetPlaceholder tests the getPlaceholder method for different backends.
func TestGetPlaceholder(t *testing.T) {
	tests := []struct {
		name    string
		backend schema.CacheBackend
		want    string
	}{
		{
			name:    "SQLite backend",
			backend: schema.SQLiteBackend,
			want:    "?",
		},
		{
			name:    "MySQL backend",
			backend: schema.MySQLBackend,
			want:    "?",
		},
		{
			name:    "PostgreSQL backend",
			backend: schema.PostgreSQLBackend,
			want:    "$1",
		},
		{
			name:    "None backend",
			backend: schema.NoneBackend,
			want:    "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &CacheStoreImpl{
				backend: tt.backend,
			}
			got := store.getPlaceholder()
			if got != tt.want {
				t.Errorf("getPlaceholder() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetUpsertQuery tests the getUpsertQuery method for different backends.
func TestGetUpsertQuery(t *testing.T) {
	tests := []struct {
		name         string
		backend      schema.CacheBackend
		tableName    string
		wantContains []string
	}{
		{
			name:      "SQLite backend",
			backend:   schema.SQLiteBackend,
			tableName: "test_table",
			wantContains: []string{
				"INSERT OR REPLACE",
				`"test_table"`,
			},
		},
		{
			name:      "MySQL backend",
			backend:   schema.MySQLBackend,
			tableName: "test_table",
			wantContains: []string{
				"INSERT INTO",
				"ON DUPLICATE KEY UPDATE",
				"`test_table`",
			},
		},
		{
			name:      "PostgreSQL backend",
			backend:   schema.PostgreSQLBackend,
			tableName: "test_table",
			wantContains: []string{
				"INSERT INTO",
				"ON CONFLICT",
				"DO UPDATE SET",
				`"test_table"`,
				"$1", "$2", "$3", "$4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &CacheStoreImpl{
				backend:   tt.backend,
				tableName: tt.tableName,
			}
			got := store.getUpsertQuery()
			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("getUpsertQuery() = %q, should contain %q", got, want)
				}
			}
		})
	}
}

// TestGetCreateTableQuery tests the getCreateTableQuery function for different backends.
func TestGetCreateTableQuery(t *testing.T) {
	tests := []struct {
		name         string
		backend      schema.CacheBackend
		tableName    string
		wantContains []string
	}{
		{
			name:      "SQLite backend",
			backend:   schema.SQLiteBackend,
			tableName: "test_table",
			wantContains: []string{
				"CREATE TABLE IF NOT EXISTS",
				`"test_table"`,
				"cache_key TEXT PRIMARY KEY",
				"cache_value BLOB",
				"cache_version INTEGER",
				"cache_timestamp INTEGER",
			},
		},
		{
			name:      "MySQL backend",
			backend:   schema.MySQLBackend,
			tableName: "test_table",
			wantContains: []string{
				"CREATE TABLE IF NOT EXISTS",
				"`test_table`",
				"cache_key VARCHAR(255) PRIMARY KEY",
				"cache_value BLOB",
				"cache_version INT",
				"cache_timestamp BIGINT",
			},
		},
		{
			name:      "PostgreSQL backend",
			backend:   schema.PostgreSQLBackend,
			tableName: "test_table",
			wantContains: []string{
				"CREATE TABLE IF NOT EXISTS",
				`"test_table"`,
				"cache_key TEXT PRIMARY KEY",
				"cache_value BYTEA",
				"cache_version INTEGER",
				"cache_timestamp BIGINT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCreateTableQuery(tt.tableName, tt.backend)
			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("getCreateTableQuery() = %q, should contain %q", got, want)
				}
			}
		})
	}
}

// TestNewCacheStoreErrors tests error scenarios in NewCacheStore.
func TestNewCacheStoreErrors(t *testing.T) {
	t.Run("invalid table name", func(t *testing.T) {
		_, err := NewCacheStore("invalid-name", schema.SQLiteBackend, "")
		if err == nil {
			t.Fatal("Expected error for invalid table name")
		}
	})

	t.Run("empty table name", func(t *testing.T) {
		_, err := NewCacheStore("", schema.SQLiteBackend, "")
		if err == nil {
			t.Fatal("Expected error for empty table name")
		}
	})

	t.Run("unsupported backend", func(t *testing.T) {
		_, err := NewCacheStore("test_table", schema.CacheBackend("unsupported"), "")
		if err == nil {
			t.Fatal("Expected error for unsupported backend")
		}
	})
}

// TestClearCache tests the ClearCache function.
func TestClearCache(t *testing.T) {
	t.Run("SQLite backend", func(t *testing.T) {
		// Create a temporary test database in a temp directory
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test_clear.db")

		// Create the database file directly with sql.Open
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		// Create a simple table
		_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
		_ = db.Close()

		// Verify file exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Fatal("Database file should exist before ClearCache")
		}

		// Clear the cache
		err = ClearCache(schema.SQLiteBackend, dbPath, "")
		if err != nil {
			t.Fatalf("ClearCache failed: %v", err)
		}

		// Verify file is removed
		if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
			t.Error("Database file should be removed after ClearCache")
		}
	})

	t.Run("SQLite backend - non-existent file", func(t *testing.T) {
		// Clearing non-existent file should not error
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "non_existent.db")
		err := ClearCache(schema.SQLiteBackend, dbPath, "")
		if err != nil {
			t.Errorf("ClearCache on non-existent file should not error: %v", err)
		}
	})

	t.Run("NoneBackend", func(t *testing.T) {
		// NoneBackend should be no-op
		err := ClearCache(schema.NoneBackend, "", "")
		if err != nil {
			t.Errorf("ClearCache with NoneBackend should not error: %v", err)
		}
	})

	t.Run("empty dbFilePath for SQLite", func(t *testing.T) {
		err := ClearCache(schema.SQLiteBackend, "", "")
		if err == nil {
			t.Error("Expected error for empty dbFilePath with SQLite backend")
		}
	})

	t.Run("unsupported backend", func(t *testing.T) {
		err := ClearCache(schema.CacheBackend("unsupported"), "", "")
		if err == nil {
			t.Error("Expected error for unsupported backend")
		}
	})
}

// TestCacheStoreManagerConcurrency tests concurrent access to CacheStoreManager.
func TestCacheStoreManagerConcurrency(t *testing.T) {
	dbPath := GetDBFilePath()
	defer func() { _ = os.Remove(dbPath) }()

	// Clean up before test
	_ = os.Remove(dbPath)

	initOnce = sync.Once{}
	closeOnce = sync.Once{}

	err := InitCaching(schema.SQLiteBackend, "")
	if err != nil {
		t.Fatalf("InitCaching failed: %v", err)
	}
	defer CloseCaching()

	// Concurrently access the manager
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer func() { done <- true }()
			store := Manager.GetActivityStore()
			if store == nil {
				t.Errorf("Goroutine %d: GetActivityStore returned nil", id)
				return
			}
			// Perform some operations
			key := "concurrent_key"
			err := store.Set(key, []byte("value"), 1, int64(1000+id))
			if err != nil {
				t.Errorf("Goroutine %d: Set failed: %v", id, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}
}

// Helper function to check if a string contains a substring.
// TestInitCachingErrors tests error handling in InitCaching.
func TestInitCachingErrors(t *testing.T) {
	t.Run("invalid table name during init", func(t *testing.T) {
		// This test verifies that errors during store creation are properly propagated
		// We can't easily test this without modifying activityTable constant,
		// but we can test with invalid backend combinations

		// Reset for clean test
		initOnce = sync.Once{}
		closeOnce = sync.Once{}
		defer func() {
			// Clean up
			initOnce = sync.Once{}
			closeOnce = sync.Once{}
		}()

		// Try to init with an invalid connection string for MySQL
		// This should fail during database connection
		err := InitCaching(schema.MySQLBackend, "invalid://connection")
		if err == nil {
			t.Error("Expected error for invalid MySQL connection string")
		}
	})
}

// TestSQLiteCloseNil tests closing a nil database.
func TestSQLiteCloseNil(t *testing.T) {
	store := &CacheStoreImpl{
		db:        nil,
		tableName: "test",
		backend:   schema.NoneBackend,
	}

	err := store.Close()
	if err != nil {
		t.Errorf("Close on nil db should not error: %v", err)
	}
}

// TestValidateTableNameRegexError tests regex error handling (edge case).
func TestValidateTableNameRegexError(t *testing.T) {
	// The current implementation uses a simple regex that shouldn't error,
	// but we test the function with various edge cases

	// Very long table name
	var sb strings.Builder
	for range 1000 {
		sb.WriteString("a")
	}
	longName := sb.String()
	err := validateTableName(longName)
	if err != nil {
		t.Errorf("Long valid table name should not error: %v", err)
	}

	// Unicode characters
	err = validateTableName("test_表")
	if err == nil {
		t.Error("Unicode characters should be rejected")
	}
}

// TestCacheStoreImplGetWithNilDB tests Get with nil database.
func TestCacheStoreImplGetWithNilDB(t *testing.T) {
	store := &CacheStoreImpl{
		db:      nil,
		backend: schema.NoneBackend,
	}

	_, _, _, err := store.Get("test_key")
	if err != sql.ErrNoRows {
		t.Errorf("Get with nil db should return sql.ErrNoRows, got: %v", err)
	}
}

// TestCacheStoreImplSetWithNilDB tests Set with nil database.
func TestCacheStoreImplSetWithNilDB(t *testing.T) {
	store := &CacheStoreImpl{
		db:      nil,
		backend: schema.NoneBackend,
	}

	err := store.Set("test_key", []byte("value"), 1, 1000)
	if err != nil {
		t.Errorf("Set with nil db (NoneBackend) should not error: %v", err)
	}
}

// TestMultipleInitCachingCalls tests that InitCaching can be called multiple times safely.
func TestMultipleInitCachingCalls(t *testing.T) {
	dbPath := GetDBFilePath()
	defer func() { _ = os.Remove(dbPath) }()

	// Clean up before test
	_ = os.Remove(dbPath)

	// Reset sync.Once for this test
	initOnce = sync.Once{}
	closeOnce = sync.Once{}

	// First call should succeed
	err := InitCaching(schema.SQLiteBackend, "")
	if err != nil {
		t.Fatalf("First InitCaching call failed: %v", err)
	}

	// Subsequent calls should be no-ops (due to sync.Once) and not error
	err = InitCaching(schema.SQLiteBackend, "")
	if err != nil {
		t.Errorf("Second InitCaching call should not error: %v", err)
	}

	err = InitCaching(schema.MySQLBackend, "different:connection@string")
	if err != nil {
		t.Errorf("Third InitCaching call (different backend) should not error: %v", err)
	}

	// Verify the store is still the SQLite one from the first call
	store := Manager.GetActivityStore()
	if store == nil {
		t.Fatal("Store should not be nil")
	}

	// Close
	CloseCaching()
}
