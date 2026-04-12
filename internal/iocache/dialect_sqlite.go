package iocache

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// SQLiteDialect handles SQLite-specific SQL syntax and data types.
type SQLiteDialect struct{}

// DriverName returns the driver name for SQLite.
func (d *SQLiteDialect) DriverName() string {
	return "sqlite"
}

// QuoteIdentifier returns a double-quoted identifier for SQLite.
func (d *SQLiteDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("\"%s\"", name)
}

// GetCreateAnalysisRunsQuery returns the SQLite-specific table creation query for analysis runs.
func (d *SQLiteDialect) GetCreateAnalysisRunsQuery(tableName string) string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			analysis_id INTEGER PRIMARY KEY AUTOINCREMENT,
			start_time TEXT NOT NULL,
			end_time TEXT,
			run_duration_ms INTEGER,
			total_files_analyzed INTEGER,
			config_params TEXT,
			urn TEXT
		);
	`, d.QuoteIdentifier(tableName))
}

// GetCreateFileScoresMetricsQuery returns the SQLite-specific table creation query for file scores.
func (d *SQLiteDialect) GetCreateFileScoresMetricsQuery(tableName string) string {
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
			score_hot REAL NOT NULL,
			score_risk REAL NOT NULL,
			score_complexity REAL NOT NULL,
			score_stale REAL NOT NULL,
			score_label TEXT NOT NULL,
			PRIMARY KEY (analysis_id, file_path)
		);
	`, d.QuoteIdentifier(tableName))
}

// BeginAnalysis inserts a new analysis run into SQLite and returns the generated ID.
func (d *SQLiteDialect) BeginAnalysis(db *sql.DB, tableName string, urn string, startTime time.Time, configJSON string) (int64, error) {
	query := fmt.Sprintf(`INSERT INTO %s (start_time, config_params, urn) VALUES (?, ?, ?)`, d.QuoteIdentifier(tableName))
	result, err := db.Exec(query, d.FormatTime(startTime), configJSON, urn)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ScanStartTime parses the start time from an SQLite row (TEXT format).
func (d *SQLiteDialect) ScanStartTime(row *sql.Row) (time.Time, error) {
	var startTimeStr string
	if err := row.Scan(&startTimeStr); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, startTimeStr)
}

// GetUpdateEndAnalysisQuery returns the SQLite-specific query for updating analysis completion.
func (d *SQLiteDialect) GetUpdateEndAnalysisQuery(tableName string) string {
	return fmt.Sprintf(`UPDATE %s SET end_time = ?, run_duration_ms = ?, total_files_analyzed = ? WHERE analysis_id = ?`, d.QuoteIdentifier(tableName))
}

// RecordFileMetricsAndScores inserts file-level metrics and scores into SQLite.
func (d *SQLiteDialect) RecordFileMetricsAndScores(db *sql.DB, tableName string, analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn,
						 contributor_count, age_days, gini_coefficient, file_owner,
						 score_hot, score_risk, score_complexity, score_stale, score_label)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.QuoteIdentifier(tableName))

	_, err := db.Exec(query,
		analysisID, filePath, d.FormatTime(metrics.AnalysisTime), metrics.TotalCommits, metrics.TotalChurn,
		metrics.ContributorCount, metrics.AgeDays, metrics.GiniCoefficient, metrics.FileOwner,
		scores.HotScore, scores.RiskScore, scores.ComplexityScore, scores.StaleScore, scores.ScoreLabel,
	)
	return err
}

// ScanLastRunInfo parses the latest analysis run metadata from SQLite.
func (d *SQLiteDialect) ScanLastRunInfo(row *sql.Row) (int64, time.Time, error) {
	var lastRunID int64
	var lastRunTimeStr string
	if err := row.Scan(&lastRunID, &lastRunTimeStr); err != nil {
		return 0, time.Time{}, err
	}
	lastRunTime, err := time.Parse(time.RFC3339Nano, lastRunTimeStr)
	return lastRunID, lastRunTime, err
}

// ScanOldestRunTime parses the oldest analysis run time from SQLite.
func (d *SQLiteDialect) ScanOldestRunTime(row *sql.Row) (time.Time, error) {
	var oldestRunTimeStr string
	if err := row.Scan(&oldestRunTimeStr); err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, oldestRunTimeStr)
}

// ScanAnalysisRunRecord parses a full analysis run record from SQLite rows.
func (d *SQLiteDialect) ScanAnalysisRunRecord(rows *sql.Rows, record *schema.AnalysisRunRecord) error {
	var startTimeStr string
	var endTimeStr *string
	if err := rows.Scan(&record.AnalysisID, &startTimeStr, &endTimeStr, &record.RunDurationMs, &record.TotalFilesAnalyzed, &record.ConfigParams, &record.URN); err != nil {
		return err
	}
	// Parse start time
	startTime, err := time.Parse(time.RFC3339Nano, startTimeStr)
	if err != nil {
		return err
	}
	record.StartTime = startTime
	// Parse end time if present
	if endTimeStr != nil {
		endTime, err := time.Parse(time.RFC3339Nano, *endTimeStr)
		if err != nil {
			return err
		}
		record.EndTime = &endTime
	}
	return nil
}

// ScanFileScoresMetricsRecord parses a full file metrics and scores record from SQLite rows.
func (d *SQLiteDialect) ScanFileScoresMetricsRecord(rows *sql.Rows, record *schema.FileScoresMetricsRecord) error {
	var analysisTimeStr string
	if err := rows.Scan(&record.AnalysisID, &record.FilePath, &analysisTimeStr, &record.TotalCommits,
		&record.TotalChurn, &record.ContributorCount, &record.AgeDays, &record.GiniCoefficient,
		&record.FileOwner, &record.ScoreHot, &record.ScoreRisk, &record.ScoreComplexity,
		&record.ScoreStale, &record.ScoreLabel); err != nil {
		return err
	}
	analysisTime, err := time.Parse(time.RFC3339Nano, analysisTimeStr)
	if err != nil {
		return err
	}
	record.AnalysisTime = analysisTime
	return nil
}

// FormatTime converts a time.Time to an SQLite-compatible string (RFC3339Nano).
func (d *SQLiteDialect) FormatTime(t time.Time) any {
	return t.Format(time.RFC3339Nano)
}
