// Package contract provides interfaces and shared utilities for the Hotspot CLI's internal architecture.
package contract

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
