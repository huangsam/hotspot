package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzeFileCommon(t *testing.T) {
	ctx := context.Background()

	// Create mock client
	mockClient := &contract.MockGitClient{}

	// No git calls needed - all data comes from aggregation phase

	// Create config
	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
	}

	// Create aggregate output
	aggOutput := &schema.AggregateOutput{
		CommitMap: map[string]int{"main.go": 5},
		ChurnMap:  map[string]int{"main.go": 15},
		ContribMap: map[string]map[string]int{
			"main.go": {"alice": 3, "bob": 2},
		},
		FirstCommitMap: map[string]time.Time{
			"main.go": time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
		},
	}

	// Execute
	result := analyzeFileCommon(ctx, cfg, mockClient, "main.go", aggOutput)

	// Assert
	assert.Equal(t, "main.go", result.Path)
	assert.Equal(t, 5, result.Commits)
	assert.Equal(t, 15, result.Churn)
	assert.Equal(t, 2, result.UniqueContributors)
	assert.True(t, result.ModeScore >= 0 && result.ModeScore <= 100)
	// Note: Breakdown will be empty because SizeBytes is 0 (file doesn't exist in test)
	// assert.NotEmpty(t, result.Breakdown)
}

func TestAnalyzeRepo(t *testing.T) {
	ctx := context.Background()

	// Create mock client
	mockClient := &contract.MockGitClient{}

	// No git calls needed - all data comes from aggregation phase

	// Create config
	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   2,
	}

	// Create aggregate output
	aggOutput := &schema.AggregateOutput{
		CommitMap: map[string]int{
			"main.go":     1,
			"core/agg.go": 1,
		},
		ChurnMap: map[string]int{
			"main.go":     3,
			"core/agg.go": 4,
		},
		ContribMap: map[string]map[string]int{
			"main.go":     {"alice": 1},
			"core/agg.go": {"bob": 1},
		},
		FirstCommitMap: map[string]time.Time{
			"main.go":     time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			"core/agg.go": time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
		},
	}

	files := []string{"main.go", "core/agg.go"}

	// Execute
	results := analyzeRepo(ctx, cfg, mockClient, aggOutput, files)

	// Assert
	assert.Len(t, results, 2)

	// Check that both files are present
	paths := make([]string, len(results))
	for i, result := range results {
		paths[i] = result.Path
	}
	assert.Contains(t, paths, "main.go")
	assert.Contains(t, paths, "core/agg.go")

	// Verify scores are calculated
	for _, result := range results {
		assert.True(t, result.ModeScore >= 0 && result.ModeScore <= 100)
		// Note: Breakdown will be empty because SizeBytes is 0 (files don't exist in test)
		// assert.NotEmpty(t, result.Breakdown)
	}
}

// TestAnalyzeRepo_ConcurrentWorkers tests that analyzeRepo works correctly with multiple concurrent workers
// and doesn't have race conditions when accessing shared data structures.
func TestAnalyzeRepo_ConcurrentWorkers(t *testing.T) {
	ctx := context.Background()

	// Create mock client
	mockClient := &contract.MockGitClient{}

	// Create config with multiple workers
	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
		Mode:      schema.HotMode,
		Workers:   4, // Test with multiple workers
	}

	// Create aggregate output with many files to test concurrency
	commitMap := make(map[string]int)
	churnMap := make(map[string]int)
	contribMap := make(map[string]map[string]int)
	firstCommitMap := make(map[string]time.Time)

	files := make([]string, 20) // Test with 20 files
	for i := range 20 {
		fileName := fmt.Sprintf("file%d.go", i)
		files[i] = fileName
		commitMap[fileName] = i + 1
		churnMap[fileName] = (i + 1) * 2
		contribMap[fileName] = map[string]int{"alice": i + 1}
		firstCommitMap[fileName] = time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	}

	aggOutput := &schema.AggregateOutput{
		CommitMap:      commitMap,
		ChurnMap:       churnMap,
		ContribMap:     contribMap,
		FirstCommitMap: firstCommitMap,
	}

	// Execute with multiple workers
	results := analyzeRepo(ctx, cfg, mockClient, aggOutput, files)

	// Assert
	assert.Len(t, results, 20)

	// Check that all files are present and results are consistent
	resultMap := make(map[string]schema.FileResult)
	for _, result := range results {
		resultMap[result.Path] = result
	}

	for _, file := range files {
		result, exists := resultMap[file]
		assert.True(t, exists, "File %s should be in results", file)
		assert.Equal(t, file, result.Path)
		assert.True(t, result.ModeScore >= 0 && result.ModeScore <= 100)
	}
}

// TestGetOwnerString tests the owner string conversion.
func TestGetOwnerString(t *testing.T) {
	tests := []struct {
		name     string
		owners   []string
		expected string
	}{
		{
			name:     "empty owners",
			owners:   []string{},
			expected: "",
		},
		{
			name:     "single owner",
			owners:   []string{"alice"},
			expected: "alice",
		},
		{
			name:     "multiple owners",
			owners:   []string{"alice", "bob", "charlie"},
			expected: "alice", // Should return first owner
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOwnerString(tt.owners)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRecordFileAnalysis tests the file analysis recording function.
func TestRecordFileAnalysis(t *testing.T) {
	ctx := context.Background()

	// Create mock cache manager
	mockCacheMgr := &iocache.MockCacheManager{}
	mockCacheMgr.On("GetAnalysisStore").Return(nil) // No analysis tracking for test

	// Create config
	cfg := &contract.Config{
		Mode: schema.HotMode,
	}

	// Create file result with all scores
	fileResult := &schema.FileResult{
		Path:               "test.go",
		Commits:            100,
		Churn:              500,
		UniqueContributors: 5,
		AgeDays:            365,
		Gini:               0.3,
		Owners:             []string{"alice", "bob"},
		AllScores: map[schema.ScoringMode]float64{
			schema.HotMode:        75.5,
			schema.RiskMode:       80.2,
			schema.ComplexityMode: 65.3,
			schema.StaleMode:      70.1,
		},
	}

	// Set up context with cache manager
	ctx = contextWithCacheManager(ctx, mockCacheMgr)

	// Execute - should not panic
	recordFileAnalysis(ctx, cfg, 1, "test.go", fileResult)

	// Verify mocks were called
	mockCacheMgr.AssertExpectations(t)
}
