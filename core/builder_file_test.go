package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestFileResultBuilder_BasicChaining(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
	}

	// Create a mock output with some basic data
	output := &schema.AggregateOutput{
		FileStats: map[string]*schema.FileAggregation{
			"test.go": {
				Commits:     10,
				Churn:       100,
				FirstCommit: time.Now().Add(-30 * 24 * time.Hour),
				Contributors: map[string]schema.Metric{
					"alice": 5,
					"bob":   5,
				},
			},
		},
	}

	builder := NewFileMetricsBuilder(ctx, cfg.Git, cfg.Scoring, nil, "test.go", output)

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
	assert.Equal(t, schema.Metric(10), fileResult.Commits)
	assert.Equal(t, schema.Metric(100), fileResult.Churn)
	assert.Equal(t, schema.Metric(2), fileResult.UniqueContributors)
	assert.NotNil(t, fileResult.AllScores)
	assert.Contains(t, fileResult.AllScores, schema.HotMode)
	assert.Contains(t, fileResult.AllScores, schema.RiskMode)
	assert.Contains(t, fileResult.AllScores, schema.ComplexityMode)
}

func TestFileResultBuilder_EmptyContribMap(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
	}

	output := &schema.AggregateOutput{
		FileStats: map[string]*schema.FileAggregation{}, // Empty stats
	}

	builder := NewFileMetricsBuilder(ctx, cfg.Git, cfg.Scoring, nil, "test.go", output)

	builder.FetchAllGitMetrics().CalculateOwner()

	fileResult := builder.Build()
	assert.Empty(t, fileResult.Owners)
	assert.Equal(t, schema.Metric(0), fileResult.UniqueContributors)
}

func TestFileResultBuilder_ZeroFirstCommit(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
		Scoring: config.ScoringConfig{
			Mode: schema.HotMode,
		},
	}

	output := &schema.AggregateOutput{
		FileStats: map[string]*schema.FileAggregation{}, // No stats
	}

	builder := NewFileMetricsBuilder(ctx, cfg.Git, cfg.Scoring, nil, "test.go", output)

	builder.FetchAllGitMetrics().CalculateDerivedMetrics()

	fileResult := builder.Build()
	assert.Equal(t, schema.Metric(0), fileResult.AgeDays)
}

func TestFileResultBuilder_CompositeModeUsesSelectedBreakdown(t *testing.T) {
	ctx := context.Background()
	const (
		aliceCommits = schema.Metric(100)
		bobCommits   = schema.Metric(15)
		carolCommits = schema.Metric(5)
	)

	repoPath := t.TempDir()
	err := os.WriteFile(filepath.Join(repoPath, "critical.go"), []byte("package main\n\nfunc x() {}\n"), 0o644)
	assert.NoError(t, err)

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: repoPath,
		},
		Scoring: config.ScoringConfig{
			Mode:                 schema.ActiveOwnersMode,
			RecencyThresholdLow:  0.1,
			RecencyThresholdHigh: 0.3,
		},
	}

	output := &schema.AggregateOutput{
		FileStats: map[string]*schema.FileAggregation{
			"critical.go": {
				Commits:       120,
				RecentCommits: 45,
				Churn:         3500,
				RecentChurn:   1600,
				FirstCommit:   time.Now().Add(-540 * 24 * time.Hour),
				Contributors: map[string]schema.Metric{
					"alice": aliceCommits,
					"bob":   bobCommits,
					"carol": carolCommits,
				},
			},
		},
	}

	result := NewFileMetricsBuilder(ctx, cfg.Git, cfg.Scoring, nil, "critical.go", output).
		FetchAllGitMetrics().
		FetchFileStats().
		CalculateDerivedMetrics().
		FetchRecentInfo().
		CalculateOwner().
		CalculateScore().
		Build()

	assert.Equal(t, schema.ActiveOwnersMode, result.Mode)
	assert.Equal(t, "composite", result.ModeType)
	assert.Equal(t, result.AllScores[schema.ActiveOwnersMode], result.ModeScore)
	assert.Equal(t, result.AllBreakdowns[schema.ActiveOwnersMode], result.ModeBreakdown)
	assert.NotEmpty(t, result.ModeBreakdown)
	assert.Contains(t, result.AllBreakdowns, schema.LegacyDebtMode)
	assert.Greater(t, result.RecencySignal, 0.0)
	assert.Equal(t, 0.1, result.RecencyThresholdLow)
	assert.Equal(t, 0.3, result.RecencyThresholdHigh)
}
