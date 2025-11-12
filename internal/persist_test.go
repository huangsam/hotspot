package internal

import (
	"os"
	"sync"
	"testing"
)

func TestPersistence(t *testing.T) {
	t.Run("single setup", func(t *testing.T) {
		// Clean up any existing test database
		testDBPath := GetDBFilePath()
		defer func() { _ = os.Remove(testDBPath) }()
		initOnce = sync.Once{}  // Reset for test
		closeOnce = sync.Once{} // Reset for test

		// Test initialization
		err := InitPersistence()
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
		err1 := InitPersistence()
		err2 := InitPersistence()
		err3 := InitPersistence()

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
}
