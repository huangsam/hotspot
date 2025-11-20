package parquet

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalysisRunStructTags(t *testing.T) {
	// Verify struct tags are properly defined for parquet schema inference
	schema := parquet.SchemaOf(new(AnalysisRun))
	require.NotNil(t, schema)

	// Check that all expected columns exist
	expectedColumns := []string{
		"analysis_id",
		"start_time",
		"end_time",
		"run_duration_ms",
		"total_files_analyzed",
		"config_params",
	}

	for _, colName := range expectedColumns {
		col, ok := schema.Lookup(colName)
		require.True(t, ok, "Column %s should exist in schema", colName)
		require.NotNil(t, col, "Column %s should not be nil", colName)
	}
}

func TestFileScoresMetricsStructTags(t *testing.T) {
	// Verify struct tags are properly defined for parquet schema inference
	schema := parquet.SchemaOf(new(FileScoresMetrics))
	require.NotNil(t, schema)

	// Check that all expected columns exist
	expectedColumns := []string{
		"analysis_id",
		"file_path",
		"analysis_time",
		"total_commits",
		"total_churn",
		"contributor_count",
		"age_days",
		"gini_coefficient",
		"file_owner",
		"score_hot",
		"score_risk",
		"score_complexity",
		"score_stale",
		"score_label",
	}

	for _, colName := range expectedColumns {
		col, ok := schema.Lookup(colName)
		require.True(t, ok, "Column %s should exist in schema", colName)
		require.NotNil(t, col, "Column %s should not be nil", colName)
	}
}

func TestWriteAnalysisRunsParquet(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "analysis_runs.parquet")

	// Get mock data
	data := MockFetchAnalysisRuns()
	require.NotEmpty(t, data, "Mock data should not be empty")

	// Write data to Parquet file
	err := WriteAnalysisRunsParquet(data, outputPath)
	require.NoError(t, err, "Writing Parquet file should not produce error")

	// Verify file was created
	info, err := os.Stat(outputPath)
	require.NoError(t, err, "Output file should exist")
	assert.Greater(t, info.Size(), int64(0), "Output file should not be empty")

	// Read back and verify data
	file, err := os.Open(outputPath)
	require.NoError(t, err, "Should be able to open output file")
	defer file.Close()

	reader := parquet.NewGenericReader[AnalysisRun](file)
	defer reader.Close()

	// Read all rows
	readData := make([]AnalysisRun, reader.NumRows())
	n, err := reader.Read(readData)
	if err != nil && err != io.EOF {
		require.NoError(t, err, "Should be able to read data")
	}
	assert.Equal(t, len(data), n, "Should read all records")

	// Verify data integrity
	for i := 0; i < len(data); i++ {
		assert.Equal(t, data[i].AnalysisID, readData[i].AnalysisID, "AnalysisID should match")
		assert.Equal(t, data[i].TotalFilesAnalyzed, readData[i].TotalFilesAnalyzed, "TotalFilesAnalyzed should match")

		// Check nullable fields
		if data[i].EndTime == nil {
			assert.Nil(t, readData[i].EndTime, "EndTime should be nil")
		} else {
			require.NotNil(t, readData[i].EndTime, "EndTime should not be nil")
			assert.WithinDuration(t, *data[i].EndTime, *readData[i].EndTime, time.Nanosecond, "EndTime should match within nanosecond precision")
		}

		if data[i].RunDurationMs == nil {
			assert.Nil(t, readData[i].RunDurationMs, "RunDurationMs should be nil")
		} else {
			require.NotNil(t, readData[i].RunDurationMs, "RunDurationMs should not be nil")
			assert.Equal(t, *data[i].RunDurationMs, *readData[i].RunDurationMs, "RunDurationMs should match")
		}

		if data[i].ConfigParams == nil {
			assert.Nil(t, readData[i].ConfigParams, "ConfigParams should be nil")
		} else {
			require.NotNil(t, readData[i].ConfigParams, "ConfigParams should not be nil")
			assert.Equal(t, *data[i].ConfigParams, *readData[i].ConfigParams, "ConfigParams should match")
		}
	}
}

