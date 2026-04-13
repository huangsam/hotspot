package iocache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/schema"
)

// Table names for analysis tracking.
const (
	analysisRunsTable      = "hotspot_analysis_runs"
	fileScoresMetricsTable = "hotspot_file_scores_metrics"
)

// AnalysisStoreImpl implements the AnalysisStore interface.
type AnalysisStoreImpl struct {
	db      *sql.DB
	backend schema.DatabaseBackend
	dialect SQLDialect
}

var _ AnalysisStore = &AnalysisStoreImpl{} // Compile-time check

// NewAnalysisStore creates a new AnalysisStore with the specified backend.
func NewAnalysisStore(backend schema.DatabaseBackend, connStr string) (AnalysisStore, error) {
	var db *sql.DB
	var err error
	var dialect SQLDialect

	switch backend {
	case schema.SQLiteBackend:
		dialect = &SQLiteDialect{}
		dbPath := connStr
		if dbPath == "" {
			dbPath = GetAnalysisDBFilePath()
		}
		db, err = sql.Open(dialect.DriverName(), dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database at %q: %w. Check that the directory is writable", dbPath, err)
		}
		// Limit SQLite to a single open connection to avoid "database is locked" errors
		db.SetMaxOpenConns(1)

	case schema.MySQLBackend:
		dialect = &MySQLDialect{}
		db, err = sql.Open(dialect.DriverName(), connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open MySQL database: %w. Check connection string format: user:password@tcp(host:port)/dbname", err)
		}

	case schema.PostgreSQLBackend:
		dialect = &PostgresDialect{}
		db, err = sql.Open(dialect.DriverName(), connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open PostgreSQL database: %w. Check connection string format: postgres://user:password@host:port/dbname", err)
		}

	case schema.NoneBackend:
		// Return a no-op store for disabled tracking
		return &AnalysisStoreImpl{
			db:      nil,
			backend: backend,
			dialect: nil,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		var connDetail string
		switch backend {
		case schema.MySQLBackend:
			connDetail = "Check that MySQL is running and the connection string is correct. Ensure user/password are valid."
		case schema.PostgreSQLBackend:
			connDetail = "Check that PostgreSQL is running and the connection string is correct. Ensure user/password are valid."
		default:
			connDetail = "Verify the database server is running and accessible."
		}
		return nil, fmt.Errorf("failed to connect to %s database: %w. %s", backend, err, connDetail)
	}

	// Create the table schemas
	if err := createAnalysisTables(db, dialect); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create analysis tables: %w", err)
	}

	store := &AnalysisStoreImpl{
		db:      db,
		backend: backend,
		dialect: dialect,
	}

	// Backfill URNs for legacy runs synchronously to ensure store consistency
	if err := BackfillAnalysisURNs(store); err != nil {
		logger.Warn("Analysis URN backfill encountered errors", err)
		// Don't fail store creation, but log the issue for debugging
	}

	return store, nil
}

// createAnalysisTables creates the analysis tracking tables.
func createAnalysisTables(db *sql.DB, dialect SQLDialect) error {
	tables := []struct {
		name  string
		query string
	}{
		{analysisRunsTable, dialect.GetCreateAnalysisRunsQuery(analysisRunsTable)},
		{fileScoresMetricsTable, dialect.GetCreateFileScoresMetricsQuery(fileScoresMetricsTable)},
	}

	for _, table := range tables {
		if _, err := db.Exec(table.query); err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
	}

	// Create index on URN column for filtered queries
	if err := createURNIndex(db, dialect); err != nil {
		return fmt.Errorf("failed to create URN index: %w", err)
	}

	return nil
}

// createURNIndex creates the idx_runs_urn index, handling dialect differences.
// MySQL does not support CREATE INDEX IF NOT EXISTS, so we check existence first.
func createURNIndex(db *sql.DB, dialect SQLDialect) error {
	tableName := dialect.QuoteIdentifier(analysisRunsTable)

	if dialect.DriverName() == "mysql" {
		// MySQL: check if index exists before creating
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_name = ? AND index_name = ?",
			analysisRunsTable, "idx_runs_urn",
		).Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			_, err = db.Exec(fmt.Sprintf("CREATE INDEX idx_runs_urn ON %s(urn)", tableName))
			return err
		}
		return nil
	}

	// SQLite and PostgreSQL support IF NOT EXISTS
	_, err := db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_runs_urn ON %s(urn)", tableName))
	return err
}

// BeginAnalysis creates a new analysis run and returns its unique ID.
func (as *AnalysisStoreImpl) BeginAnalysis(urn string, startTime time.Time, configParams map[string]any) (int64, error) {
	// Skip for NoneBackend
	if as.db == nil || as.dialect == nil {
		return 0, nil
	}

	// Serialize config params to JSON
	configJSON, err := json.Marshal(configParams)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal config params: %w", err)
	}

	analysisID, err := as.dialect.BeginAnalysis(as.db, analysisRunsTable, urn, startTime, string(configJSON))
	if err != nil {
		return 0, fmt.Errorf("failed to insert analysis run: %w", err)
	}

	return analysisID, nil
}

