package iocache

import (
	"database/sql"
	"time"

	"github.com/huangsam/hotspot/schema"
)

// SQLDialect encapsulates all backend-specific SQL logic for the AnalysisStore.
type SQLDialect interface {
	// DriverName returns the database driver name (e.g., "sqlite", "mysql", "pgx").
	DriverName() string

	// QuoteIdentifier returns a quoted SQL identifier for the backend.
	QuoteIdentifier(name string) string

	// BeginAnalysis inserts a new analysis run and returns its ID.
	BeginAnalysis(db *sql.DB, tableName string, urn string, startTime time.Time, configJSON string) (int64, error)

	// ScanStartTime parses the start_time from a database row.
	ScanStartTime(row *sql.Row) (time.Time, error)

	// GetUpdateEndAnalysisQuery returns the query to finalize an analysis run.
	GetUpdateEndAnalysisQuery(tableName string) string

	// RecordFileMetricsAndScores inserts metrics and scores for a specific file.
	RecordFileMetricsAndScores(db *sql.DB, tableName string, analysisID int64, filePath string, metrics schema.FileMetrics, scores schema.FileScores) error

	// ScanLastRunInfo parses the latest analysis run info.
	ScanLastRunInfo(row *sql.Row) (int64, time.Time, error)

	// ScanOldestRunTime parses the oldest analysis run time.
	ScanOldestRunTime(row *sql.Row) (time.Time, error)

	// ScanAnalysisRunRecord parses an AnalysisRunRecord from rows.
	ScanAnalysisRunRecord(rows *sql.Rows, record *schema.AnalysisRunRecord) error

	// ScanFileScoresMetricsRecord parses a FileScoresMetricsRecord from rows.
	ScanFileScoresMetricsRecord(rows *sql.Rows, record *schema.FileScoresMetricsRecord) error

	// FormatTime converts a time.Time to a backend-compatible value.
	FormatTime(t time.Time) any
}