func TestWriteFileScoresMetricsParquet(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "file_scores_metrics.parquet")

	// Get mock data
	data := MockFetchFileScoresMetrics()
	require.NotEmpty(t, data, "Mock data should not be empty")

	// Write data to Parquet file
	err := WriteFileScoresMetricsParquet(data, outputPath)
	require.NoError(t, err, "Writing Parquet file should not produce error")

	// Verify file was created
	info, err := os.Stat(outputPath)
	require.NoError(t, err, "Output file should exist")
	assert.Greater(t, info.Size(), int64(0), "Output file should not be empty")

	// Read back and verify data
	file, err := os.Open(outputPath)
	require.NoError(t, err, "Should be able to open output file")
	defer file.Close()

	reader := parquet.NewGenericReader[FileScoresMetrics](file)
	defer reader.Close()

	// Read all rows
	readData := make([]FileScoresMetrics, reader.NumRows())
	n, err := reader.Read(readData)
	if err != nil && err != io.EOF {
		require.NoError(t, err, "Should be able to read data")
	}
	assert.Equal(t, len(data), n, "Should read all records")

	// Verify data integrity
	for i := 0; i < len(data); i++ {
		assert.Equal(t, data[i].AnalysisID, readData[i].AnalysisID, "AnalysisID should match")
		assert.Equal(t, data[i].FilePath, readData[i].FilePath, "FilePath should match")
		assert.Equal(t, data[i].TotalCommits, readData[i].TotalCommits, "TotalCommits should match")
		assert.Equal(t, data[i].TotalChurn, readData[i].TotalChurn, "TotalChurn should match")
		assert.Equal(t, data[i].ContributorCount, readData[i].ContributorCount, "ContributorCount should match")
		assert.InDelta(t, data[i].AgeDays, readData[i].AgeDays, 0.01, "AgeDays should match")
		assert.InDelta(t, data[i].GiniCoefficient, readData[i].GiniCoefficient, 0.001, "GiniCoefficient should match")
		assert.InDelta(t, data[i].ScoreHot, readData[i].ScoreHot, 0.01, "ScoreHot should match")
		assert.InDelta(t, data[i].ScoreRisk, readData[i].ScoreRisk, 0.01, "ScoreRisk should match")
		assert.InDelta(t, data[i].ScoreComplexity, readData[i].ScoreComplexity, 0.01, "ScoreComplexity should match")
		assert.InDelta(t, data[i].ScoreStale, readData[i].ScoreStale, 0.01, "ScoreStale should match")
		assert.Equal(t, data[i].ScoreLabel, readData[i].ScoreLabel, "ScoreLabel should match")

		// Check nullable FileOwner field
		if data[i].FileOwner == nil {
			assert.Nil(t, readData[i].FileOwner, "FileOwner should be nil")
		} else {
			require.NotNil(t, readData[i].FileOwner, "FileOwner should not be nil")
			assert.Equal(t, *data[i].FileOwner, *readData[i].FileOwner, "FileOwner should match")
		}
	}
}

func TestWriteAnalysisRunsParquet_EmptyData(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty_analysis_runs.parquet")

	// Write empty data
	err := WriteAnalysisRunsParquet([]AnalysisRun{}, outputPath)
	require.NoError(t, err, "Writing empty data should not produce error")

	// Verify file was created
	info, err := os.Stat(outputPath)
	require.NoError(t, err, "Output file should exist")
	assert.Greater(t, info.Size(), int64(0), "Output file should contain schema even if empty")
}

func TestWriteFileScoresMetricsParquet_EmptyData(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty_file_scores.parquet")

	// Write empty data
	err := WriteFileScoresMetricsParquet([]FileScoresMetrics{}, outputPath)
	require.NoError(t, err, "Writing empty data should not produce error")

	// Verify file was created
	info, err := os.Stat(outputPath)
	require.NoError(t, err, "Output file should exist")
	assert.Greater(t, info.Size(), int64(0), "Output file should contain schema even if empty")
}

func TestWriteAnalysisRunsParquet_InvalidPath(t *testing.T) {
	// Try to write to invalid path
	data := MockFetchAnalysisRuns()
	err := WriteAnalysisRunsParquet(data, "/nonexistent/directory/output.parquet")
	require.Error(t, err, "Writing to invalid path should produce error")
}

func TestWriteFileScoresMetricsParquet_InvalidPath(t *testing.T) {
	// Try to write to invalid path
	data := MockFetchFileScoresMetrics()
	err := WriteFileScoresMetricsParquet(data, "/nonexistent/directory/output.parquet")
	require.Error(t, err, "Writing to invalid path should produce error")
}

