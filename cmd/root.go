package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// All linker flags will be set by goreleaser infra at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// rootCtx is the root context for all operations.
var rootCtx = context.Background()

// cfg will hold the validated, final configuration.
var cfg = &contract.Config{}

// input holds the raw, unvalidated configuration from all sources (file, env, flags).
// Viper will unmarshal into this struct.
var input = &contract.ConfigRawInput{}

// profile holds profiling configuration.
var profile = &contract.ProfileConfig{}

// cacheManager is the global persistence manager instance.
var cacheManager contract.CacheManager

// startProfiling starts CPU and memory profiling if enabled.
func startProfiling() error {
	if !profile.Enabled {
		return nil
	}

	// Start CPU profiling
	cpuFile, err := os.Create(profile.Prefix + ".cpu.prof")
	if err != nil {
		return fmt.Errorf("could not create CPU profile: %w", err)
	}
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return fmt.Errorf("could not start CPU profiling: %w", err)
	}

	// Memory profiling will be captured at the end
	_, err = fmt.Fprintf(os.Stdout, "Profiling enabled. CPU profile: %s.cpu.prof, Memory profile: %s.mem.prof\n", profile.Prefix, profile.Prefix)
	return err
}

// stopProfiling stops profiling and writes memory profile.
func stopProfiling() error {
	if !profile.Enabled {
		return nil
	}

	pprof.StopCPUProfile()

	// Write memory profile
	memFile, err := os.Create(profile.Prefix + ".mem.prof")
	if err != nil {
		return fmt.Errorf("could not create memory profile: %w", err)
	}
	defer func() { _ = memFile.Close() }()

	if err := pprof.WriteHeapProfile(memFile); err != nil {
		return fmt.Errorf("could not write memory profile: %w", err)
	}

	_, err = fmt.Fprintf(os.Stdout, "Profiling complete. Use 'go tool pprof %s.cpu.prof' to analyze.\n", profile.Prefix)
	return err
}

// rootCmd is the command-line entrypoint for all other commands.
var rootCmd = &cobra.Command{
	Use:                "hotspot",
	Short:              "Analyze Git repository activity to find code hotspots.",
	Long:               `Hotspot cuts through Git history to show you which files and folders are your greatest risk.`,
	Version:            version,
	SilenceErrors:      true,
	SilenceUsage:       true,
	DisableSuggestions: true,
	Run: func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	},
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Check if a specific config file is provided
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Set config file name and paths
		viper.SetConfigName(".hotspot") // Name of config file (without extension)
		viper.SetConfigType("yaml")     // We'll use YAML format
		viper.AddConfigPath(".")        // Look in the current directory
		viper.AddConfigPath("$HOME")    // Look in the home directory
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("HOTSPOT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Read in environment variables that match

	// Set defaults in Viper
	viper.SetDefault("limit", contract.DefaultResultLimit)
	viper.SetDefault("workers", contract.DefaultWorkers)
	viper.SetDefault("mode", schema.HotMode)
	viper.SetDefault("precision", contract.DefaultPrecision)
	viper.SetDefault("output", schema.TextOut)
	viper.SetDefault("lookback", "6 months")
	viper.SetDefault("cache-backend", schema.SQLiteBackend)
	viper.SetDefault("cache-db-connect", "")
	viper.SetDefault("analysis-backend", "")
	viper.SetDefault("analysis-db-connect", "")
	viper.SetDefault("color", "yes")
}

// sharedSetup unmarshals config and runs validation.
func sharedSetup(ctx context.Context, _ *cobra.Command, args []string) error {
	// Handle profiling flag
	profilePrefix := viper.GetString("profile")
	if err := contract.ProcessProfilingConfig(profile, profilePrefix); err != nil {
		return fmt.Errorf("failed to process profiling config: %w", err)
	}
	if profile.Enabled {
		if err := startProfiling(); err != nil {
			return fmt.Errorf("failed to start profiling: %w", err)
		}
	}

	// 1. Read config file. This merges defaults, file, env, and flags.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, which is fine; we'll use defaults/env/flags.
	}

	// 2. Unmarshal all resolved values from Viper into our raw input struct.
	if err := viper.Unmarshal(input); err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
	}

	// 3. Handle positional arguments (which Viper doesn't do).
	if len(args) == 1 {
		input.RepoPathStr = args[0]
	} else {
		input.RepoPathStr = "."
	}

	// 4. Run all validation and complex parsing.
	// This function now populates the global 'cfg' from 'input'.
	client := contract.NewLocalGitClient()
	if err := contract.ProcessAndValidate(ctx, cfg, client, input); err != nil {
		return err
	}

	// 5. Initialize persistence layer with validated config
	if err := iocache.InitStores(cfg.CacheBackend, cfg.CacheDBConnect, cfg.AnalysisBackend, cfg.AnalysisDBConnect); err != nil {
		return fmt.Errorf("failed to initialize persistence: %w", err)
	}

	return nil
}

// sharedSetupWrapper wraps sharedSetup to provide context for Cobra's PreRunE.
func sharedSetupWrapper(cmd *cobra.Command, args []string) error {
	return sharedSetup(rootCtx, cmd, args)
}

// loadConfigFile handles config file loading logic common to all setup functions.
func loadConfigFile() error {
	// Handle config file
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".hotspot")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME")
	}

	// Load config file if present
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// SetCacheManager sets the global cache manager.
func SetCacheManager(mgr contract.CacheManager) {
	cacheManager = mgr
}

// StopProfiling stops profiling if enabled.
func StopProfiling() error {
	return stopProfiling()
}
