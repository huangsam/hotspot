package core

import (
	"math"
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

// TestGini tests the Gini coefficient calculation
func TestGini(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
		delta    float64
	}{
		{
			name:     "empty slice",
			values:   []float64{},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "perfect equality",
			values:   []float64{1, 1, 1, 1},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "perfect inequality",
			values:   []float64{0, 0, 0, 10},
			expected: 0.75,
			delta:    0.001,
		},
		{
			name:     "moderate inequality",
			values:   []float64{1, 2, 3, 4},
			expected: 0.25,
			delta:    0.001,
		},
		{
			name:     "single value",
			values:   []float64{5},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "all zeros",
			values:   []float64{0, 0, 0},
			expected: 0.0,
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gini(tt.values)
			assert.LessOrEqual(t, math.Abs(result-tt.expected), tt.delta)
		})
	}
}

// TestComputeScoreHotMode tests the default hot mode scoring
func TestComputeScoreHotMode(t *testing.T) {
	tests := []struct {
		name     string
		metrics  schema.FileMetrics
		minScore float64
		maxScore float64
	}{
		{
			name: "zero metrics",
			metrics: schema.FileMetrics{
				Path:               "test.go",
				UniqueContributors: 0,
				Commits:            0,
				SizeBytes:          1,
				AgeDays:            0,
				Churn:              0,
				Gini:               0,
			},
			minScore: 0,
			maxScore: 5,
		},
		{
			name: "high activity file",
			metrics: schema.FileMetrics{
				Path:               "active.go",
				UniqueContributors: 10,
				Commits:            100,
				SizeBytes:          50 * 1024, // 50KB
				AgeDays:            365,
				Churn:              500,
				Gini:               0.3,
			},
			minScore: 25,
			maxScore: 35,
		},
		{
			name: "saturated metrics",
			metrics: schema.FileMetrics{
				Path:               "huge.go",
				UniqueContributors: 30,   // beyond maxContrib
				Commits:            1000, // beyond maxCommits
				SizeBytes:          1024 * 1024,
				AgeDays:            5000,
				Churn:              10000,
				Gini:               0.1,
			},
			minScore: 70,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeScore(&tt.metrics, "hot")
			assert.True(t, score >= tt.minScore && score <= tt.maxScore)
			// Verify score is in valid range
			assert.True(t, score >= 0 && score <= 100)
			// Verify breakdown was populated
			assert.NotEmpty(t, tt.metrics.Breakdown)
		})
	}
}

func TestComputeScoreHotMode_EmptyFile(t *testing.T) {
	metrics := schema.FileMetrics{
		Path:      "active.go",
		SizeBytes: 0,
	}
	score := computeScore(&metrics, "hot")
	assert.Empty(t, score, "Should not create score for empty file")
	assert.Empty(t, metrics.Breakdown, "Should not create breakdown for empty file")
}

// TestComputeScoreRiskMode tests risk mode scoring
func TestComputeScoreRiskMode(t *testing.T) {
	tests := []struct {
		name     string
		metrics  schema.FileMetrics
		minScore float64
		maxScore float64
	}{
		{
			name: "low risk - many contributors, low gini",
			metrics: schema.FileMetrics{
				Path:               "safe.go",
				UniqueContributors: 15,
				Commits:            50,
				SizeBytes:          20 * 1024,
				AgeDays:            100,
				Churn:              100,
				Gini:               0.1, // low inequality
			},
			minScore: 0,
			maxScore: 30,
		},
		{
			name: "high risk - few contributors, high gini",
			metrics: schema.FileMetrics{
				Path:               "risky.go",
				UniqueContributors: 2,
				Commits:            100,
				SizeBytes:          100 * 1024,
				AgeDays:            1000,
				Churn:              500,
				Gini:               0.8, // high inequality
			},
			minScore: 40,
			maxScore: 80,
		},
		{
			name: "test file should get reduced score",
			metrics: schema.FileMetrics{
				Path:               "controller_test.go",
				UniqueContributors: 1,
				Commits:            50,
				SizeBytes:          30 * 1024,
				AgeDays:            500,
				Churn:              200,
				Gini:               0.9,
			},
			minScore: 0,
			maxScore: 60, // should be reduced by 0.75 multiplier
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeScore(&tt.metrics, "risk")
			assert.True(t, score >= tt.minScore && score <= tt.maxScore)
			assert.True(t, score >= 0 && score <= 100)
		})
	}
}

// TestComputeScoreAllModes ensures all modes produce valid scores
func TestComputeScoreAllModes(t *testing.T) {
	modes := []string{"hot", "risk", "complexity", "stale"}

	metrics := schema.FileMetrics{
		Path:               "test.go",
		UniqueContributors: 5,
		Commits:            50,
		RecentCommits:      10,
		SizeBytes:          50 * 1024,
		AgeDays:            365,
		Churn:              250,
		Gini:               0.3,
		FirstCommit:        time.Now().AddDate(0, 0, -365),
	}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			score := computeScore(&metrics, mode)
			assert.True(t, score >= 0 && score <= 100)
			assert.NotEmpty(t, metrics.Breakdown)
		})
	}
}

// TestRankFiles tests file ranking logic
func TestRankFiles(t *testing.T) {
	files := []schema.FileMetrics{
		{Path: "low.go", SizeBytes: 1, Score: 10},
		{Path: "high.go", SizeBytes: 1, Score: 90},
		{Path: "medium.go", SizeBytes: 1, Score: 50},
		{Path: "critical.go", SizeBytes: 1, Score: 95},
	}

	t.Run("rank and limit", func(t *testing.T) {
		ranked := RankFiles(files, 2)
		assert.Equal(t, 2, len(ranked))
		assert.Equal(t, "critical.go", ranked[0].Path)
		assert.Equal(t, "high.go", ranked[1].Path)
	})

	t.Run("limit exceeds length", func(t *testing.T) {
		ranked := RankFiles(files, 10)
		assert.Equal(t, 4, len(ranked))
	})

	t.Run("scores in descending order", func(t *testing.T) {
		ranked := RankFiles(files, 10)
		for i := 1; i < len(ranked); i++ {
			assert.LessOrEqual(t, ranked[i].Score, ranked[i-1].Score)
		}
	})
}

// BenchmarkGini benchmarks the Gini coefficient calculation
func BenchmarkGini(b *testing.B) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gini(values)
	}
}

// BenchmarkComputeScore benchmarks score calculation
func BenchmarkComputeScore(b *testing.B) {
	metrics := schema.FileMetrics{
		Path:               "test.go",
		UniqueContributors: 5,
		Commits:            50,
		SizeBytes:          50 * 1024,
		AgeDays:            365,
		Churn:              250,
		Gini:               0.3,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		computeScore(&metrics, "hot")
	}
}
