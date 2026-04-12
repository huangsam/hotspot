package iocache

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// MySQLDialect handles MySQL-specific SQL syntax and data types.
type MySQLDialect struct{}

// DriverName returns the driver name for MySQL.
func (d *MySQLDialect) DriverName() string {
	return "mysql"
}

// QuoteIdentifier returns a backtick-quoted identifier for MySQL.
func (d *MySQLDialect) QuoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", name)
}

// GetCreateAnalysisRunsQuery returns the MySQL-specific table creation query for analysis runs.
func (d *MySQLDialect) GetCreateAnalysisRunsQuery(tableName string) string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			analysis_id BIGINT AUTO_INCREMENT PRIMARY KEY,
			start_time DATETIME(6) NOT NULL,
			end_time DATETIME(6),
			run_duration_ms INT,
			total_files_analyzed INT,
			config_params TEXT,
			urn VARCHAR(255)
		);
	`, d.QuoteIdentifier(tableName))
}

// GetCreateFileScoresMetricsQuery returns the MySQL-specific table creation query for file scores.
func (d *MySQLDialect) GetCreateFileScoresMetricsQuery(tableName string) string {
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
			score_hot DOUBLE NOT NULL,
			score_risk DOUBLE NOT NULL,
			score_complexity DOUBLE NOT NULL,
			score_stale DOUBLE NOT NULL,
			score_label VARCHAR(50) NOT NULL,
			PRIMARY KEY (analysis_id, file_path)
		);
	`, d.QuoteIdentifier(tableName))
}

// BeginAnalysis inserts a new analysis run into MySQL and returns the generated ID.
func (d *MySQLDialect) BeginAnalysis(db *sql.DB, tableName string, urn string, startTime time.Time, configJSON string) (int64, error) {
	query := fmt.Sprintf(`INSERT INTO %s (start_time, config_params, urn) VALUES (?, ?, ?)`, d.QuoteIdentifier(tableName))
	result, err := db.Exec(query, d.FormatTime(startTime), configJSON, urn)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// ScanStartTime parses the start time from a MySQL row.
func (d *MySQLDialect) ScanStartTime(row *sql.Row) (time.Time, error) {
	var startTime time.Time
	if err := row.Scan(&startTime); err != nil {
		return time.Time{}, err
	}
	return startTime, nil
}

// GetUpdateEndAnalysisQuery returns the MySQL-specific query for updating analysis completion.
func (d *MySQLDialect) GetUpdateEndAnalysisQuery(tableName string) string {
	return fmt.Sprintf(`UPDATE %s SET end_time = ?, run_duration_ms = ?, total_files_analyzed = ? WHERE analysis_id = ?`, d.QuoteIdentifier(tableName))
}

// RecordFileMetricsAndScores inserts file-level metrics and scores into MySQL.
func (d *MySQLDialect) RecordFileMetricsAndScores(db *sql.DB, tableName string, analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error {
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

// ScanLastRunInfo parses the latest analysis run metadata from MySQL.
func (d *MySQLDialect) ScanLastRunInfo(row *sql.Row) (int64, time.Time, error) {
	var lastRunID int64
	var lastRunTime time.Time
	if err := row.Scan(&lastRunID, &lastRunTime); err != nil {
		return 0, time.Time{}, err
	}
	return lastRunID, lastRunTime, nil
}

// ScanOldestRunTime parses the oldest analysis run time from MySQL.
func (d *MySQLDialect) ScanOldestRunTime(row *sql.Row) (time.Time, error) {
	var oldestRunTime time.Time
	if err := row.Scan(&oldestRunTime); err != nil {
		return time.Time{}, err
	}
	return oldestRunTime, nil
}

// ScanAnalysisRunRecord parses a full analysis run record from MySQL rows.
func (d *MySQLDialect) ScanAnalysisRunRecord(rows *sql.Rows, record *schema.AnalysisRunRecord) error {
	return rows.Scan(&record.AnalysisID, &record.StartTime, &record.EndTime, &record.RunDurationMs, &record.TotalFilesAnalyzed, &record.ConfigParams, &record.URN)
}

// ScanFileScoresMetricsRecord parses a full file metrics and scores record from MySQL rows.
func (d *MySQLDialect) ScanFileScoresMetricsRecord(rows *sql.Rows, record *schema.FileScoresMetricsRecord) error {
	return rows.Scan(&record.AnalysisID, &record.FilePath, &record.AnalysisTime, &record.TotalCommits,
		&record.TotalChurn, &record.ContributorCount, &record.AgeDays, &record.GiniCoefficient,
		&record.FileOwner, &record.ScoreHot, &record.ScoreRisk, &record.ScoreComplexity,
		&record.ScoreStale, &record.ScoreLabel)
}

// FormatTime converts a time.Time to a MySQL-compatible format (passing native time.Time works).
func (d *MySQLDialect) FormatTime(t time.Time) any {
	return t
}
