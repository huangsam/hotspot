// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	"github.com/huangsam/hotspot/cmd"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
)

// main starts the execution of the logic.
func main() {
	// Set the global caching manager (will be initialized in sharedSetup)
	cmd.SetCacheManager(iocache.Manager)

	defer func() {
		// Close caching on exit
		iocache.CloseCaching()

		if err := cmd.StopProfiling(); err != nil {
			contract.LogFatal("Error stopping profiling", err)
		}
	}()

	if err := cmd.Execute(); err != nil {
		contract.LogFatal("Error starting CLI", err)
	}
}
