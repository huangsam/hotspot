package core

import (
	"context"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestFileResultBuilder_BasicChaining(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath: "/test/repo",
		Mode:     schema.HotMode,
	}

	// Create a mock output with some basic data
	output := &schema.AggregateOutput{
		CommitMap: map[string]int{
			"test.go": 10,
		},
		ChurnMap: map[string]int{
			"test.go": 100,
		},
		ContribMap: map[string]map[string]int{
			"test.go": {
				"alice": 5,
				"bob":   5,
			},
		},
		FirstCommitMap: map[string]time.Time{
			"test.go": time.Now().Add(-30 * 24 * time.Hour),
		},
	}

	builder := NewFileMetricsBuilder(ctx, cfg, nil, "test.go", output)

	// Test that builder returns itself for chaining
	result := builder.FetchAllGitMetrics()
	assert.Equal(t, builder, result)

	result = builder.FetchFileStats()
	assert.Equal(t, builder, result)

	result = builder.CalculateDerivedMetrics()
	assert.Equal(t, builder, result)

	result = builder.FetchRecentInfo()
	assert.Equal(t, builder, result)

	result = builder.CalculateOwner()
	assert.Equal(t, builder, result)

	result = builder.CalculateScore()
	assert.Equal(t, builder, result)

	// Test final build
	fileResult := builder.Build()
	assert.Equal(t, "test.go", fileResult.Path)
	assert.Equal(t, schema.HotMode, fileResult.Mode)
	assert.Equal(t, 10, fileResult.Commits)
	assert.Equal(t, 100, fileResult.Churn)
	assert.Equal(t, 2, fileResult.UniqueContributors)
	assert.NotNil(t, fileResult.AllScores)
	assert.Contains(t, fileResult.AllScores, schema.HotMode)
	assert.Contains(t, fileResult.AllScores, schema.RiskMode)
	assert.Contains(t, fileResult.AllScores, schema.ComplexityMode)
	assert.Contains(t, fileResult.AllScores, schema.StaleMode)
}

func TestFileResultBuilder_EmptyContribMap(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath: "/test/repo",
		Mode:     schema.HotMode,
	}

	output := &schema.AggregateOutput{
		ContribMap: map[string]map[string]int{}, // Empty contrib map
	}

	builder := NewFileMetricsBuilder(ctx, cfg, nil, "test.go", output)

	builder.FetchAllGitMetrics().CalculateOwner()

	fileResult := builder.Build()
	assert.Empty(t, fileResult.Owners)
	assert.Equal(t, 0, fileResult.UniqueContributors)
}

func TestFileResultBuilder_ZeroFirstCommit(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath: "/test/repo",
		Mode:     schema.HotMode,
	}

	output := &schema.AggregateOutput{
		FirstCommitMap: map[string]time.Time{}, // No first commit
	}

	builder := NewFileMetricsBuilder(ctx, cfg, nil, "test.go", output)

	builder.FetchAllGitMetrics().CalculateDerivedMetrics()

	fileResult := builder.Build()
	assert.Equal(t, 0, fileResult.AgeDays)
}
