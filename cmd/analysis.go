package cmd

import (
	"fmt"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/internal/outwriter"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// analysisSetup loads minimal configuration needed for analysis operations.
// This is used by commands that need analysis access without full shared setup.
func analysisSetup() error {
	if err := loadConfigFile(); err != nil {
		return err
	}

	// Get analysis-related config values
	backend := schema.DatabaseBackend(viper.GetString("analysis-backend"))
	connStr := viper.GetString("analysis-db-connect")

	// If backend is empty (unlikely with flag default), use SQLite
	if backend == "" {
		backend = schema.SQLiteBackend
	}

	// For SQLite with no connection string, use the default path
	if backend == schema.SQLiteBackend && connStr == "" {
		connStr = iocache.GetAnalysisDBFilePath()
	}

	// Basic validation for database backends
	if err := config.ValidateDatabaseConnectionString(backend, connStr); err != nil {
		return err
	}

	// Get output-related config values (used by export command)
	outputFile := viper.GetString("output-file")
	outputFormat := viper.GetString("output")

	// Initialize stores with the loaded config (no cache tracking for analysis commands)
	if err := iocache.InitStores(schema.NoneBackend, "", backend, connStr, git.NewLocalGitClient()); err != nil {
		return fmt.Errorf("failed to initialize analysis: %w", err)
	}

	cfg.Runtime.AnalysisBackend = backend
	cfg.Runtime.AnalysisDBConnect = connStr
	cfg.Output.OutputFile = outputFile
	if outputFormat != "" {
		cfg.Output.Format = schema.OutputMode(outputFormat)
	} else {
		cfg.Output.Format = schema.TextOut // Default
	}

	// Initialize output infrastructure
	resultWriter = outwriter.NewOutWriter()

	return nil
}

// analysisSetupWrapper wraps analysisSetup to provide PreRunE for analysis commands.
func analysisSetupWrapper(_ *cobra.Command, _ []string) error {
	return analysisSetup()
}

// analysisMigrateSetup loads minimal configuration needed for migrate operations.
// This is a specialized setup that does NOT initialize stores or create tables,
// allowing migrations to run on a fresh database.
func analysisMigrateSetup() error {
	if err := loadConfigFile(); err != nil {
		return err
	}

	// Get analysis-related config values
	backend := schema.DatabaseBackend(viper.GetString("analysis-backend"))
	connStr := viper.GetString("analysis-db-connect")

	// If backend is empty (unlikely with flag default), use SQLite
	if backend == "" {
		backend = schema.SQLiteBackend
	}

	// Basic validation for database backends
	if err := config.ValidateDatabaseConnectionString(backend, connStr); err != nil {
		return err
	}

	// For SQLite backend with empty connection string, use default path
	if backend == schema.SQLiteBackend && connStr == "" {
		connStr = iocache.GetAnalysisDBFilePath()
	}

	cfg.Runtime.AnalysisBackend = backend
	cfg.Runtime.AnalysisDBConnect = connStr

	return nil
}

// analysisMigrateSetupWrapper wraps analysisMigrateSetup to provide PreRunE for migrate command.
func analysisMigrateSetupWrapper(_ *cobra.Command, _ []string) error {
	return analysisMigrateSetup()
}

// analysisCmd focused on analysis data management.
//
// Note: Analysis subcommands use minimal initialization (analysisSetup) instead of
// the full sharedSetup used by analysis commands. This avoids Git repo validation
// and complex config processing for simple analysis operations.
var analysisCmd = &cobra.Command{
	Use:   "analysis",
	Short: "Manage historical analysis tracking and exports",
	Long: `Manage historical analysis data used for trend tracking and reporting.

When enabled, Hotspot tracks every analysis run, storing:
- Run metadata (timestamp, configuration, duration)
- File scores across all modes (hot, risk, complexity, roi)
- Raw Git metrics (commits, churn, contributors, etc.)

This enables longitudinal analysis, trend detection, and data export for BI tools.

Supported backends: SQLite (default), MySQL, PostgreSQL, or None (disabled)

Subcommands:
  status  - Show analysis tracking statistics
  export  - Export data to Parquet for analytics
  clear   - Remove all tracking data
  migrate - Run database schema migrations

Examples:
  # Check tracking status
  hotspot analysis status

  # View recent history
  hotspot analysis history --limit 10

  # Export for analysis in pandas/DuckDB
  hotspot analysis export --output-file analysis-data.parquet`,
}

// analysisHistoryCmd shows the analysis history.
var analysisHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show chronological history of analysis runs",
	Long: `Display a list of past analysis executions stored in the tracking database.

Displays:
- Analysis ID
- Start time
- Run duration
- Total files analyzed
- Repository URN
- Configuration parameters

Use this to:
- Verify that your analysis runs are being recorded
- Look up IDs for specific runs
- Compare run durations across different configurations
- Confirm which repositories have been analyzed`,
	PreRunE: analysisSetupWrapper,
	Run: func(cmd *cobra.Command, _ []string) {
		limit := viper.GetInt("limit")
		runs, err := iocache.Manager.GetAnalysisStore().GetAnalysisRuns(schema.AnalysisQueryFilter{
			Limit: limit,
		})
		if err != nil {
			logger.Fatal("Failed to get analysis history", err)
		}
		if err := resultWriter.WriteHistory(cmd.OutOrStdout(), runs, cfg.Output); err != nil {
			logger.Fatal("Failed to write analysis history", err)
		}
	},
}

