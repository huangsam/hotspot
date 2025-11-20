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

func TestAnalysisStore_GetStatus(t *testing.T) {
	t.Run("SQLite backend with data", func(t *testing.T) {
		store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
		require.NoError(t, err)
		require.NotNil(t, store)
		defer func() { _ = store.Close() }()

		// Create some analysis runs
		startTime := time.Now()
		runs := []map[string]any{
			{"mode": "hot", "lookback": "30d"},
			{"mode": "risk", "lookback": "60d"},
			{"mode": "complexity", "lookback": "90d"},
		}

		var analysisIDs []int64
		for _, config := range runs {
			id, err := store.BeginAnalysis(startTime, config)
			require.NoError(t, err)
			analysisIDs = append(analysisIDs, id)

			// Add some metrics
			metrics := schema.FileMetrics{
				AnalysisTime:     startTime,
				TotalCommits:     100,
				TotalChurn:       500,
				ContributorCount: 5,
				AgeDays:          365.0,
				GiniCoefficient:  0.5,
				FileOwner:        "owner",
			}
			scores := schema.FileScores{
				AnalysisTime:    startTime,
				HotScore:        75.5,
				RiskScore:       80.2,
				ComplexityScore: 65.3,
				StaleScore:      70.1,
				ScoreLabel:      "hot",
			}
			err = store.RecordFileMetricsAndScores(id, "test.go", metrics, scores)
			assert.NoError(t, err)

			err = store.EndAnalysis(id, startTime.Add(time.Minute), 1)
			assert.NoError(t, err)
		}

		// Get status
		status, err := store.GetStatus()
		assert.NoError(t, err, "GetStatus should not fail")

		assert.Len(t, analysisIDs, 3, "Should have collected 3 analysis IDs")
		assert.Equal(t, "sqlite", status.Backend, "Backend should be sqlite")
		assert.True(t, status.Connected, "Should be connected")
		assert.Equal(t, 3, status.TotalRuns, "Total runs should be 3")
		assert.Greater(t, status.LastRunID, int64(0), "Last run ID should be set")
		assert.True(t, !status.LastRunTime.IsZero(), "Last run time should not be zero")
		assert.True(t, status.OldestRunTime.Equal(startTime) || !status.OldestRunTime.IsZero(), "Oldest run time should be set")
		assert.Equal(t, 3, status.TotalFilesAnalyzed, "Total files analyzed should be 3")
		assert.Contains(t, status.TableSizes, "hotspot_analysis_runs", "Should have analysis_runs table size")
		assert.Contains(t, status.TableSizes, "hotspot_file_scores_metrics", "Should have file_scores_metrics table size")
	})

	t.Run("SQLite backend empty", func(t *testing.T) {
		store, err := NewAnalysisStore(schema.SQLiteBackend, ":memory:")
		require.NoError(t, err)
		require.NotNil(t, store)
		defer func() { _ = store.Close() }()

		// Get status without data
		status, err := store.GetStatus()
		assert.NoError(t, err, "GetStatus should not fail")

		assert.Equal(t, "sqlite", status.Backend, "Backend should be sqlite")
		assert.True(t, status.Connected, "Should be connected")
		assert.Equal(t, 0, status.TotalRuns, "Total runs should be 0")
		assert.Equal(t, int64(0), status.LastRunID, "Last run ID should be 0")
		assert.True(t, status.LastRunTime.IsZero(), "Last run time should be zero")
		assert.True(t, status.OldestRunTime.IsZero(), "Oldest run time should be zero")
		assert.Equal(t, 0, status.TotalFilesAnalyzed, "Total files analyzed should be 0")
		assert.Contains(t, status.TableSizes, "hotspot_analysis_runs", "Should have analysis_runs table size")
		assert.Contains(t, status.TableSizes, "hotspot_file_scores_metrics", "Should have file_scores_metrics table size")
	})

	t.Run("None backend", func(t *testing.T) {
		store, err := NewAnalysisStore(schema.NoneBackend, "")
		require.NoError(t, err)
		require.NotNil(t, store)

		// Get status
		status, err := store.GetStatus()
		assert.NoError(t, err, "GetStatus should not fail")

		assert.Equal(t, "none", status.Backend, "Backend should be none")
		assert.False(t, status.Connected, "Should not be connected")
		assert.Equal(t, 0, status.TotalRuns, "Total runs should be 0")
		assert.Equal(t, int64(0), status.LastRunID, "Last run ID should be 0")
		assert.True(t, status.LastRunTime.IsZero(), "Last run time should be zero")
		assert.True(t, status.OldestRunTime.IsZero(), "Oldest run time should be zero")
		assert.Equal(t, 0, status.TotalFilesAnalyzed, "Total files analyzed should be 0")
		assert.Empty(t, status.TableSizes, "Table sizes should be empty")
	})
}
