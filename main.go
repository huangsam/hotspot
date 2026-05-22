// Package main invokes entrypoint logic for hotspot CLI.
package main

import (
	_ "embed"

	"github.com/huangsam/hotspot/cmd"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/logger"
)

//go:embed AGENTS.md
var agentsDoc string

// main starts the execution of the logic.
func main() {
	cmd.AgentsDoc = agentsDoc

	// Set the global caching manager (will be initialized in sharedSetup)
	cmd.SetCacheManager(iocache.Manager)

	// Run the application logic in a closure to ensure defers are executed before exit
	err := func() error {
		defer iocache.CloseCaching()
		defer func() {
			if err := cmd.StopProfiling(); err != nil {
				// Use Error instead of Fatal here so we don't interrupt the shutdown sequence
				logger.Error("Error stopping profiling: %v", err)
			}
		}()

		return cmd.Execute()
	}()
	if err != nil {
		logger.Fatal("Application error: %v", err)
	}
}
