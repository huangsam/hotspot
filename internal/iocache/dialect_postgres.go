package iocache

import (
	"database/sql"
	"encoding/json"
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
	reasoningJSON, _ := json.Marshal(scores.Reasoning)
	query := fmt.Sprintf(`
		INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn, lines_added, lines_deleted, decayed_commits, decayed_churn, lines_of_code,
						 contributor_count, recent_commits, recent_churn, recent_lines_added, recent_lines_deleted, recent_contributor_count,
						 age_days, gini_coefficient, file_owner,
						 score_hot, score_risk, score_complexity, score_roi, score_label, reasoning,
						 recency_signal, recency_threshold_low, recency_threshold_high)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
	`, d.QuoteIdentifier(tableName))

	_, err := db.Exec(query,
		analysisID, filePath, d.FormatTime(metrics.AnalysisTime), metrics.TotalCommits, metrics.TotalChurn, metrics.LinesAdded, metrics.LinesDeleted, metrics.DecayedCommits, metrics.DecayedChurn, metrics.LinesOfCode,
		metrics.ContributorCount, metrics.RecentCommits, metrics.RecentChurn, metrics.RecentLinesAdded, metrics.RecentLinesDeleted, metrics.RecentContributorCount,
		metrics.AgeDays, metrics.GiniCoefficient, metrics.FileOwner,
		scores.HotScore, scores.RiskScore, scores.ComplexityScore, scores.ROIScore, scores.ScoreLabel, reasoningJSON,
		metrics.RecencySignal, metrics.RecencyThresholdLow, metrics.RecencyThresholdHigh,
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
	var urn *string
	if err := rows.Scan(&record.AnalysisID, &record.StartTime, &record.EndTime, &record.RunDurationMs, &record.TotalFilesAnalyzed, &record.ConfigParams, &urn); err != nil {
		return err
	}
	if urn != nil {
		record.URN = *urn
	}
	return nil
}

// ScanFileScoresMetricsRecord parses a full file metrics and scores record from PostgreSQL rows.
func (d *PostgresDialect) ScanFileScoresMetricsRecord(rows *sql.Rows, record *schema.FileScoresMetricsRecord) error {
	var reasoningJSON []byte
	if err := rows.Scan(&record.AnalysisID, &record.FilePath, &record.AnalysisTime, &record.TotalCommits,
		&record.TotalChurn, &record.LinesAdded, &record.LinesDeleted, &record.DecayedCommits, &record.DecayedChurn, &record.LinesOfCode, &record.ContributorCount,
		&record.RecentCommits, &record.RecentChurn, &record.RecentLinesAdded, &record.RecentLinesDeleted, &record.RecentContributorCount,
		&record.AgeDays, &record.GiniCoefficient,
		&record.FileOwner, &record.ScoreHot, &record.ScoreRisk, &record.ScoreComplexity, &record.ScoreROI,
		&record.ScoreLabel, &reasoningJSON,
		&record.RecencySignal, &record.RecencyThresholdLow, &record.RecencyThresholdHigh); err != nil {
		return err
	}
	if len(reasoningJSON) > 0 {
		_ = json.Unmarshal(reasoningJSON, &record.Reasoning)
	}
	return nil
}

// FormatTime converts a time.Time to a PostgreSQL-compatible format (passing native time.Time works).
func (d *PostgresDialect) FormatTime(t time.Time) any {
	return t
}

// Placeholder returns a PostgreSQL-compatible placeholder ($N).
func (d *PostgresDialect) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// GetSelectStartTimeQuery returns the PostgreSQL-specific query for selecting start_time.
func (d *PostgresDialect) GetSelectStartTimeQuery(tableName string) string {
	return fmt.Sprintf(`SELECT start_time FROM %s WHERE analysis_id = $1`, d.QuoteIdentifier(tableName))
}

// GetUpdateURNQuery returns the PostgreSQL-specific query for updating analysis URN.
func (d *PostgresDialect) GetUpdateURNQuery(tableName string) string {
	return fmt.Sprintf(`UPDATE %s SET urn = $1 WHERE analysis_id = $2`, d.QuoteIdentifier(tableName))
}
