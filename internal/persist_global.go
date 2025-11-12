package internal

import (
	"fmt"
	"sync"
)

// activityTable is the name of the SQLite table for activity caching.
const activityTable = "activity_cache"

// Global Manager instance for main logic
var (
	Manager   = &PersistStoreManager{}
	initOnce  sync.Once
	closeOnce sync.Once
)

// InitPersistence uses sync.Once to safely initialize the global stores.
func InitPersistence() error { // called in main entrypoint
	var initErr error

	initOnce.Do(func() {
		// This function body runs exactly once, even with concurrent calls.
		var err error

		// Initialize Activity Store
		activityPersistStore, err := NewPersistStore(activityTable)
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
		if Manager.activity != nil {
			_ = Manager.activity.db.Close()
		}
	})
}
