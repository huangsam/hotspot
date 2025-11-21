package iocache

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestCaching(t *testing.T) {
	t.Run("single setup", func(t *testing.T) {
		// Clean up any existing test database
		testDBPath := GetDBFilePath()
		defer func() { _ = os.Remove(testDBPath) }()
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Test initialization with SQLite backend
		err := InitStores(schema.SQLiteBackend, "", "", "")
		assert.NoError(t, err, "Failed to initialize persistence")

		// Test that Manager is accessible
		assert.NotNil(t, Manager, "Manager should not be nil")

		// Test that stores are accessible
		assert.NotNil(t, Manager.GetActivityStore(), "Activity store should not be nil")

		// Test cleanup
		CloseCaching()

		// Verify database file was created
		_, err = os.Stat(testDBPath)
		assert.False(t, os.IsNotExist(err), "Database file should be created")
	})

	t.Run("idempotent setup", func(t *testing.T) {
		// Clean up any existing test database
		testDBPath := GetDBFilePath()
		defer func() { _ = os.Remove(testDBPath) }()
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Multiple initializations should be safe (sync.Once)
		err1 := InitStores(schema.SQLiteBackend, "", "", "")
		err2 := InitStores(schema.SQLiteBackend, "", "", "")
		err3 := InitStores(schema.SQLiteBackend, "", "", "")

		assert.NoError(t, err1, "First init should not fail")
		assert.NoError(t, err2, "Second init should not fail")
		assert.NoError(t, err3, "Third init should not fail")

		// Multiple closes should be safe (sync.Once)
		CloseCaching()
		CloseCaching()
		CloseCaching()
	})

	t.Run("none backend", func(t *testing.T) {
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Test initialization with None backend (no database)
		err := InitStores(schema.NoneBackend, "", "", "")
		assert.NoError(t, err, "Failed to initialize persistence with none backend")

		// Test that Manager is accessible
		assert.NotNil(t, Manager, "Manager should not be nil")

		// Test that stores are accessible
		store := Manager.GetActivityStore()
		assert.NotNil(t, store, "Activity store should not be nil")

		// Test cleanup (should be safe even with no DB)
		CloseCaching()
	})

	t.Run("none backend operations", func(t *testing.T) {
		// Create a none backend store directly
		store, err := NewCacheStore("test_table", schema.NoneBackend, "")
		assert.NoError(t, err, "Failed to create none backend store")

		// Test Get returns error (no data)
		_, _, _, err = store.Get("test_key")
		assert.Error(t, err, "Expected error from Get on none backend")

		// Test Set is no-op (no error)
		err = store.Set("test_key", []byte("test_value"), 1, 123456789)
		assert.NoError(t, err, "Set should not error on none backend")

		// Verify Get still returns error after Set (no-op)
		_, _, _, err = store.Get("test_key")
		assert.Error(t, err, "Expected error from Get after Set on none backend")

		// Close is safe
		err = store.Close()
		assert.NoError(t, err, "Close should not error on none backend")
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
			if tt.wantErr {
				assert.Error(t, err, "validateTableName should error for %q", tt.tableName)
			} else {
				assert.NoError(t, err, "validateTableName should not error for %q", tt.tableName)
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
			assert.Equal(t, tt.want, got, "quoteTableName(%q, %q)", tt.tableName, tt.backend)
		})
	}
}

