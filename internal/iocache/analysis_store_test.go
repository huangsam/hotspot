package iocache

import (
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalysisStore_NoneBackend(t *testing.T) {
	store, err := NewAnalysisStore(schema.NoneBackend, "")
	require.NoError(t, err)
	require.NotNil(t, store)

	// BeginAnalysis should return 0 for NoneBackend
	analysisID, err := store.BeginAnalysis(time.Now(), map[string]any{"test": "value"})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), analysisID)

	// Other operations should not error
	err = store.EndAnalysis(1, time.Now(), 10)
	assert.NoError(t, err)

	err = store.RecordFileMetricsAndScores(1, "test.go", schema.FileMetrics{}, schema.FileScores{})
	assert.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)
}

func TestAnalysisStore_SQLite(t *testing.T) {
	// Use in-memory SQLite for testing
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	// Test BeginAnalysis
	startTime := time.Now()
	configParams := map[string]any{
		"mode":      "hot",
		"lookback":  "30d",
		"repo_path": "/test/repo",
	}
	analysisID, err := store.BeginAnalysis(startTime, configParams)
	require.NoError(t, err)
	assert.Greater(t, analysisID, int64(0))

	// Test RecordFileMetricsAndScores
	metrics := schema.FileMetrics{
		AnalysisTime:     time.Now(),
		TotalCommits:     100,
		TotalChurn:       500,
		ContributorCount: 5,
		AgeDays:          365.0,
		GiniCoefficient:  0.5,
		FileOwner:        "test-owner",
	}
	scores := schema.FileScores{
		AnalysisTime:    time.Now(),
		HotScore:        75.5,
		RiskScore:       80.2,
		ComplexityScore: 65.3,
		StaleScore:      70.1,
		ScoreLabel:      "hot",
	}
	err = store.RecordFileMetricsAndScores(analysisID, "test/file.go", metrics, scores)
	assert.NoError(t, err)

	// Test EndAnalysis
	endTime := time.Now()
	err = store.EndAnalysis(analysisID, endTime, 1)
	assert.NoError(t, err)
}

func TestAnalysisStore_MultipleFiles(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	// Begin analysis
	analysisID, err := store.BeginAnalysis(time.Now(), map[string]any{"test": "multi-file"})
	require.NoError(t, err)

	// Record multiple files
	files := []string{"file1.go", "file2.go", "file3.go"}
	for _, file := range files {
		metrics := schema.FileMetrics{
			AnalysisTime:     time.Now(),
			TotalCommits:     100,
			TotalChurn:       500,
			ContributorCount: 5,
			AgeDays:          365.0,
			GiniCoefficient:  0.5,
			FileOwner:        "owner",
		}
		scores := schema.FileScores{
			AnalysisTime:    time.Now(),
			HotScore:        75.5,
			RiskScore:       80.2,
			ComplexityScore: 65.3,
			StaleScore:      70.1,
			ScoreLabel:      "hot",
		}
		err = store.RecordFileMetricsAndScores(analysisID, file, metrics, scores)
		assert.NoError(t, err)
	}

	// End analysis
	err = store.EndAnalysis(analysisID, time.Now(), len(files))
	assert.NoError(t, err)
}

func TestAnalysisStore_MultipleRuns(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	// Create multiple analysis runs
	var analysisIDs []int64
	for i := range 3 {
		id, err := store.BeginAnalysis(time.Now(), map[string]any{"run": i})
		require.NoError(t, err)
		analysisIDs = append(analysisIDs, id)

		// Record a file for each run
		metrics := schema.FileMetrics{
			AnalysisTime:     time.Now(),
			TotalCommits:     100 + i*10,
			TotalChurn:       500 + i*50,
			ContributorCount: 5,
			AgeDays:          365.0,
			GiniCoefficient:  0.5,
			FileOwner:        "owner",
		}
		scores := schema.FileScores{
			AnalysisTime:    time.Now(),
			HotScore:        75.5 + float64(i),
			RiskScore:       80.2 + float64(i),
			ComplexityScore: 65.3 + float64(i),
			StaleScore:      70.1 + float64(i),
			ScoreLabel:      "hot",
		}
		err = store.RecordFileMetricsAndScores(id, "test.go", metrics, scores)
		assert.NoError(t, err)

		err = store.EndAnalysis(id, time.Now(), 1)
		assert.NoError(t, err)
	}

	// Verify all IDs are unique
	assert.Equal(t, 3, len(analysisIDs))
	assert.NotEqual(t, analysisIDs[0], analysisIDs[1])
	assert.NotEqual(t, analysisIDs[1], analysisIDs[2])
}