// analysisClearCmd clears the analysis data.
var analysisClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all historical analysis tracking data",
	Long: `Delete all stored analysis runs and file score history.

This removes:
- All analysis run metadata
- Historical file scores across all modes
- Raw Git metrics for analyzed files

WARNING: This action cannot be undone. Consider exporting data first.

Use this when:
- Resetting trend tracking
- Database storage is full
- Starting fresh analysis history
- Testing analysis features

Examples:
  # Export before clearing
  hotspot analysis export --output-file backup.parquet
  hotspot analysis clear

  # Clear and start fresh
  hotspot analysis clear`,
	PreRunE: analysisMigrateSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := iocache.ClearAnalysis(cfg.Runtime.AnalysisBackend, iocache.GetAnalysisDBFilePath(), cfg.Runtime.AnalysisDBConnect); err != nil {
			logger.Fatal("Failed to clear analysis data", err)
		}
		fmt.Println("Analysis data cleared successfully.")
	},
}

// analysisStatusCmd shows analysis status.
var analysisStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display analysis tracking statistics and connection details",
	Long: `Show detailed information about historical analysis tracking.

Displays:
- Backend type and connection status
- Total number of analysis runs stored
- Last and oldest analysis run timestamps
- Total files analyzed across all runs
- Database table sizes

Use this to:
- Verify analysis tracking is enabled and working
- Monitor data accumulation over time
- Check database connection health
- Estimate storage requirements

Examples:
  # Check analysis tracking status
  hotspot analysis status`,
	PreRunE: analysisSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		status, err := iocache.Manager.GetAnalysisStore().GetStatus()
		if err != nil {
			logger.Fatal("Failed to get analysis status", err)
		}
		iocache.PrintAnalysisStatus(status)
	},
}

// analysisExportCmd exports analysis data to Parquet files.
var analysisExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export historical data to Parquet for BI tools and analytics",
	Long: `Export all stored analysis data to Parquet format for use with analytics tools.

Exports two datasets:
- Analysis runs - metadata about each analysis execution
- File scores/metrics - detailed scores and Git metrics per file

Parquet format enables:
- Fast querying with DuckDB, Apache Spark, pandas
- Efficient storage with columnar compression
- Schema evolution for future data additions
- Direct import into BI tools (Tableau, Metabase, etc.)

Requires: --output-file parameter

Use cases:
- Trend analysis across multiple runs
- Custom dashboards and visualizations
- ML model training on code metrics
- Executive reporting and KPIs

Examples:
  # Export all data
  hotspot analysis export --output-file hotspot-data.parquet

  # Use with DuckDB for analysis
  hotspot analysis export --output-file data.parquet
  duckdb -c "SELECT * FROM read_parquet('data.parquet/runs.parquet') LIMIT 10"`,
	PreRunE: analysisMigrateSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		if err := iocache.ExecuteAnalysisExport(cfg.Output.OutputFile); err != nil {
			logger.Fatal("Failed to export analysis data", err)
		}
	},
}

// analysisMigrateCmd runs database migrations for the analysis store.
var analysisMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database schema migrations (upgrades/downgrades)",
	Long: `Manage database schema versions for the analysis tracking store.

Migrations allow:
- Upgrading to new schema versions when Hotspot is updated
- Safely modifying database structure without data loss
- Rolling back schema changes if needed
- Testing new features on specific schema versions

By default, migrates to the latest version. Use --target-version for specific versions.

Examples:
  # Migrate to latest version (default)
  hotspot analysis migrate

  # Migrate to specific version
  hotspot analysis migrate --target-version 2

  # Rollback to previous version
  hotspot analysis migrate --target-version 0`,
	PreRunE: analysisMigrateSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		targetVersion := viper.GetInt("target-version")
		force := viper.GetBool("force")
		if err := iocache.MigrateAnalysis(cfg.Runtime.AnalysisBackend, cfg.Runtime.AnalysisDBConnect, targetVersion, force); err != nil {
			logger.Fatal("Failed to run migrations", err)
		}
	},
}
