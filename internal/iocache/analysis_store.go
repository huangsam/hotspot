package iocache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// Table names for analysis tracking.
const (
	analysisRunsTable  = "hotspot_analysis_runs"
	rawGitMetricsTable = "hotspot_raw_git_metrics"
	finalScoresTable   = "hotspot_final_scores"
)

// AnalysisStoreImpl implements the AnalysisStore interface.
type AnalysisStoreImpl struct {
	db         *sql.DB
	backend    schema.CacheBackend
	driverName string
}

var _ contract.AnalysisStore = &AnalysisStoreImpl{} // Compile-time check

// NewAnalysisStore creates a new AnalysisStore with the specified backend.
func NewAnalysisStore(backend schema.CacheBackend, connStr string) (contract.AnalysisStore, error) {
	var db *sql.DB
	var err error
	var driverName string

	switch backend {
	case schema.SQLiteBackend:
		driverName = "sqlite3"
		dbPath := connStr
		if dbPath == "" {
			dbPath = GetAnalysisDBFilePath()
		}
		db, err = sql.Open(driverName, dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database: %w", err)
		}
		// Limit SQLite to a single open connection to avoid "database is locked" errors
		db.SetMaxOpenConns(1)

	case schema.MySQLBackend:
		driverName = "mysql"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open MySQL database: %w", err)
		}

	case schema.PostgreSQLBackend:
		driverName = "pgx"
		db, err = sql.Open(driverName, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
		}

	case schema.NoneBackend:
		// Return a no-op store for disabled tracking
		return &AnalysisStoreImpl{
			db:         nil,
			backend:    backend,
			driverName: "",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported backend: %s", backend)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping %s database: %w", backend, err)
	}

	// Create the table schemas
	if err := createAnalysisTables(db, backend); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create analysis tables: %w", err)
	}

	return &AnalysisStoreImpl{
		db:         db,
		backend:    backend,
		driverName: driverName,
	}, nil
}

// createAnalysisTables creates all three analysis tracking tables.
func createAnalysisTables(db *sql.DB, backend schema.CacheBackend) error {
	tables := []struct {
		name  string
		query string
	}{
		{analysisRunsTable, getCreateAnalysisRunsQuery(backend)},
		{rawGitMetricsTable, getCreateRawGitMetricsQuery(backend)},
		{finalScoresTable, getCreateFinalScoresQuery(backend)},
	}

	for _, table := range tables {
		if _, err := db.Exec(table.query); err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
	}

	return nil
}

// getCreateAnalysisRunsQuery returns the CREATE TABLE query for hotspot_analysis_runs.
func getCreateAnalysisRunsQuery(backend schema.CacheBackend) string {
	quotedTableName := quoteTableName(analysisRunsTable, backend)

	switch backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id BIGINT AUTO_INCREMENT PRIMARY KEY,
				start_time DATETIME(6) NOT NULL,
				end_time DATETIME(6),
				run_duration_ms INT,
				total_files_analyzed INT,
				config_params TEXT
			);
		`, quotedTableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id BIGSERIAL PRIMARY KEY,
				start_time TIMESTAMPTZ NOT NULL,
				end_time TIMESTAMPTZ,
				run_duration_ms INT,
				total_files_analyzed INT,
				config_params TEXT
			);
		`, quotedTableName)

	default: // SQLite
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id INTEGER PRIMARY KEY AUTOINCREMENT,
				start_time TEXT NOT NULL,
				end_time TEXT,
				run_duration_ms INTEGER,
				total_files_analyzed INTEGER,
				config_params TEXT
			);
		`, quotedTableName)
	}
}

// getCreateRawGitMetricsQuery returns the CREATE TABLE query for hotspot_raw_git_metrics.
func getCreateRawGitMetricsQuery(backend schema.CacheBackend) string {
	quotedTableName := quoteTableName(rawGitMetricsTable, backend)

	switch backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id BIGINT NOT NULL,
				file_path VARCHAR(512) NOT NULL,
				analysis_time DATETIME(6) NOT NULL,
				total_commits INT NOT NULL,
				total_churn INT NOT NULL,
				contributor_count INT NOT NULL,
				age_days DOUBLE NOT NULL,
				gini_coefficient DOUBLE NOT NULL,
				file_owner VARCHAR(100),
				PRIMARY KEY (analysis_id, file_path)
			);
		`, quotedTableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id BIGINT NOT NULL,
				file_path TEXT NOT NULL,
				analysis_time TIMESTAMPTZ NOT NULL,
				total_commits INT NOT NULL,
				total_churn INT NOT NULL,
				contributor_count INT NOT NULL,
				age_days DOUBLE PRECISION NOT NULL,
				gini_coefficient DOUBLE PRECISION NOT NULL,
				file_owner TEXT,
				PRIMARY KEY (analysis_id, file_path)
			);
		`, quotedTableName)

	default: // SQLite
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id INTEGER NOT NULL,
				file_path TEXT NOT NULL,
				analysis_time TEXT NOT NULL,
				total_commits INTEGER NOT NULL,
				total_churn INTEGER NOT NULL,
				contributor_count INTEGER NOT NULL,
				age_days REAL NOT NULL,
				gini_coefficient REAL NOT NULL,
				file_owner TEXT,
				PRIMARY KEY (analysis_id, file_path)
			);
		`, quotedTableName)
	}
}

