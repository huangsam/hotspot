package cmd

import (
	"fmt"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// cacheSetup loads minimal configuration needed for cache operations.
// This is used by commands that need cache access without full shared setup.
func cacheSetup() error {
	if err := loadConfigFile(); err != nil {
		return err
	}

	// Get cache-related config values
	backend := schema.DatabaseBackend(viper.GetString("cache-backend"))
	connStr := viper.GetString("cache-db-connect")

	// Basic validation for database backends
	if err := contract.ValidateDatabaseConnectionString(backend, connStr); err != nil {
		return err
	}

	// Initialize caching with the loaded config (no analysis tracking for cache commands)
	if err := iocache.InitStores(backend, connStr, "", ""); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	cfg.CacheBackend = backend
	cfg.CacheDBConnect = connStr

	return nil
}

// cacheSetupWrapper wraps cacheSetup to provide PreRunE for cache commands.
func cacheSetupWrapper(_ *cobra.Command, _ []string) error {
	return cacheSetup()
}

// cacheCmd focused on cache management.
//
// Note: Cache subcommands use minimal initialization (initCacheConfig) instead of
// the full sharedSetup used by analysis commands. This avoids Git repo validation
// and complex config processing for simple cache operations.
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage Git activity cache (improves performance)",
	Long: `Manage the Git activity cache that speeds up repeated analyses.

Hotspot caches Git log aggregation results to avoid re-parsing history on every run.
This dramatically improves performance when analyzing the same repository multiple times.

Supported backends: SQLite (default), MySQL, PostgreSQL, or None (in-memory)

Subcommands:
  status - Show cache statistics and connection info
  clear  - Remove all cached data

Examples:
  # Check cache status
  hotspot cache status

  # Clear cache after major repository changes
  hotspot cache clear`,
}

// cacheClearCmd clears the cache.
var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all cached Git activity data",
	Long: `Delete all cached Git activity data from the configured backend.

Use this when:
- Repository history was rewritten (rebase, force push)
- Cache may be stale or corrupted
- Testing performance without cache
- Switching analysis time ranges significantly

For SQLite: Deletes the database file
For MySQL/PostgreSQL: Drops the cache table

Examples:
  # Clear SQLite cache (default)
  hotspot cache clear

  # Clear MySQL cache (set connection string via env variable)
  HOTSPOT_CACHE_BACKEND=mysql HOTSPOT_CACHE_DB_CONNECT="..." hotspot cache clear`,
	PreRunE: cacheSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := iocache.ClearCache(cfg.CacheBackend, contract.GetCacheDBFilePath(), cfg.CacheDBConnect); err != nil {
			contract.LogFatal("Failed to clear cache", err)
		}
		fmt.Println("Cache cleared successfully.")
	},
}

// cacheStatusCmd shows cache status.
var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display cache statistics and connection details",
	Long: `Show detailed information about the Git activity cache.

Displays:
- Backend type and connection status
- Total number of cached entries
- Last and oldest cache entry timestamps
- Cache database size

Use this to:
- Verify cache is working and connected
- Monitor cache growth over time
- Check when cache was last updated
- Debug cache-related issues

Examples:
  # Check cache status
  hotspot cache status`,
	PreRunE: cacheSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		status, err := iocache.Manager.GetActivityStore().GetStatus()
		if err != nil {
			contract.LogFatal("Failed to get cache status", err)
		}
		iocache.PrintCacheStatus(status)
	},
}
