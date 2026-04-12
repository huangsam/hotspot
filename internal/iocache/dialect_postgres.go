package iocache

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// PostgresDialect handles PostgreSQL-specific SQL syntax and data types.
type PostgresDialect struct{}

// DriverName returns the driver name for PostgreSQL (pgx).
func (d *PostgresDialect) DriverName() string {
	return "pgx"
}

// QuoteIdentifier returns a double-quoted identifier for PostgreSQL.
func (d *PostgresDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("\"%s\"", name)
}

// GetCreateAnalysisRunsQuery returns the PostgreSQL-specific table creation query for analysis runs.
func (d *PostgresDialect) GetCreateAnalysisRunsQuery(tableName string) string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			analysis_id BIGSERIAL PRIMARY KEY,
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ,
			run_duration_ms INT,
			total_files_analyzed INT,
			config_params TEXT,
			urn TEXT
		);
	`, d.QuoteIdentifier(tableName))
}

// GetCreateFileScoresMetricsQuery returns the PostgreSQL-specific table creation query for file scores.
func (d *PostgresDialect) GetCreateFileScoresMetricsQuery(tableName string) string {
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
			score_hot DOUBLE PRECISION NOT NULL,
			score_risk DOUBLE PRECISION NOT NULL,
			score_complexity DOUBLE PRECISION NOT NULL,
			score_stale DOUBLE PRECISION NOT NULL,
			score_label TEXT NOT NULL,
			PRIMARY KEY (analysis_id, file_path)
		);
	`, d.QuoteIdentifier(tableName))
}

// BeginAnalysis inserts a new analysis run into PostgreSQL and returns the generated ID using RETURNING.
func (d *PostgresDialect) BeginAnalysis(db *sql.DB, tableName string, urn string, startTime time.Time, configJSON string) (int64, error) {
	query := fmt.Sprintf(`INSERT INTO %s (start_time, config_params, urn) VALUES ($1, $2, $3) RETURNING analysis_id`, d.QuoteIdentifier(tableName))
	var analysisID int64
	err := db.QueryRow(query, startTime, configJSON, urn).Scan(&analysisID)
	return analysisID, err
}

// ScanStartTime parses the start time from a PostgreSQL row.
func (d *PostgresDialect) ScanStartTime(row *sql.Row) (time.Time, error) {
	var startTime time.Time
	if err := row.Scan(&startTime); err != nil {
		return time.Time{}, err
	}
	return startTime, nil
}

// GetUpdateEndAnalysisQuery returns the PostgreSQL-specific query for updating analysis completion using $n placeholders.
func (d *PostgresDialect) GetUpdateEndAnalysisQuery(tableName string) string {
	return fmt.Sprintf(`UPDATE %s SET end_time = $1, run_duration_ms = $2, total_files_analyzed = $3 WHERE analysis_id = $4`, d.QuoteIdentifier(tableName))
}

// RecordFileMetricsAndScores inserts file-level metrics and scores into PostgreSQL.
func (d *PostgresDialect) RecordFileMetricsAndScores(db *sql.DB, tableName string, analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn,
						 contributor_count, age_days, gini_coefficient, file_owner,
						 score_hot, score_risk, score_complexity, score_stale, score_label)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, d.QuoteIdentifier(tableName))

	_, err := db.Exec(query,
		analysisID, filePath, d.FormatTime(metrics.AnalysisTime), metrics.TotalCommits, metrics.TotalChurn,
		metrics.ContributorCount, metrics.AgeDays, metrics.GiniCoefficient, metrics.FileOwner,
		scores.HotScore, scores.RiskScore, scores.ComplexityScore, scores.StaleScore, scores.ScoreLabel,
	)
	return err
}

// ScanLastRunInfo parses the latest analysis run metadata from PostgreSQL.
func (d *PostgresDialect) ScanLastRunInfo(row *sql.Row) (int64, time.Time, error) {
	var lastRunID int64
	var lastRunTime time.Time
	if err := row.Scan(&lastRunID, &lastRunTime); err != nil {
		return 0, time.Time{}, err
	}
	return lastRunID, lastRunTime, nil
}

// ScanOldestRunTime parses the oldest analysis run time from PostgreSQL.
func (d *PostgresDialect) ScanOldestRunTime(row *sql.Row) (time.Time, error) {
	var oldestRunTime time.Time
	if err := row.Scan(&oldestRunTime); err != nil {
		return time.Time{}, err
	}
	return oldestRunTime, nil
}

// ScanAnalysisRunRecord parses a full analysis run record from PostgreSQL rows.
func (d *PostgresDialect) ScanAnalysisRunRecord(rows *sql.Rows, record *schema.AnalysisRunRecord) error {
	return rows.Scan(&record.AnalysisID, &record.StartTime, &record.EndTime, &record.RunDurationMs, &record.TotalFilesAnalyzed, &record.ConfigParams, &record.URN)
}

// ScanFileScoresMetricsRecord parses a full file metrics and scores record from PostgreSQL rows.
func (d *PostgresDialect) ScanFileScoresMetricsRecord(rows *sql.Rows, record *schema.FileScoresMetricsRecord) error {
	return rows.Scan(&record.AnalysisID, &record.FilePath, &record.AnalysisTime, &record.TotalCommits,
		&record.TotalChurn, &record.ContributorCount, &record.AgeDays, &record.GiniCoefficient,
		&record.FileOwner, &record.ScoreHot, &record.ScoreRisk, &record.ScoreComplexity,
		&record.ScoreStale, &record.ScoreLabel)
}

// FormatTime converts a time.Time to a PostgreSQL-compatible format (passing native time.Time works).
func (d *PostgresDialect) FormatTime(t time.Time) any {
	return t
}
