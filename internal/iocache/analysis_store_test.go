package iocache

import (
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
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

	err = store.RecordFileMetrics(1, "test.go", contract.FileMetrics{})
	assert.NoError(t, err)

	err = store.RecordFileScores(1, "test.go", contract.FileScores{})
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
	metrics := contract.FileMetrics{
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
	scores := contract.FileScores{
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
		metrics := contract.FileMetrics{
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

		scores := contract.FileScores{
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
		metrics := contract.FileMetrics{
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

		scores := contract.FileScores{
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
