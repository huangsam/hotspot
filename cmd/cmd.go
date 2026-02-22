// Package cmd defines the command-line interface for hotspot.
package cmd

import (
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Call initConfig on Cobra's initialization
	cobra.OnInitialize(initConfig)

	// Add primary subcommands to the root command
	rootCmd.AddCommand(filesCmd)
	rootCmd.AddCommand(foldersCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(timeseriesCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(metricsCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(analysisCmd)

	// Add the compare subcommands to the parent compare command
	compareCmd.AddCommand(compareFilesCmd)
	compareCmd.AddCommand(compareFoldersCmd)

	// Add the cache subcommands to the parent cache command
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheStatusCmd)

	// Add the analysis subcommands to the parent analysis command
	analysisCmd.AddCommand(analysisClearCmd)
	analysisCmd.AddCommand(analysisStatusCmd)
	analysisCmd.AddCommand(analysisExportCmd)
	analysisCmd.AddCommand(analysisMigrateCmd)

	// Bind all persistent flags of rootCmd to Viper
	rootCmd.PersistentFlags().Bool("detail", false, "Print per-target metadata (lines of code, size, age)")
	rootCmd.PersistentFlags().String("end", "", "End date in ISO8601 or time ago")
	rootCmd.PersistentFlags().String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore")
	rootCmd.PersistentFlags().StringP("filter", "f", "", "Filter targets by path prefix")
	rootCmd.PersistentFlags().IntP("limit", "l", contract.DefaultResultLimit, "Number of results to display")
	rootCmd.PersistentFlags().String("mode", string(schema.HotMode), "Scoring mode: hot or risk or complexity or stale")
	rootCmd.PersistentFlags().String("output", string(schema.TextOut), "Output format: text or csv or json or parquet")
	rootCmd.PersistentFlags().String("output-file", "", "Optional path to write output to")
	rootCmd.PersistentFlags().Bool("owner", false, "Print per-target owner")
	rootCmd.PersistentFlags().Int("precision", contract.DefaultPrecision, "Decimal precision for numeric columns")
	rootCmd.PersistentFlags().String("profile", "", "Enable profiling and write profiles to files with this prefix")
	rootCmd.PersistentFlags().String("start", "", "Start date in ISO8601 or time ago")
	rootCmd.PersistentFlags().Int("workers", contract.DefaultWorkers, "Number of concurrent workers")
	rootCmd.PersistentFlags().Int("width", 0, "Terminal width override (0 = auto-detect)")
	rootCmd.PersistentFlags().String("cache-backend", string(schema.SQLiteBackend), "Cache backend: sqlite or mysql or postgresql or none")
	rootCmd.PersistentFlags().String("cache-db-connect", "", "Database connection string for mysql/postgresql (e.g., user:pass@tcp(host:port)/dbname)")
	rootCmd.PersistentFlags().String("analysis-backend", "", "Analysis tracking backend: sqlite or mysql or postgresql or none")
	rootCmd.PersistentFlags().String("analysis-db-connect", "", "Database connection string for analysis tracking (must differ from cache-db-connect)")
	rootCmd.PersistentFlags().String("color", "yes", "Enable colored labels in output (yes/no/true/false/1/0)")
	rootCmd.PersistentFlags().String("lookback", "6 months", "Time duration to look back from Base/Target ref commit time")
	rootCmd.PersistentFlags().String("base-ref", "", "Base Git reference for the BEFORE state")
	rootCmd.PersistentFlags().String("target-ref", "", "Target Git reference for the AFTER state")
	rootCmd.PersistentFlags().String("config", "", "Path to config file")
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		contract.LogFatal("Error binding root flags", err)
	}

	// Bind all flags of filesCmd to Viper
	filesCmd.Flags().Bool("explain", false, "Print per-file component score breakdown")
	filesCmd.Flags().Bool("follow", false, "Re-run per-file analysis with --follow")
	if err := viper.BindPFlags(filesCmd.Flags()); err != nil {
		contract.LogFatal("Error binding files flags", err)
	}

	// Bind all flags of timeseriesCmd to Viper
	timeseriesCmd.Flags().String("path", "", "Path to the file or folder to analyze")
	timeseriesCmd.Flags().String("interval", "3 months", "Total time interval")
	timeseriesCmd.Flags().Int("points", 3, "Number of lookback points")
	if err := viper.BindPFlags(timeseriesCmd.Flags()); err != nil {
		contract.LogFatal("Error binding timeseries flags", err)
	}

	// Bind all flags of checkCmd to Viper
	checkCmd.Flags().String("thresholds-override", "", "Risk thresholds for CI/CD gating (format: 'hot:50,risk:50,complexity:50,stale:50')")
	if err := viper.BindPFlags(checkCmd.Flags()); err != nil {
		contract.LogFatal("Error binding check flags", err)
	}

	// Bind all flags of analysisMigrateCmd to Viper
	analysisMigrateCmd.Flags().Int("target-version", -1, "Target migration version (-1 means latest, 0 means rollback to initial state)")
	if err := viper.BindPFlags(analysisMigrateCmd.Flags()); err != nil {
		contract.LogFatal("Error binding analysis migrate flags", err)
	}
}
