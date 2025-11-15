package internal

import (
	"fmt"
	"sync"

	"github.com/huangsam/hotspot/schema"
)

// activityTable is the name of the table for activity caching.
const activityTable = "activity_cache"

// Global Manager instance for main logic
var (
	Manager   = &PersistStoreManager{}
	initOnce  sync.Once
	closeOnce sync.Once
)

// InitPersistence uses sync.Once to safely initialize the global stores with the given backend.
func InitPersistence(backend schema.CacheBackend, connStr string) error {
	var initErr error

	initOnce.Do(func() {
		// This function body runs exactly once, even with concurrent calls.
		var err error

		// Initialize Activity Store with the specified backend
		activityPersistStore, err := NewPersistStore(activityTable, backend, connStr)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize activity persistence: %w", err)
			return
		}

		// Assign to global manager
		Manager.activity = activityPersistStore
	})

	// After once.Do, initErr will contain any error from the initialization block.
	return initErr
}

// ClosePersistence should be called on application shutdown.
func ClosePersistence() { // called in main defer
	closeOnce.Do(func() {
		Manager.Lock()
		defer Manager.Unlock()
		if Manager.activity != nil && Manager.activity.db != nil {
			_ = Manager.activity.db.Close()
		}
	})
}