// TestSQLiteBackendOperations tests the full lifecycle of SQLite backend operations.
func TestSQLiteBackendOperations(t *testing.T) {
	t.Run("set and get operations", func(t *testing.T) {
		store, err := NewCacheStore("test_table", schema.SQLiteBackend, ":memory:")
		assert.NoError(t, err, "Failed to create SQLite store")
		defer func() { _ = store.Close() }()

		// Test Set operation
		testKey := "test_key"
		testValue := []byte("test_value_data")
		testVersion := 1
		testTimestamp := int64(1234567890)

		err = store.Set(testKey, testValue, testVersion, testTimestamp)
		assert.NoError(t, err, "Set should not fail")

		// Test Get operation
		value, version, timestamp, err := store.Get(testKey)
		assert.NoError(t, err, "Get should not fail")

		assert.Equal(t, string(testValue), string(value), "Get value mismatch")
		assert.Equal(t, testVersion, version, "Get version mismatch")
		assert.Equal(t, testTimestamp, timestamp, "Get timestamp mismatch")
	})

	t.Run("upsert behavior", func(t *testing.T) {
		store, err := NewCacheStore("test_table", schema.SQLiteBackend, ":memory:")
		assert.NoError(t, err, "Failed to create SQLite store")
		defer func() { _ = store.Close() }()

		// Insert initial value
		testKey := "upsert_key"
		err = store.Set(testKey, []byte("initial_value"), 1, 1000)
		assert.NoError(t, err, "Initial Set should not fail")

		// Update with new value
		err = store.Set(testKey, []byte("updated_value"), 2, 2000)
		assert.NoError(t, err, "Update Set should not fail")

		// Verify updated value
		value, version, timestamp, err := store.Get(testKey)
		assert.NoError(t, err, "Get after update should not fail")

		assert.Equal(t, "updated_value", string(value), "After upsert, value mismatch")
		assert.Equal(t, 2, version, "After upsert, version mismatch")
		assert.Equal(t, int64(2000), timestamp, "After upsert, timestamp mismatch")
	})

	t.Run("get non-existent key", func(t *testing.T) {
		store, err := NewCacheStore("test_table", schema.SQLiteBackend, ":memory:")
		assert.NoError(t, err, "Failed to create SQLite store")
		defer func() { _ = store.Close() }()

		_, _, _, err = store.Get("non_existent_key")
		assert.Equal(t, sql.ErrNoRows, err, "Get non-existent key should return sql.ErrNoRows")
	})

	t.Run("multiple keys", func(t *testing.T) {
		store, err := NewCacheStore("test_table", schema.SQLiteBackend, ":memory:")
		assert.NoError(t, err, "Failed to create SQLite store")
		defer func() { _ = store.Close() }()

		// Set multiple keys
		keys := []string{"key1", "key2", "key3"}
		for i, key := range keys {
			err := store.Set(key, []byte("value"+key), i+1, int64(1000+i))
			assert.NoError(t, err, "Set %s should not fail", key)
		}

		// Verify all keys can be retrieved
		for i, key := range keys {
			value, version, timestamp, err := store.Get(key)
			assert.NoError(t, err, "Get %s should not fail", key)
			expectedValue := "value" + key
			assert.Equal(t, expectedValue, string(value), "Get %s value mismatch", key)
			assert.Equal(t, i+1, version, "Get %s version mismatch", key)
			assert.Equal(t, int64(1000+i), timestamp, "Get %s timestamp mismatch", key)
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
			assert.Equal(t, tt.want, got, "getPlaceholder()")
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
				assert.Contains(t, got, want, "getUpsertQuery() should contain %q", want)
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
				assert.Contains(t, got, want, "getCreateTableQuery() should contain %q", want)
			}
		})
	}
}

// TestNewCacheStoreErrors tests error scenarios in NewCacheStore.
func TestNewCacheStoreErrors(t *testing.T) {
	t.Run("invalid table name", func(t *testing.T) {
		_, err := NewCacheStore("invalid-name", schema.SQLiteBackend, "")
		assert.Error(t, err, "Expected error for invalid table name")
	})

	t.Run("empty table name", func(t *testing.T) {
		_, err := NewCacheStore("", schema.SQLiteBackend, "")
		assert.Error(t, err, "Expected error for empty table name")
	})

	t.Run("unsupported backend", func(t *testing.T) {
		_, err := NewCacheStore("test_table", "unsupported", "")
		assert.Error(t, err, "Expected error for unsupported backend")
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
		assert.NoError(t, err, "Failed to create database")
		defer func() { _ = db.Close() }()

		// Create a simple table
		_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
		assert.NoError(t, err, "Failed to create table")

		// Verify file exists
		_, err = os.Stat(dbPath)
		assert.False(t, os.IsNotExist(err), "Database file should exist before ClearCache")

		// Clear the cache
		err = ClearCache(schema.SQLiteBackend, dbPath, "")
		assert.NoError(t, err, "ClearCache should not fail")

		// Verify file is removed
		_, err = os.Stat(dbPath)
		assert.True(t, os.IsNotExist(err), "Database file should be removed after ClearCache")
	})

	t.Run("SQLite backend - non-existent file", func(t *testing.T) {
		// Clearing non-existent file should not error
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "non_existent.db")
		err := ClearCache(schema.SQLiteBackend, dbPath, "")
		assert.NoError(t, err, "ClearCache on non-existent file should not error")
	})

	t.Run("NoneBackend", func(t *testing.T) {
		// NoneBackend should be no-op
		err := ClearCache(schema.NoneBackend, "", "")
		assert.NoError(t, err, "ClearCache with NoneBackend should not error")
	})

	t.Run("empty dbFilePath for SQLite", func(t *testing.T) {
		err := ClearCache(schema.SQLiteBackend, "", "")
		assert.Error(t, err, "Expected error for empty dbFilePath with SQLite backend")
	})

	t.Run("unsupported backend", func(t *testing.T) {
		err := ClearCache("unsupported", "", "")
		assert.Error(t, err, "Expected error for unsupported backend")
	})
}

