package iocache

import (
	"database/sql"
	"encoding/json"
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
	return d.RecordFileResultsBatch(db, tableName, analysisID, []schema.BatchFileResult{
		{
			Path:    filePath,
			Metrics: metrics,
			Scores:  scores,
		},
	})
}

// RecordFileResultsBatch inserts multiple file metrics and scores into SQLite using a single transaction.
func (d *SQLiteDialect) RecordFileResultsBatch(db *sql.DB, tableName string, analysisID int64, results []schema.BatchFileResult) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	query := fmt.Sprintf(`
		INSERT INTO %s (analysis_id, file_path, analysis_time, total_commits, total_churn, lines_added, lines_deleted, decayed_commits, decayed_churn, lines_of_code,
						 contributor_count, recent_commits, recent_churn, recent_lines_added, recent_lines_deleted, recent_contributor_count,
						 age_days, gini_coefficient, file_owner,
						 score_hot, score_risk, score_complexity, score_roi, score_label, reasoning,
						 recency_signal, recency_threshold_low, recency_threshold_high)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.QuoteIdentifier(tableName))

	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, res := range results {
		reasoningJSON, _ := json.Marshal(res.Scores.Reasoning)
		_, err := stmt.Exec(
			analysisID, res.Path, d.FormatTime(res.Metrics.AnalysisTime), res.Metrics.TotalCommits, res.Metrics.TotalChurn, res.Metrics.LinesAdded, res.Metrics.LinesDeleted, res.Metrics.DecayedCommits, res.Metrics.DecayedChurn, res.Metrics.LinesOfCode,
			res.Metrics.ContributorCount, res.Metrics.RecentCommits, res.Metrics.RecentChurn, res.Metrics.RecentLinesAdded, res.Metrics.RecentLinesDeleted, res.Metrics.RecentContributorCount,
			res.Metrics.AgeDays, res.Metrics.GiniCoefficient, res.Metrics.FileOwner,
			res.Scores.HotScore, res.Scores.RiskScore, res.Scores.ComplexityScore, res.Scores.ROIScore, res.Scores.ScoreLabel, string(reasoningJSON),
			res.Metrics.RecencySignal, res.Metrics.RecencyThresholdLow, res.Metrics.RecencyThresholdHigh,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
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
	var urn *string
	if err := rows.Scan(&record.AnalysisID, &startTimeStr, &endTimeStr, &record.RunDurationMs, &record.TotalFilesAnalyzed, &record.ConfigParams, &urn); err != nil {
		return err
	}
	if urn != nil {
		record.URN = *urn
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
	var reasoningJSON []byte
	if err := rows.Scan(&record.AnalysisID, &record.FilePath, &analysisTimeStr, &record.TotalCommits,
		&record.TotalChurn, &record.LinesAdded, &record.LinesDeleted, &record.DecayedCommits, &record.DecayedChurn, &record.LinesOfCode, &record.ContributorCount,
		&record.RecentCommits, &record.RecentChurn, &record.RecentLinesAdded, &record.RecentLinesDeleted, &record.RecentContributorCount,
		&record.AgeDays, &record.GiniCoefficient,
		&record.FileOwner, &record.ScoreHot, &record.ScoreRisk, &record.ScoreComplexity, &record.ScoreROI,
		&record.ScoreLabel, &reasoningJSON,
		&record.RecencySignal, &record.RecencyThresholdLow, &record.RecencyThresholdHigh); err != nil {
		return err
	}
	analysisTime, err := time.Parse(time.RFC3339Nano, analysisTimeStr)
	if err != nil {
		return err
	}
	record.AnalysisTime = analysisTime
	if len(reasoningJSON) > 0 {
		_ = json.Unmarshal(reasoningJSON, &record.Reasoning)
	}
	return nil
}

// FormatTime converts a time.Time to an SQLite-compatible string (RFC3339Nano).
func (d *SQLiteDialect) FormatTime(t time.Time) any {
	return t.Format(time.RFC3339Nano)
}

// Placeholder returns an SQLite-compatible placeholder (?).
func (d *SQLiteDialect) Placeholder(_ int) string {
	return "?"
}

// GetSelectStartTimeQuery returns the SQLite-specific query for selecting start_time.
func (d *SQLiteDialect) GetSelectStartTimeQuery(tableName string) string {
	return fmt.Sprintf(`SELECT start_time FROM %s WHERE analysis_id = ?`, d.QuoteIdentifier(tableName))
}

// GetUpdateURNQuery returns the SQLite-specific query for updating analysis URN.
func (d *SQLiteDialect) GetUpdateURNQuery(tableName string) string {
	return fmt.Sprintf(`UPDATE %s SET urn = ? WHERE analysis_id = ?`, d.QuoteIdentifier(tableName))
}
