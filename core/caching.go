package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// currentCacheVersion defines the version of the cache schema
const currentCacheVersion = 1

// cachedAggregateActivity - Simplified and validated using DB columns
func cachedAggregateActivity(ctx context.Context, cfg *internal.Config, client internal.GitClient, mgr internal.PersistenceManager) (*schema.AggregateOutput, error) {
	activity := mgr.GetActivityStore()
	if activity == nil {
		// Fallback to direct computation
		return aggregateActivity(ctx, cfg, client)
	}

	key := generateCacheKey(ctx, cfg, client)

	// Check for cache hit
	if result := checkCacheHit(activity, key); result != nil {
		return result, nil
	}

	// Cache miss: compute and store
	return computeAndStore(ctx, cfg, client, activity, key)
}

// checkCacheHit attempts to retrieve and validate a cached result
func checkCacheHit(activity internal.PersistenceStore, key string) *schema.AggregateOutput {
	data, version, ts, err := activity.Get(key)
	if err != nil {
		return nil // Cache miss
	}

	// Validate version and staleness
	if version == currentCacheVersion {
		entryTimestamp := time.Unix(ts, 0)
		if time.Since(entryTimestamp) <= 7*24*time.Hour {
			var result schema.AggregateOutput
			if err := json.Unmarshal(data, &result); err == nil {
				return &result // Cache hit
			}
		}
	}

	return nil // Cache miss (stale or version mismatch)
}

// computeAndStore computes the result and stores it in cache
func computeAndStore(ctx context.Context, cfg *internal.Config, client internal.GitClient, activity internal.PersistenceStore, key string) (*schema.AggregateOutput, error) {
	result, err := aggregateActivity(ctx, cfg, client)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		_ = activity.Set(key, data, currentCacheVersion, time.Now().Unix())
	}

	return result, nil
}

// generateCacheKey creates a unique key based on analysis parameters
func generateCacheKey(ctx context.Context, cfg *internal.Config, client internal.GitClient) string {
	// Use canonical helpers from internal.Config to ensure consistent time granularity
	startHour := cfg.GetAnalysisStartTime()
	endHour := cfg.GetAnalysisEndTime()

	// Include repo hash to invalidate cache when repository state changes
	repoHash, err := client.GetRepoHash(ctx, cfg.RepoPath)
	if err != nil {
		repoHash = ""
	}

	key := fmt.Sprintf("%s:%s:%s:%d:%d:%s",
		cfg.RepoPath,
		cfg.Mode,
		cfg.Lookback,
		startHour.Unix(),
		endHour.Unix(),
		repoHash,
	)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