// TestCacheStoreManagerConcurrency tests concurrent access to CacheStoreManager.
func TestCacheStoreManagerConcurrency(t *testing.T) {
	initOnce = sync.Once{}
	closeOnce = sync.Once{}

	err := InitStores(schema.SQLiteBackend, ":memory:", "", "")
	if err != nil {
		t.Fatalf("InitStores failed: %v", err)
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
// TestInitStoresErrors tests error handling in InitStores.
func TestInitStoresErrors(t *testing.T) {
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
		err := InitStores(schema.MySQLBackend, "invalid://connection", "", "")
		assert.Error(t, err, "Expected error for invalid MySQL connection string")
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
	assert.NoError(t, err, "Close on nil db should not error")
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
	assert.NoError(t, err, "Long valid table name should not error")

	// Unicode character '表' (meaning 'table') is intentionally used here to test that
	// table names with Unicode are rejected. This is not a typo.
	err = validateTableName("test_表")
	assert.Error(t, err, "Unicode characters should be rejected")
}

// TestCacheStoreImplGetWithNilDB tests Get with nil database.
func TestCacheStoreImplGetWithNilDB(t *testing.T) {
	store := &CacheStoreImpl{
		db:      nil,
		backend: schema.NoneBackend,
	}

	_, _, _, err := store.Get("test_key")
	assert.Equal(t, sql.ErrNoRows, err, "Get with nil db should return sql.ErrNoRows")
}

// TestCacheStoreImplSetWithNilDB tests Set with nil database.
func TestCacheStoreImplSetWithNilDB(t *testing.T) {
	store := &CacheStoreImpl{
		db:      nil,
		backend: schema.NoneBackend,
	}

	err := store.Set("test_key", []byte("value"), 1, 1000)
	assert.NoError(t, err, "Set with nil db (NoneBackend) should not error")
}

// TestInitStoresNoneBackend tests that InitStores properly handles NoneBackend
// for both cache and analysis stores, creating no-op implementations.
func TestInitStoresNoneBackend(t *testing.T) {
	// Reset sync.Once for clean test state
	initOnce = sync.Once{}
	closeOnce = sync.Once{}
	defer func() {
		// Clean up
		initOnce = sync.Once{}
		closeOnce = sync.Once{}
	}()

	t.Run("cache backend none", func(t *testing.T) {
		// Reset for this subtest
		initOnce = sync.Once{}
		closeOnce = sync.Once{}

		// Initialize with NoneBackend for cache, in-memory SQLite for analysis
		err := InitStores(schema.NoneBackend, "", schema.SQLiteBackend, ":memory:")
		assert.NoError(t, err, "InitStores with NoneBackend cache should not error")

		// Verify Manager is initialized
		assert.NotNil(t, Manager, "Manager should not be nil")

		// Verify cache store is created (no-op implementation)
		cacheStore := Manager.GetActivityStore()
		assert.NotNil(t, cacheStore, "Activity store should not be nil for NoneBackend")

		// Verify analysis store is created
		analysisStore := Manager.GetAnalysisStore()
		assert.NotNil(t, analysisStore, "Analysis store should not be nil")

		// Test that NoneBackend cache store behaves as no-op
		testKey := "none_cache_test"
		testValue := []byte("test_value")

		// Set should not error
		err = cacheStore.Set(testKey, testValue, 1, 1234567890)
		assert.NoError(t, err, "Set on NoneBackend cache store should not error")

		// Get should return ErrNoRows (no data persisted)
		_, _, _, err = cacheStore.Get(testKey)
		assert.Equal(t, sql.ErrNoRows, err, "Get on NoneBackend cache store should return ErrNoRows")

		CloseCaching()
	})

	t.Run("analysis backend none", func(t *testing.T) {
		// Reset for this subtest
		initOnce = sync.Once{}
		closeOnce = sync.Once{}

		// Initialize with in-memory SQLite for cache, NoneBackend for analysis
		err := InitStores(schema.SQLiteBackend, ":memory:", schema.NoneBackend, "")
		assert.NoError(t, err, "InitStores with NoneBackend analysis should not error")

		// Verify Manager is initialized
		assert.NotNil(t, Manager, "Manager should not be nil")

		// Verify cache store is created
		cacheStore := Manager.GetActivityStore()
		assert.NotNil(t, cacheStore, "Activity store should not be nil")

		// Verify analysis store is created (no-op implementation)
		analysisStore := Manager.GetAnalysisStore()
		assert.NotNil(t, analysisStore, "Analysis store should not be nil for NoneBackend")

		// Test that SQLite cache store works (basic smoke test)
		err = cacheStore.Set("test_key", []byte("test_value"), 1, 1000)
		assert.NoError(t, err, "Set on SQLite cache store should not error")

		CloseCaching()
	})

	t.Run("both backends none", func(t *testing.T) {
		// Reset for this subtest
		initOnce = sync.Once{}
		closeOnce = sync.Once{}

		// Initialize with NoneBackend for both
		err := InitStores(schema.NoneBackend, "", schema.NoneBackend, "")
		assert.NoError(t, err, "InitStores with both NoneBackend should not error")

		// Verify Manager is initialized
		assert.NotNil(t, Manager, "Manager should not be nil")

		// Verify both stores are created (no-op implementations)
		cacheStore := Manager.GetActivityStore()
		assert.NotNil(t, cacheStore, "Activity store should not be nil for NoneBackend")

		analysisStore := Manager.GetAnalysisStore()
		assert.NotNil(t, analysisStore, "Analysis store should not be nil for NoneBackend")

		// Test no-op behavior for cache store
		err = cacheStore.Set("test", []byte("value"), 1, 1000)
		assert.NoError(t, err, "Set on NoneBackend should not error")

		_, _, _, err = cacheStore.Get("test")
		assert.Equal(t, sql.ErrNoRows, err, "Get on NoneBackend should return ErrNoRows")

		CloseCaching()
	})
}

// TestCacheStoreGetStatus tests the GetStatus method for different backends.
func TestCacheStoreGetStatus(t *testing.T) {
	t.Run("SQLite backend with data", func(t *testing.T) {
		store, err := NewCacheStore("test_status_table", schema.SQLiteBackend, ":memory:")
		assert.NoError(t, err, "Failed to create SQLite store")
		defer func() { _ = store.Close() }()

		// Add some test data
		testData := []struct {
			key   string
			value []byte
			ts    int64
		}{
			{"key1", []byte("value1"), 1000},
			{"key2", []byte("value2"), 2000},
			{"key3", []byte("value3"), 1500},
		}

		for _, data := range testData {
			err := store.Set(data.key, data.value, 1, data.ts)
			assert.NoError(t, err, "Set should not fail")
		}

		// Get status
		status, err := store.GetStatus()
		assert.NoError(t, err, "GetStatus should not fail")

		assert.Equal(t, "sqlite", status.Backend, "Backend should be sqlite")
		assert.True(t, status.Connected, "Should be connected")
		assert.Equal(t, 3, status.TotalEntries, "Total entries should be 3")
		assert.Equal(t, time.Unix(2000, 0), status.LastEntryTime, "Last entry time should be 2000")
		assert.Equal(t, time.Unix(1000, 0), status.OldestEntryTime, "Oldest entry time should be 1000")
		assert.Greater(t, status.TableSizeBytes, int64(0), "Table size should be greater than 0")
	})

	t.Run("SQLite backend empty", func(t *testing.T) {
		store, err := NewCacheStore("test_empty_table", schema.SQLiteBackend, ":memory:")
		assert.NoError(t, err, "Failed to create SQLite store")
		defer func() { _ = store.Close() }()

		// Get status without data
		status, err := store.GetStatus()
		assert.NoError(t, err, "GetStatus should not fail")

		assert.Equal(t, "sqlite", status.Backend, "Backend should be sqlite")
		assert.True(t, status.Connected, "Should be connected")
		assert.Equal(t, 0, status.TotalEntries, "Total entries should be 0")
		assert.True(t, status.LastEntryTime.IsZero(), "Last entry time should be zero")
		assert.True(t, status.OldestEntryTime.IsZero(), "Oldest entry time should be zero")
		assert.Equal(t, int64(0), status.TableSizeBytes, "Table size should be 0")
	})

	t.Run("None backend", func(t *testing.T) {
		store, err := NewCacheStore("test_none", schema.NoneBackend, "")
		assert.NoError(t, err, "Failed to create None store")

		// Get status
		status, err := store.GetStatus()
		assert.NoError(t, err, "GetStatus should not fail")

		assert.Equal(t, "none", status.Backend, "Backend should be none")
		assert.False(t, status.Connected, "Should not be connected")
		assert.Equal(t, 0, status.TotalEntries, "Total entries should be 0")
		assert.True(t, status.LastEntryTime.IsZero(), "Last entry time should be zero")
		assert.True(t, status.OldestEntryTime.IsZero(), "Oldest entry time should be zero")
		assert.Equal(t, int64(0), status.TableSizeBytes, "Table size should be 0")
	})
}
