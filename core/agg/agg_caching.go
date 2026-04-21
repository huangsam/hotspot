package agg

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
)

// currentCacheVersion defines the version of the cache schema.
const currentCacheVersion = 3

// CachedAggregateActivity - Simplified and validated using DB columns.
func CachedAggregateActivity(
	ctx context.Context,
	gitSettings config.GitSettings,
	compareSettings config.ComparisonSettings,
	client git.Client,
	mgr iocache.CacheManager,
	urn string, // Added URN
) (*schema.AggregateOutput, error) {
	activity := mgr.GetActivityStore()
	if activity == nil {
		// Fallback to direct computation
		return aggregateActivity(ctx, gitSettings, client)
	}

	key := generateCacheKey(ctx, gitSettings, compareSettings, client, urn)

	// Check for cache hit
	if result := checkCacheHit(activity, key); result != nil {
		return result, nil
	}

	// Cache miss: compute and store
	return computeAndStore(ctx, gitSettings, client, activity, key)
}

// checkCacheHit attempts to retrieve and validate a cached result.
func checkCacheHit(activity iocache.CacheStore, key string) *schema.AggregateOutput {
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

// computeAndStore computes the result and stores it in cache.
func computeAndStore(ctx context.Context, gitSettings config.GitSettings, client git.Client, activity iocache.CacheStore, key string) (*schema.AggregateOutput, error) {
	result, err := aggregateActivity(ctx, gitSettings, client)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if data, err := json.Marshal(result); err == nil {
		_ = activity.Set(key, data, currentCacheVersion, time.Now().Unix())
	}

	return result, nil
}

// generateCacheKey creates a unique key based on analysis parameters.
func generateCacheKey(ctx context.Context, gitSettings config.GitSettings, compareSettings config.ComparisonSettings, client git.Client, urn string) string {
	// Truncate to the caching granularity
	startHour := gitSettings.GetStartTime().Truncate(config.CacheGranularity)
	endHour := gitSettings.GetEndTime().Truncate(config.CacheGranularity)

	// Include repo hash to invalidate cache when repository state changes
	repoHash, err := client.GetRepoHash(ctx, gitSettings.GetRepoPath())
	if err != nil {
		repoHash = ""
	}

	// Use RepoURN if available for path-independent caching.
	// Fall back to resolving the URN to ensure consistency even for legacy callers.
	repoID := urn
	if repoID == "" {
		repoID = git.ResolveURN(ctx, client, gitSettings.GetRepoPath())
	}

	key := fmt.Sprintf("%s:%s:%d:%d:%d:%s",
		repoID,
		gitSettings.GetPathFilter(),
		int64(compareSettings.GetLookback()),
		startHour.Unix(),
		endHour.Unix(),
		repoHash,
	)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
