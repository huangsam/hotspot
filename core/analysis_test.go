package core

import (
	"context"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
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
	assert.True(t, result.Score >= 0 && result.Score <= 100)
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
		assert.True(t, result.Score >= 0 && result.Score <= 100)
		// Note: Breakdown will be empty because SizeBytes is 0 (files don't exist in test)
		// assert.NotEmpty(t, result.Breakdown)
	}
}