// EndAnalysis updates the analysis run with completion data.
func (as *AnalysisStoreImpl) EndAnalysis(analysisID int64, endTime time.Time, totalFiles int) error {
	// Skip for NoneBackend
	if as.db == nil || as.dialect == nil {
		return nil
	}

	// First, get the start_time to calculate duration
	quotedTableName := as.dialect.QuoteIdentifier(analysisRunsTable)

	var query string
	switch as.backend {
	case schema.PostgreSQLBackend:
		query = fmt.Sprintf(`SELECT start_time FROM %s WHERE analysis_id = $1`, quotedTableName)
	default: // SQLite and MySQL
		query = fmt.Sprintf(`SELECT start_time FROM %s WHERE analysis_id = ?`, quotedTableName)
	}

	row := as.db.QueryRow(query, analysisID)
	startTime, err := as.dialect.ScanStartTime(row)
	if err != nil {
		return fmt.Errorf("failed to get start_time for analysis %d: %w", analysisID, err)
	}

	// Calculate duration in milliseconds
	durationMs := endTime.Sub(startTime).Milliseconds()

	// Update the analysis run with completion data
	updateQuery := as.dialect.GetUpdateEndAnalysisQuery(analysisRunsTable)
	var args []any

	switch as.backend {
	case schema.PostgreSQLBackend:
		args = []any{endTime, durationMs, totalFiles, analysisID}
	default: // SQLite and MySQL
		args = []any{as.dialect.FormatTime(endTime), durationMs, totalFiles, analysisID}
	}

	_, err = as.db.Exec(updateQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to update analysis run: %w", err)
	}

	return nil
}

// UpdateAnalysisRunURN updates the urn for an existing analysis run record.
func (as *AnalysisStoreImpl) UpdateAnalysisRunURN(analysisID int64, urn string) error {
	// Skip for NoneBackend
	if as.db == nil || as.dialect == nil {
		return nil
	}

	quotedTableName := as.dialect.QuoteIdentifier(analysisRunsTable)
	var query string
	switch as.dialect.DriverName() {
	case "pgx":
		query = fmt.Sprintf("UPDATE %s SET urn = $1 WHERE analysis_id = $2", quotedTableName)
	default:
		query = fmt.Sprintf("UPDATE %s SET urn = ? WHERE analysis_id = ?", quotedTableName)
	}

	_, err := as.db.Exec(query, urn, analysisID)
	return err
}