// getCreateFinalScoresQuery returns the CREATE TABLE query for hotspot_final_scores.
func getCreateFinalScoresQuery(backend schema.CacheBackend) string {
	quotedTableName := quoteTableName(finalScoresTable, backend)

	switch backend {
	case schema.MySQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id BIGINT NOT NULL,
				file_path VARCHAR(512) NOT NULL,
				analysis_time DATETIME(6) NOT NULL,
				score_hot DOUBLE NOT NULL,
				score_risk DOUBLE NOT NULL,
				score_complexity DOUBLE NOT NULL,
				score_stale DOUBLE NOT NULL,
				score_label VARCHAR(50) NOT NULL,
				PRIMARY KEY (analysis_id, file_path)
			);
		`, quotedTableName)

	case schema.PostgreSQLBackend:
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id BIGINT NOT NULL,
				file_path TEXT NOT NULL,
				analysis_time TIMESTAMPTZ NOT NULL,
				score_hot DOUBLE PRECISION NOT NULL,
				score_risk DOUBLE PRECISION NOT NULL,
				score_complexity DOUBLE PRECISION NOT NULL,
				score_stale DOUBLE PRECISION NOT NULL,
				score_label TEXT NOT NULL,
				PRIMARY KEY (analysis_id, file_path)
			);
		`, quotedTableName)

	default: // SQLite
		return fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				analysis_id INTEGER NOT NULL,
				file_path TEXT NOT NULL,
				analysis_time TEXT NOT NULL,
				score_hot REAL NOT NULL,
				score_risk REAL NOT NULL,
				score_complexity REAL NOT NULL,
				score_stale REAL NOT NULL,
				score_label TEXT NOT NULL,
				PRIMARY KEY (analysis_id, file_path)
			);
		`, quotedTableName)
	}
}

// BeginAnalysis creates a new analysis run and returns its unique ID.
func (as *AnalysisStoreImpl) BeginAnalysis(startTime time.Time, configParams map[string]any) (int64, error) {
	// Skip for NoneBackend
	if as.backend == schema.NoneBackend || as.db == nil {
		return 0, nil
	}

	// Serialize config params to JSON
	configJSON, err := json.Marshal(configParams)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal config params: %w", err)
	}

	quotedTableName := quoteTableName(analysisRunsTable, as.backend)

	var analysisID int64
	switch as.backend {
	case schema.PostgreSQLBackend:
		query := fmt.Sprintf(`INSERT INTO %s (start_time, config_params) VALUES ($1, $2) RETURNING analysis_id`, quotedTableName)
		err = as.db.QueryRow(query, startTime, string(configJSON)).Scan(&analysisID)
	default: // SQLite and MySQL
		query := fmt.Sprintf(`INSERT INTO %s (start_time, config_params) VALUES (?, ?)`, quotedTableName)
		var result sql.Result
		result, err = as.db.Exec(query, formatTime(startTime, as.backend), string(configJSON))
		if err != nil {
			return 0, err
		}
		analysisID, err = result.LastInsertId()
	}

	if err != nil {
		return 0, fmt.Errorf("failed to insert analysis run: %w", err)
	}

	return analysisID, nil
}

// EndAnalysis updates the analysis run with completion data.
func (as *AnalysisStoreImpl) EndAnalysis(analysisID int64, endTime time.Time, totalFiles int) error {
	// Skip for NoneBackend
	if as.backend == schema.NoneBackend || as.db == nil {
		return nil
	}

	// First, get the start_time to calculate duration
	quotedTableName := quoteTableName(analysisRunsTable, as.backend)
	var startTime time.Time

	var query string
	switch as.backend {
	case schema.PostgreSQLBackend:
		query = fmt.Sprintf(`SELECT start_time FROM %s WHERE analysis_id = $1`, quotedTableName)
	default: // SQLite and MySQL
		query = fmt.Sprintf(`SELECT start_time FROM %s WHERE analysis_id = ?`, quotedTableName)
	}

	row := as.db.QueryRow(query, analysisID)

	// Handle different time storage formats per backend
	switch as.backend {
	case schema.SQLiteBackend:
		var startTimeStr string
		if err := row.Scan(&startTimeStr); err != nil {
			return fmt.Errorf("failed to get start_time for analysis %d: %w", analysisID, err)
		}
		var err error
		startTime, err = time.Parse(time.RFC3339Nano, startTimeStr)
		if err != nil {
			return fmt.Errorf("failed to parse start_time: %w", err)
		}
	default: // MySQL and PostgreSQL store as native datetime
		if err := row.Scan(&startTime); err != nil {
			return fmt.Errorf("failed to get start_time for analysis %d: %w", analysisID, err)
		}
	}

	// Calculate duration in milliseconds
	durationMs := endTime.Sub(startTime).Milliseconds()

	// Update the analysis run with completion data
	var updateQuery string
	var args []any

	switch as.backend {
	case schema.PostgreSQLBackend:
		updateQuery = fmt.Sprintf(`UPDATE %s SET end_time = $1, run_duration_ms = $2, total_files_analyzed = $3 WHERE analysis_id = $4`, quotedTableName)
		args = []any{endTime, durationMs, totalFiles, analysisID}
	default: // SQLite and MySQL
		updateQuery = fmt.Sprintf(`UPDATE %s SET end_time = ?, run_duration_ms = ?, total_files_analyzed = ? WHERE analysis_id = ?`, quotedTableName)
		args = []any{formatTime(endTime, as.backend), durationMs, totalFiles, analysisID}
	}

	_, err := as.db.Exec(updateQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to update analysis run: %w", err)
	}

	return nil
}

// RecordFileMetrics stores raw git metrics for a file.
func (as *AnalysisStoreImpl) RecordFileMetrics(analysisID int64, filePath string, metrics schema.FileMetrics) error {
	// Skip for NoneBackend
	if as.backend == schema.NoneBackend || as.db == nil {
		return nil
	}

	quotedTableName := quoteTableName(rawGitMetricsTable, as.backend)

	var query string
	var args []any

	switch as.backend {
	case schema.MySQLBackend:
		query = fmt.Sprintf(`
			INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn,
			                 contributor_count, age_days, gini_coefficient, file_owner)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, quotedTableName)
		args = []any{
			analysisID, filePath, metrics.AnalysisTime, metrics.TotalCommits, metrics.TotalChurn,
			metrics.ContributorCount, metrics.AgeDays, metrics.GiniCoefficient, metrics.FileOwner,
		}
	case schema.PostgreSQLBackend:
		query = fmt.Sprintf(`
			INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn,
			                 contributor_count, age_days, gini_coefficient, file_owner)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, quotedTableName)
		args = []any{
			analysisID, filePath, metrics.AnalysisTime, metrics.TotalCommits, metrics.TotalChurn,
			metrics.ContributorCount, metrics.AgeDays, metrics.GiniCoefficient, metrics.FileOwner,
		}
	default: // SQLite
		query = fmt.Sprintf(`
			INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn,
			                 contributor_count, age_days, gini_coefficient, file_owner)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, quotedTableName)
		args = []any{
			analysisID, filePath, metrics.AnalysisTime, metrics.TotalCommits, metrics.TotalChurn,
			metrics.ContributorCount, metrics.AgeDays, metrics.GiniCoefficient, metrics.FileOwner,
		}
	}

	_, err := as.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert file metrics: %w", err)
	}

	return nil
}