func TestMockFetchAnalysisRuns(t *testing.T) {
	data := MockFetchAnalysisRuns()
	require.NotEmpty(t, data, "Mock data should not be empty")
	assert.Len(t, data, 3, "Should return 3 mock records")

	// Verify the structure of mock data
	assert.Equal(t, int64(1), data[0].AnalysisID)
	assert.NotNil(t, data[0].EndTime, "First record should have EndTime")
	assert.NotNil(t, data[0].RunDurationMs, "First record should have RunDurationMs")
	assert.NotNil(t, data[0].ConfigParams, "First record should have ConfigParams")

	// Third record should have nil nullable fields
	assert.Equal(t, int64(3), data[2].AnalysisID)
	assert.Nil(t, data[2].EndTime, "Third record should have nil EndTime")
	assert.Nil(t, data[2].RunDurationMs, "Third record should have nil RunDurationMs")
	assert.Nil(t, data[2].ConfigParams, "Third record should have nil ConfigParams")
}

func TestMockFetchFileScoresMetrics(t *testing.T) {
	data := MockFetchFileScoresMetrics()
	require.NotEmpty(t, data, "Mock data should not be empty")
	assert.Len(t, data, 3, "Should return 3 mock records")

	// Verify the structure of mock data
	assert.Equal(t, int64(1), data[0].AnalysisID)
	assert.Equal(t, "src/main.go", data[0].FilePath)
	assert.NotNil(t, data[0].FileOwner, "First record should have FileOwner")

	// Third record should have nil FileOwner
	assert.Equal(t, int64(2), data[2].AnalysisID)
	assert.Nil(t, data[2].FileOwner, "Third record should have nil FileOwner")
}

func TestNullableFieldHandling(t *testing.T) {
	// Test that we can create structs with various combinations of null fields
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "nullable_test.parquet")

	now := time.Now()
	endTime := now.Add(1 * time.Hour)
	durationMs := int32(3600000)
	config := `{"test":"config"}`

	testData := []AnalysisRun{
		// All fields populated
		{
			AnalysisID:         1,
			StartTime:          now,
			EndTime:            &endTime,
			RunDurationMs:      &durationMs,
			TotalFilesAnalyzed: 100,
			ConfigParams:       &config,
		},
		// All nullable fields are nil
		{
			AnalysisID:         2,
			StartTime:          now,
			EndTime:            nil,
			RunDurationMs:      nil,
			TotalFilesAnalyzed: 0,
			ConfigParams:       nil,
		},
	}

	// Write and read back
	err := WriteAnalysisRunsParquet(testData, outputPath)
	require.NoError(t, err)

	// Read back and verify
	file, err := os.Open(outputPath)
	require.NoError(t, err)
	defer file.Close()

	reader := parquet.NewGenericReader[AnalysisRun](file)
	defer reader.Close()

	readData := make([]AnalysisRun, reader.NumRows())
	n, err := reader.Read(readData)
	if err != nil && err != io.EOF {
		require.NoError(t, err)
	}
	assert.Equal(t, len(testData), n)

	// Verify first record has all fields
	assert.NotNil(t, readData[0].EndTime)
	assert.NotNil(t, readData[0].RunDurationMs)
	assert.NotNil(t, readData[0].ConfigParams)

	// Verify second record has nil nullable fields
	assert.Nil(t, readData[1].EndTime)
	assert.Nil(t, readData[1].RunDurationMs)
	assert.Nil(t, readData[1].ConfigParams)
}

func TestTimestampPrecision(t *testing.T) {
	// Test that timestamps are stored with nanosecond precision
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "timestamp_test.parquet")

	// Create a timestamp with nanosecond precision
	now := time.Now()
	// Note: Parquet stores timestamps with nanosecond precision internally

	testData := []AnalysisRun{
		{
			AnalysisID:         1,
			StartTime:          now,
			EndTime:            &now,
			RunDurationMs:      nil,
			TotalFilesAnalyzed: 0,
			ConfigParams:       nil,
		},
	}

	// Write and read back
	err := WriteAnalysisRunsParquet(testData, outputPath)
	require.NoError(t, err)

	file, err := os.Open(outputPath)
	require.NoError(t, err)
	defer file.Close()

	reader := parquet.NewGenericReader[AnalysisRun](file)
	defer reader.Close()

	readData := make([]AnalysisRun, reader.NumRows())
	_, err = reader.Read(readData)
	if err != nil && err != io.EOF {
		require.NoError(t, err)
	}

	// Verify timestamp precision (should be within nanosecond)
	assert.WithinDuration(t, testData[0].StartTime, readData[0].StartTime, time.Nanosecond)
	assert.WithinDuration(t, *testData[0].EndTime, *readData[0].EndTime, time.Nanosecond)
}
