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
		INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn, lines_added, lines_deleted, decayed_commits, decayed_churn, lines_of_code,
						 contributor_count, recent_commits, recent_churn, recent_lines_added, recent_lines_deleted, recent_contributor_count,
						 age_days, gini_coefficient, file_owner,
						 score_hot, score_risk, score_complexity, score_label)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.QuoteIdentifier(tableName))

	_, err := db.Exec(query,
		analysisID, filePath, d.FormatTime(metrics.AnalysisTime), metrics.TotalCommits, metrics.TotalChurn, metrics.LinesAdded, metrics.LinesDeleted, metrics.DecayedCommits, metrics.DecayedChurn, metrics.LinesOfCode,
		metrics.ContributorCount, metrics.RecentCommits, metrics.RecentChurn, metrics.RecentLinesAdded, metrics.RecentLinesDeleted, metrics.RecentContributorCount,
		metrics.AgeDays, metrics.GiniCoefficient, metrics.FileOwner,
		scores.HotScore, scores.RiskScore, scores.ComplexityScore, scores.ScoreLabel,
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
	var urn *string
	if err := rows.Scan(&record.AnalysisID, &record.StartTime, &record.EndTime, &record.RunDurationMs, &record.TotalFilesAnalyzed, &record.ConfigParams, &urn); err != nil {
		return err
	}
	if urn != nil {
		record.URN = *urn
	}
	return nil
}

// ScanFileScoresMetricsRecord parses a full file metrics and scores record from MySQL rows.
func (d *MySQLDialect) ScanFileScoresMetricsRecord(rows *sql.Rows, record *schema.FileScoresMetricsRecord) error {
	return rows.Scan(&record.AnalysisID, &record.FilePath, &record.AnalysisTime, &record.TotalCommits,
		&record.TotalChurn, &record.LinesAdded, &record.LinesDeleted, &record.DecayedCommits, &record.DecayedChurn, &record.LinesOfCode, &record.ContributorCount,
		&record.RecentCommits, &record.RecentChurn, &record.RecentLinesAdded, &record.RecentLinesDeleted, &record.RecentContributorCount,
		&record.AgeDays, &record.GiniCoefficient,
		&record.FileOwner, &record.ScoreHot, &record.ScoreRisk, &record.ScoreComplexity,
		&record.ScoreLabel)
}

// FormatTime converts a time.Time to a MySQL-compatible format (passing native time.Time works).
func (d *MySQLDialect) FormatTime(t time.Time) any {
	return t
}

// Placeholder returns a MySQL-compatible placeholder (?).
func (d *MySQLDialect) Placeholder(_ int) string {
	return "?"
}

// GetSelectStartTimeQuery returns the MySQL-specific query for selecting start_time.
func (d *MySQLDialect) GetSelectStartTimeQuery(tableName string) string {
	return fmt.Sprintf(`SELECT start_time FROM %s WHERE analysis_id = ?`, d.QuoteIdentifier(tableName))
}

// GetUpdateURNQuery returns the MySQL-specific query for updating analysis URN.
func (d *MySQLDialect) GetUpdateURNQuery(tableName string) string {
	return fmt.Sprintf(`UPDATE %s SET urn = ? WHERE analysis_id = ?`, d.QuoteIdentifier(tableName))
}