// RecordFileScores stores final scores for a file.
func (as *AnalysisStoreImpl) RecordFileScores(analysisID int64, filePath string, scores schema.FileScores) error {
	// Skip for NoneBackend
	if as.backend == schema.NoneBackend || as.db == nil {
		return nil
	}

	quotedTableName := quoteTableName(finalScoresTable, as.backend)

	var query string
	var args []any

	switch as.backend {
	case schema.MySQLBackend:
		query = fmt.Sprintf(`
			INSERT INTO %s (analysis_id, file_path, analysis_time, score_hot, score_risk,
			                 score_complexity, score_stale, score_label)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, quotedTableName)
		args = []any{
			analysisID, filePath, formatTime(scores.AnalysisTime, as.backend), scores.HotScore, scores.RiskScore,
			scores.ComplexityScore, scores.StaleScore, scores.ScoreLabel,
		}
	case schema.PostgreSQLBackend:
		query = fmt.Sprintf(`
			INSERT INTO %s (analysis_id, file_path, analysis_time, score_hot, score_risk,
			                 score_complexity, score_stale, score_label)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, quotedTableName)
		args = []any{
			analysisID, filePath, formatTime(scores.AnalysisTime, as.backend), scores.HotScore, scores.RiskScore,
			scores.ComplexityScore, scores.StaleScore, scores.ScoreLabel,
		}
	default: // SQLite
		query = fmt.Sprintf(`
			INSERT INTO %s (analysis_id, file_path, analysis_time, score_hot, score_risk,
			                 score_complexity, score_stale, score_label)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, quotedTableName)
		args = []any{
			analysisID, filePath, formatTime(scores.AnalysisTime, as.backend), scores.HotScore, scores.RiskScore,
			scores.ComplexityScore, scores.StaleScore, scores.ScoreLabel,
		}
	}

	_, err := as.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert file scores: %w", err)
	}

	return nil
}

// Close closes the underlying connection.
func (as *AnalysisStoreImpl) Close() error {
	if as.db != nil {
		return as.db.Close()
	}
	return nil
}

// formatTime converts a time.Time to the appropriate format for the backend.
func formatTime(t time.Time, backend schema.CacheBackend) any {
	switch backend {
	case schema.SQLiteBackend:
		return t.Format(time.RFC3339Nano)
	default:
		return t
	}
}
