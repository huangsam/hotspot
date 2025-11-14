package core

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// currentCacheVersion defines the version of the cache schema
const currentCacheVersion = 1

// CachedAggregateActivity - Simplified and validated using DB columns
func CachedAggregateActivity(ctx context.Context, cfg *internal.Config, client internal.GitClient, mgr internal.PersistenceManager) (*schema.AggregateOutput, error) {
	activity := mgr.GetActivityStore()
	if activity == nil {
		// Fallback to direct computation
		return aggregateActivity(ctx, cfg, client)
	}

	key := generateCacheKey(cfg)

	// Try to get from SQLite store
	// Get raw data and metadata from DB columns
	data, version, ts, err := activity.Get(key)

	if err == nil {
		// 1. Validate version (DB column)
		if version == currentCacheVersion {
			// 2. Validate staleness (DB column)
			entryTimestamp := time.Unix(ts, 0)
			if time.Since(entryTimestamp) <= 7*24*time.Hour {
				// Cache Hit: Unmarshal the raw result data
				var result schema.AggregateOutput
				if err := json.Unmarshal(data, &result); err == nil {
					return &result, nil
				}
			}
			// Entry is stale - treat as cache miss, continue to computation
		}
		// Version mismatch - treat as cache miss, continue to computation
	}

	// Compute if cache miss or validation/unmarshal failed
	result, err := aggregateActivity(ctx, cfg, client)
	if err != nil {
		return nil, err
	}

	// Store raw result in SQLite, using columns for metadata
	if data, err := json.Marshal(result); err == nil {
		_ = activity.Set(key, data, currentCacheVersion, time.Now().Unix())
	}

	return result, nil
}

// generateCacheKey creates a unique key based on analysis parameters
func generateCacheKey(cfg *internal.Config) string {
	// Use canonical helpers from internal.Config to ensure consistent time granularity
	startHour := cfg.GetAnalysisStartTime()
	endHour := cfg.GetAnalysisEndTime()

	// Include repo hash to invalidate cache when repository state changes
	repoHash := getRepoHash(cfg.RepoPath)

	key := fmt.Sprintf("%s:%s:%s:%d:%d:%s",
		cfg.RepoPath,
		cfg.Mode,
		cfg.Lookback,
		startHour.Unix(),
		endHour.Unix(),
		repoHash,
	)
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}

// getRepoHash gets the current Git HEAD hash for the repository
func getRepoHash(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Trim whitespace/newline to ensure consistent hash across environments
	return strings.TrimSpace(string(output))
}