func TestAnalysisStore_RuntimeCapture(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	t.Run("runtime calculation", func(t *testing.T) {
		// Start analysis at a known time
		startTime := time.Now().Add(-100 * time.Millisecond) // Start 100ms ago
		analysisID, err := store.BeginAnalysis(startTime, map[string]any{"test": "runtime"})
		require.NoError(t, err)

		// Wait a bit to ensure measurable duration
		time.Sleep(50 * time.Millisecond)

		// End analysis
		endTime := time.Now()
		err = store.EndAnalysis(analysisID, endTime, 1)
		assert.NoError(t, err)

		// Query the database to verify runtime was captured
		db := store.(*AnalysisStoreImpl).db
		var storedStartTime, storedEndTime string
		var storedDurationMs int64

		row := db.QueryRow("SELECT start_time, end_time, run_duration_ms FROM hotspot_analysis_runs WHERE analysis_id = ?", analysisID)
		err = row.Scan(&storedStartTime, &storedEndTime, &storedDurationMs)
		assert.NoError(t, err)

		// Parse stored times
		storedStart, err := time.Parse(time.RFC3339Nano, storedStartTime)
		assert.NoError(t, err)
		storedEnd, err := time.Parse(time.RFC3339Nano, storedEndTime)
		assert.NoError(t, err)

		// Verify duration calculation: should be approximately end - start
		expectedDurationMs := storedEnd.Sub(storedStart).Milliseconds()
		assert.Equal(t, expectedDurationMs, storedDurationMs)

		// Verify duration is reasonable (should be around 150ms ± some tolerance)
		assert.GreaterOrEqual(t, storedDurationMs, int64(100)) // At least 100ms (our initial offset)
		assert.LessOrEqual(t, storedDurationMs, int64(300))    // At most 300ms (allowing for test overhead)
	})

	t.Run("zero duration edge case", func(t *testing.T) {
		// Test with same start and end time
		startTime := time.Now()
		analysisID, err := store.BeginAnalysis(startTime, map[string]any{"test": "zero_duration"})
		require.NoError(t, err)

		// End immediately with same time
		err = store.EndAnalysis(analysisID, startTime, 1)
		assert.NoError(t, err)

		// Verify duration is 0
		db := store.(*AnalysisStoreImpl).db
		var storedDurationMs int64
		row := db.QueryRow("SELECT run_duration_ms FROM hotspot_analysis_runs WHERE analysis_id = ?", analysisID)
		err = row.Scan(&storedDurationMs)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), storedDurationMs)
	})

	t.Run("large duration", func(t *testing.T) {
		// Test with a longer duration
		startTime := time.Now().Add(-5 * time.Second)
		analysisID, err := store.BeginAnalysis(startTime, map[string]any{"test": "large_duration"})
		require.NoError(t, err)

		endTime := time.Now()
		err = store.EndAnalysis(analysisID, endTime, 1)
		assert.NoError(t, err)

		// Verify duration is approximately 5 seconds
		db := store.(*AnalysisStoreImpl).db
		var storedDurationMs int64
		row := db.QueryRow("SELECT run_duration_ms FROM hotspot_analysis_runs WHERE analysis_id = ?", analysisID)
		err = row.Scan(&storedDurationMs)
		assert.NoError(t, err)

		// Should be around 5000ms ± tolerance
		assert.GreaterOrEqual(t, storedDurationMs, int64(4900))
		assert.LessOrEqual(t, storedDurationMs, int64(5100))
	})
}

func TestAnalysisStore_GetAllAnalysisRuns(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	// Test empty store
	runs, err := store.GetAllAnalysisRuns()
	assert.NoError(t, err)
	assert.Empty(t, runs)

	// Add some analysis runs
	startTime := time.Now()
	configs := []map[string]any{
		{"mode": "hot", "lookback": "30d"},
		{"mode": "risk", "lookback": "60d"},
	}

	var analysisIDs []int64
	for _, config := range configs {
		id, err := store.BeginAnalysis(startTime, config)
		require.NoError(t, err)
		analysisIDs = append(analysisIDs, id)

		err = store.EndAnalysis(id, startTime.Add(time.Minute), 1)
		assert.NoError(t, err)
	}

	// Get all runs
	runs, err = store.GetAllAnalysisRuns()
	assert.NoError(t, err)
	assert.Len(t, runs, 2)

	// Verify the runs
	for i, run := range runs {
		assert.Equal(t, analysisIDs[i], run.AnalysisID)
		// ConfigParams is stored as JSON string, so we can't directly compare
		assert.Equal(t, int32(1), run.TotalFilesAnalyzed)
		assert.NotNil(t, run.RunDurationMs)
		assert.Greater(t, *run.RunDurationMs, int32(0))
	}
}

