package internal

import (
	"os"
	"sync"
	"testing"

	"github.com/huangsam/hotspot/schema"
)

func TestPersistence(t *testing.T) {
	t.Run("single setup", func(t *testing.T) {
		// Clean up any existing test database
		testDBPath := GetDBFilePath()
		defer func() { _ = os.Remove(testDBPath) }()
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Test initialization with SQLite backend
		err := InitPersistence(schema.SQLiteBackend, "")
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
		ClosePersistence()

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
		err1 := InitPersistence(schema.SQLiteBackend, "")
		err2 := InitPersistence(schema.SQLiteBackend, "")
		err3 := InitPersistence(schema.SQLiteBackend, "")

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
		ClosePersistence()
		ClosePersistence()
		ClosePersistence()
	})

	t.Run("none backend", func(t *testing.T) {
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Test initialization with None backend (no database)
		err := InitPersistence(schema.NoneBackend, "")
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

		// Verify backend is none
		if store.backend != schema.NoneBackend {
			t.Fatalf("Expected backend to be none, got %s", store.backend)
		}

		// Test cleanup (should be safe even with no DB)
		ClosePersistence()
	})

	t.Run("none backend operations", func(t *testing.T) {
		// Create a none backend store directly
		store, err := NewPersistStore("test_table", schema.NoneBackend, "")
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
