package iocache

import (
	"database/sql"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// SQLDialect encapsulates all backend-specific SQL logic for the AnalysisStore.
type SQLDialect interface {
	SQLBaseDialect
	SQLQueryDialect
	SQLScanner
}

// SQLBaseDialect defines core SQL primitive logic like identifier quoting and driver naming.
type SQLBaseDialect interface {
	// DriverName returns the database driver name (e.g., "sqlite", "mysql", "pgx").
	DriverName() string

	// QuoteIdentifier returns a quoted SQL identifier for the backend.
	QuoteIdentifier(name string) string

	// Placeholder returns a backend-appropriate placeholder string (e.g., "$1" or "?").
	Placeholder(index int) string

	// FormatTime converts a time.Time to a backend-compatible value.
	FormatTime(t time.Time) any
}

// SQLQueryDialect defines logic for constructing and executing analysis-specific SQL queries.
type SQLQueryDialect interface {
	// BeginAnalysis inserts a new analysis run and returns its ID.
	BeginAnalysis(db *sql.DB, tableName string, urn string, startTime time.Time, configJSON string) (int64, error)

	// GetUpdateEndAnalysisQuery returns the query to finalize an analysis run.
	GetUpdateEndAnalysisQuery(tableName string) string

	// GetSelectStartTimeQuery returns the query to select the start_time of an analysis run.
	GetSelectStartTimeQuery(tableName string) string

	// GetUpdateURNQuery returns the query to update the URN of an analysis run.
	GetUpdateURNQuery(tableName string) string

	// RecordFileMetricsAndScores inserts metrics and scores for a specific file.
	RecordFileMetricsAndScores(db *sql.DB, tableName string, analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error
}

// SQLScanner defines logic for mapping database rows and results back to schema structs.
type SQLScanner interface {
	// ScanStartTime parses the start_time from a database row.
	ScanStartTime(row *sql.Row) (time.Time, error)

	// ScanLastRunInfo parses the latest analysis run info.
	ScanLastRunInfo(row *sql.Row) (int64, time.Time, error)

	// ScanOldestRunTime parses the oldest analysis run time.
	ScanOldestRunTime(row *sql.Row) (time.Time, error)

	// ScanAnalysisRunRecord parses an AnalysisRunRecord from rows.
	ScanAnalysisRunRecord(rows *sql.Rows, record *schema.AnalysisRunRecord) error

	// ScanFileScoresMetricsRecord parses a FileScoresMetricsRecord from rows.
	ScanFileScoresMetricsRecord(rows *sql.Rows, record *schema.FileScoresMetricsRecord) error
}