// RecordFileMetricsAndScores stores both raw git metrics and final scores for a file in one operation.
func (as *AnalysisStoreImpl) RecordFileMetricsAndScores(analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error {
	// Skip for NoneBackend
	if as.db == nil || as.dialect == nil {
		return nil
	}

	return as.dialect.RecordFileMetricsAndScores(as.db, fileScoresMetricsTable, analysisID, filePath, metrics, scores)
}

// Close closes the underlying connection.
func (as *AnalysisStoreImpl) Close() error {
	if as.db != nil {
		return as.db.Close()
	}
	return nil
}

// placeholder returns a SQL clause fragment with a backend-appropriate placeholder.
// For PostgreSQL it uses $N; for others it uses ?.
func (as *AnalysisStoreImpl) placeholder(prefix string, argIdx *int) string {
	return prefix + as.placeholderStr(argIdx)
}

// placeholderStr returns a single placeholder token and increments the index.
func (as *AnalysisStoreImpl) placeholderStr(argIdx *int) string {
	if as.dialect.DriverName() == "pgx" {
		p := fmt.Sprintf("$%d", *argIdx)
		*argIdx++
		return p
	}
	*argIdx++
	return "?"
}

// GetStatus returns status information about the analysis store.
func (as *AnalysisStoreImpl) GetStatus() (schema.AnalysisStatus, error) {
	status := schema.AnalysisStatus{
		Backend:    string(as.backend),
		Connected:  as.db != nil,
		TableSizes: make(map[string]int64),
	}

	if as.db == nil || as.dialect == nil {
		return status, nil
	}

	// Get total runs
	runsQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", as.dialect.QuoteIdentifier(analysisRunsTable))
	row := as.db.QueryRow(runsQuery)
	if err := row.Scan(&status.TotalRuns); err != nil {
		return status, fmt.Errorf("failed to get total runs: %w", err)
	}

	if status.TotalRuns > 0 {
		// Get last run info
		lastRunQuery := fmt.Sprintf("SELECT analysis_id, start_time FROM %s ORDER BY analysis_id DESC LIMIT 1", as.dialect.QuoteIdentifier(analysisRunsTable))
		row = as.db.QueryRow(lastRunQuery)

		var err error
		status.LastRunID, status.LastRunTime, err = as.dialect.ScanLastRunInfo(row)
		if err != nil {
			return status, fmt.Errorf("failed to get last run info: %w", err)
		}

		// Get oldest run time
		oldestRunQuery := fmt.Sprintf("SELECT start_time FROM %s ORDER BY analysis_id ASC LIMIT 1", as.dialect.QuoteIdentifier(analysisRunsTable))
		row = as.db.QueryRow(oldestRunQuery)

		status.OldestRunTime, err = as.dialect.ScanOldestRunTime(row)
		if err != nil {
			return status, fmt.Errorf("failed to get oldest run time: %w", err)
		}

		// Get total files analyzed
		filesQuery := fmt.Sprintf("SELECT COALESCE(SUM(total_files_analyzed), 0) FROM %s", as.dialect.QuoteIdentifier(analysisRunsTable))
		row = as.db.QueryRow(filesQuery)
		if err := row.Scan(&status.TotalFilesAnalyzed); err != nil {
			return status, fmt.Errorf("failed to get total files analyzed: %w", err)
		}
	}

	// Get table sizes
	tables := []string{analysisRunsTable, fileScoresMetricsTable}
	for _, table := range tables {
		quotedTable := as.dialect.QuoteIdentifier(table)
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
		row = as.db.QueryRow(countQuery)
		var count int64
		if err := row.Scan(&count); err != nil {
			return status, fmt.Errorf("failed to get count for table %s: %w", table, err)
		}
		status.TableSizes[table] = count
	}

	return status, nil
}

// GetAllAnalysisRuns retrieves all analysis runs from the store.
func (as *AnalysisStoreImpl) GetAllAnalysisRuns() ([]schema.AnalysisRunRecord, error) {
	return as.GetAnalysisRuns(schema.AnalysisQueryFilter{})
}

// GetAnalysisRuns retrieves analysis runs with optional filtering and pagination.
func (as *AnalysisStoreImpl) GetAnalysisRuns(filter schema.AnalysisQueryFilter) ([]schema.AnalysisRunRecord, error) {
	// Skip for NoneBackend
	if as.db == nil || as.dialect == nil {
		return nil, nil
	}

	quotedTableName := as.dialect.QuoteIdentifier(analysisRunsTable)
	query := fmt.Sprintf("SELECT analysis_id, start_time, end_time, run_duration_ms, total_files_analyzed, config_params, urn FROM %s", quotedTableName)

	var args []any
	argIdx := 1 // For PostgreSQL $N placeholders

	if filter.URN != "" {
		query += as.placeholder(" WHERE urn = ", &argIdx)
		args = append(args, filter.URN)
	}

	query += " ORDER BY analysis_id"

	if filter.Limit > 0 {
		query += as.placeholder(" LIMIT ", &argIdx)
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += as.placeholder(" OFFSET ", &argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := as.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []schema.AnalysisRunRecord

	for rows.Next() {
		var record schema.AnalysisRunRecord
		if err := as.dialect.ScanAnalysisRunRecord(rows, &record); err != nil {
			return nil, fmt.Errorf("failed to scan analysis run: %w", err)
		}
		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating analysis runs: %w", err)
	}

	return results, nil
}

// GetAllFileScoresMetrics retrieves all file scores and metrics from the store.
func (as *AnalysisStoreImpl) GetAllFileScoresMetrics() ([]schema.FileScoresMetricsRecord, error) {
	return as.GetFileScoresMetrics(schema.AnalysisQueryFilter{})
}

// GetFileScoresMetrics retrieves file scores and metrics with optional filtering and pagination.
func (as *AnalysisStoreImpl) GetFileScoresMetrics(filter schema.AnalysisQueryFilter) ([]schema.FileScoresMetricsRecord, error) {
	// Skip for NoneBackend
	if as.db == nil || as.dialect == nil {
		return nil, nil
	}

	quotedTableName := as.dialect.QuoteIdentifier(fileScoresMetricsTable)
	query := fmt.Sprintf(`SELECT analysis_id, file_path, analysis_time, total_commits, total_churn,
    contributor_count, age_days, gini_coefficient, file_owner,
    score_hot, score_risk, score_complexity, score_stale, score_label
    FROM %s`, quotedTableName)

	var args []any
	argIdx := 1

	if filter.URN != "" {
		runsTable := as.dialect.QuoteIdentifier(analysisRunsTable)
		query += fmt.Sprintf(" WHERE analysis_id IN (SELECT analysis_id FROM %s WHERE urn = %s)", runsTable, as.placeholderStr(&argIdx))
		args = append(args, filter.URN)
	}

	query += " ORDER BY analysis_id, file_path"

	if filter.Limit > 0 {
		query += as.placeholder(" LIMIT ", &argIdx)
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += as.placeholder(" OFFSET ", &argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := as.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query file scores metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []schema.FileScoresMetricsRecord

	for rows.Next() {
		var record schema.FileScoresMetricsRecord
		if err := as.dialect.ScanFileScoresMetricsRecord(rows, &record); err != nil {
			return nil, fmt.Errorf("failed to scan file scores metrics: %w", err)
		}
		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating file scores metrics: %w", err)
	}

	return results, nil
}