func TestAnalysisStore_GetAllFileScoresMetrics(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	// Test empty store
	metrics, err := store.GetAllFileScoresMetrics()
	assert.NoError(t, err)
	assert.Empty(t, metrics)

	// Add analysis run and file metrics
	analysisID, err := store.BeginAnalysis(time.Now(), map[string]any{"test": "metrics"})
	require.NoError(t, err)

	fileMetrics := schema.FileMetrics{
		AnalysisTime:     time.Now(),
		TotalCommits:     100,
		TotalChurn:       500,
		ContributorCount: 5,
		AgeDays:          365.0,
		GiniCoefficient:  0.5,
		FileOwner:        "test-owner",
	}
	fileScores := schema.FileScores{
		AnalysisTime:    time.Now(),
		HotScore:        75.5,
		RiskScore:       80.2,
		ComplexityScore: 65.3,
		StaleScore:      70.1,
		ScoreLabel:      "hot",
	}

	err = store.RecordFileMetricsAndScores(analysisID, "test/file.go", fileMetrics, fileScores)
	assert.NoError(t, err)

	err = store.EndAnalysis(analysisID, time.Now(), 1)
	assert.NoError(t, err)

	// Get all metrics
	metrics, err = store.GetAllFileScoresMetrics()
	assert.NoError(t, err)
	assert.Len(t, metrics, 1)

	// Verify the metrics
	record := metrics[0]
	assert.Equal(t, analysisID, record.AnalysisID)
	assert.Equal(t, "test/file.go", record.FilePath)
	assert.Equal(t, int32(fileMetrics.TotalCommits), record.TotalCommits)
	assert.Equal(t, int32(fileMetrics.TotalChurn), record.TotalChurn)
	assert.Equal(t, int32(fileMetrics.ContributorCount), record.ContributorCount)
	assert.Equal(t, fileMetrics.AgeDays, record.AgeDays)
	assert.Equal(t, fileMetrics.GiniCoefficient, record.GiniCoefficient)
	assert.Equal(t, fileScores.HotScore, record.ScoreHot)
	assert.Equal(t, fileScores.ScoreLabel, record.ScoreLabel)
}

func TestAnalysisStore_BeginEndAnalysis(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	// Test BeginAnalysis
	startTime := time.Now()
	configParams := map[string]any{"mode": "hot", "workers": 4}
	analysisID, err := store.BeginAnalysis(startTime, configParams)
	assert.NoError(t, err)
	assert.Greater(t, analysisID, int64(0))

	// Test EndAnalysis
	endTime := time.Now()
	totalFiles := 42
	err = store.EndAnalysis(analysisID, endTime, totalFiles)
	assert.NoError(t, err)

	// Verify the data was stored correctly
	runs, err := store.GetAllAnalysisRuns()
	assert.NoError(t, err)
	assert.Len(t, runs, 1)

	run := runs[0]
	assert.Equal(t, analysisID, run.AnalysisID)
	assert.Equal(t, int32(totalFiles), run.TotalFilesAnalyzed)
	assert.NotNil(t, run.RunDurationMs)
}

func TestAnalysisStore_RecordFileMetricsAndScores(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
	require.NoError(t, err)
	require.NotNil(t, store)
	defer func() { _ = store.Close() }()

	// Create analysis run
	analysisID, err := store.BeginAnalysis(time.Now(), map[string]any{"test": "record"})
	require.NoError(t, err)

	// Test recording metrics and scores
	filePath := "src/main.go"
	metrics := schema.FileMetrics{
		AnalysisTime:     time.Now(),
		TotalCommits:     150,
		TotalChurn:       750,
		ContributorCount: 8,
		AgeDays:          200.5,
		GiniCoefficient:  0.3,
		FileOwner:        "developer@example.com",
	}
	scores := schema.FileScores{
		AnalysisTime:    time.Now(),
		HotScore:        85.2,
		RiskScore:       72.1,
		ComplexityScore: 90.5,
		StaleScore:      45.3,
		ScoreLabel:      "complexity",
	}

	err = store.RecordFileMetricsAndScores(analysisID, filePath, metrics, scores)
	assert.NoError(t, err)

	// Verify the data was stored
	fileMetrics, err := store.GetAllFileScoresMetrics()
	assert.NoError(t, err)
	assert.Len(t, fileMetrics, 1)

	record := fileMetrics[0]
	assert.Equal(t, analysisID, record.AnalysisID)
	assert.Equal(t, filePath, record.FilePath)
	assert.Equal(t, int32(metrics.TotalCommits), record.TotalCommits)
	assert.Equal(t, scores.HotScore, record.ScoreHot)
	assert.Equal(t, scores.ScoreLabel, record.ScoreLabel)
}
