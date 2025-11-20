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

	err = store.RecordFileMetrics(1, "test.go", schema.FileMetrics{})
	assert.NoError(t, err)

	err = store.RecordFileScores(1, "test.go", schema.FileScores{})
	assert.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)
}

func TestAnalysisStore_SQLite(t *testing.T) {
	// Use in-memory SQLite for testing
	store, err := NewAnalysisStore(schema.SQLiteBackend, "")
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

	// Test RecordFileMetrics
	metrics := schema.FileMetrics{
		AnalysisTime:     time.Now(),
		TotalCommits:     100,
		TotalChurn:       500,
		ContributorCount: 5,
		AgeDays:          365.0,
		GiniCoefficient:  0.5,
		FileOwner:        "test-owner",
	}
	err = store.RecordFileMetrics(analysisID, "test/file.go", metrics)
	assert.NoError(t, err)

	// Test RecordFileScores
	scores := schema.FileScores{
		AnalysisTime:    time.Now(),
		HotScore:        75.5,
		RiskScore:       80.2,
		ComplexityScore: 65.3,
		StaleScore:      70.1,
		ScoreLabel:      "hot",
	}
	err = store.RecordFileScores(analysisID, "test/file.go", scores)
	assert.NoError(t, err)

	// Test EndAnalysis
	endTime := time.Now()
	err = store.EndAnalysis(analysisID, endTime, 1)
	assert.NoError(t, err)
}

func TestAnalysisStore_MultipleFiles(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, "")
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
		err = store.RecordFileMetrics(analysisID, file, metrics)
		assert.NoError(t, err)

		scores := schema.FileScores{
			AnalysisTime:    time.Now(),
			HotScore:        75.5,
			RiskScore:       80.2,
			ComplexityScore: 65.3,
			StaleScore:      70.1,
			ScoreLabel:      "hot",
		}
		err = store.RecordFileScores(analysisID, file, scores)
		assert.NoError(t, err)
	}

	// End analysis
	err = store.EndAnalysis(analysisID, time.Now(), len(files))
	assert.NoError(t, err)
}

func TestAnalysisStore_MultipleRuns(t *testing.T) {
	store, err := NewAnalysisStore(schema.SQLiteBackend, "")
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
		err = store.RecordFileMetrics(id, "test.go", metrics)
		assert.NoError(t, err)

		scores := schema.FileScores{
			AnalysisTime:    time.Now(),
			HotScore:        75.5 + float64(i),
			RiskScore:       80.2 + float64(i),
			ComplexityScore: 65.3 + float64(i),
			StaleScore:      70.1 + float64(i),
			ScoreLabel:      "hot",
		}
		err = store.RecordFileScores(id, "test.go", scores)
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
	store, err := NewAnalysisStore(schema.SQLiteBackend, "")
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
