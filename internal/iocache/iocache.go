// Package iocache is for caching I/O calls.
package iocache

import (
	"sync"

	"github.com/huangsam/hotspot/internal/contract"
)

// CacheStoreManager manages multiple CacheStore instances.
type CacheStoreManager struct {
	sync.RWMutex // Protects the store pointers during initialization
	activity     contract.CacheStore
	analysis     contract.AnalysisStore
}

var _ contract.CacheManager = &CacheStoreManager{} // Compile-time check

// GetActivityStore returns the activity CacheStore.
func (mgr *CacheStoreManager) GetActivityStore() contract.CacheStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.activity
}

// GetAnalysisStore returns the analysis AnalysisStore.
func (mgr *CacheStoreManager) GetAnalysisStore() contract.AnalysisStore {
	mgr.RLock()
	defer mgr.RUnlock()
	return mgr.analysis
}
