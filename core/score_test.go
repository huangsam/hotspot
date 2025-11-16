package core

import (
	"math"
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

// TestGini tests the Gini coefficient calculation.
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

// TestComputeScoreAllModes ensures all modes produce valid scores.
func TestComputeScoreAllModes(t *testing.T) {
	modes := []schema.ScoringMode{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.StaleMode}

	metrics := schema.FileResult{
		Path:               "test.go",
		UniqueContributors: 5,
		Commits:            50,
		RecentCommits:      10,
		SizeBytes:          50 * 1024,
		AgeDays:            365,
		Churn:              250,
		Gini:               0.3,
		FirstCommit:        time.Now().AddDate(0, 0, -365),
		Mode:               schema.HotMode,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			score := computeScore(&metrics, mode, nil)
			assert.True(t, score >= 0 && score <= 100)
			assert.NotEmpty(t, metrics.Breakdown)
		})
	}
}

// TestComputeFolderScore validates folder computation.
func TestComputeFolderScore(t *testing.T) {
	t.Run("divide by zero", func(t *testing.T) {
		results := &schema.FolderResult{
			Path:             ".",
			TotalLOC:         0,
			WeightedScoreSum: 100.0,
		}
		score := computeFolderScore(results)
		assert.Empty(t, score)
	})

	t.Run("valid calculation", func(t *testing.T) {
		results := &schema.FolderResult{
			Path:             ".",
			TotalLOC:         100,
			WeightedScoreSum: 92.0,
		}
		score := computeFolderScore(results)
		assert.InEpsilon(t, float64(.92), score, 0.01)
	})
}

// BenchmarkGini benchmarks the Gini coefficient calculation.
func BenchmarkGini(b *testing.B) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	for b.Loop() {
		gini(values)
	}
}

// BenchmarkComputeScore benchmarks score calculation.
func BenchmarkComputeScore(b *testing.B) {
	metrics := schema.FileResult{
		Path:               "test.go",
		UniqueContributors: 5,
		Commits:            50,
		SizeBytes:          50 * 1024,
		AgeDays:            365,
		Churn:              250,
		Gini:               0.3,
		Mode:               schema.HotMode,
	}

	for b.Loop() {
		computeScore(&metrics, schema.HotMode, nil)
	}
}

// TestComputeScoreWithCustomWeights tests that custom weights produce different results than defaults.
func TestComputeScoreWithCustomWeights(t *testing.T) {
	metrics := schema.FileResult{
		Path:               "test.go",
		UniqueContributors: 5,
		Commits:            50,
		SizeBytes:          50 * 1024,
		AgeDays:            365,
		Churn:              250,
		Gini:               0.3,
		Mode:               schema.HotMode,
	}

	// Get default score
	defaultScore := computeScore(&metrics, schema.HotMode, nil)

	// Test with custom weights that heavily weight commits
	customWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.HotMode: {
			schema.BreakdownCommits: 0.8, // Much higher weight on commits
			schema.BreakdownChurn:   0.1,
			schema.BreakdownAge:     0.05,
			schema.BreakdownContrib: 0.03,
			schema.BreakdownSize:    0.02,
		},
	}
	customScore := computeScore(&metrics, schema.HotMode, customWeights)

	// Scores should be different
	assert.NotEqual(t, defaultScore, customScore, "Custom weights should produce different score than defaults")

	// Both should be valid scores
	assert.True(t, defaultScore >= 0 && defaultScore <= 100, "Default score should be valid")
	assert.True(t, customScore >= 0 && customScore <= 100, "Custom score should be valid")
}

// TestComputeScoreCustomWeightsAllModes tests custom weights for all scoring modes.
func TestComputeScoreCustomWeightsAllModes(t *testing.T) {
	modes := []schema.ScoringMode{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.StaleMode}

	metrics := schema.FileResult{
		Path:               "test.go",
		UniqueContributors: 5,
		Commits:            50,
		RecentCommits:      10,
		SizeBytes:          50 * 1024,
		AgeDays:            365,
		Churn:              250,
		Gini:               0.3,
		LinesOfCode:        500,
		Mode:               schema.HotMode,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			// Get default score
			defaultScore := computeScore(&metrics, mode, nil)

			// Create custom weights that emphasize different aspects
			customWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				mode: {},
			}

			// Add some custom weights based on mode
			switch mode {
			case schema.HotMode:
				customWeights[mode][schema.BreakdownCommits] = 0.6
				customWeights[mode][schema.BreakdownChurn] = 0.4
			case schema.RiskMode:
				customWeights[mode][schema.BreakdownInvContrib] = 0.5
				customWeights[mode][schema.BreakdownGini] = 0.5
			case schema.StaleMode:
				customWeights[mode][schema.BreakdownInvRecent] = 0.5
				customWeights[mode][schema.BreakdownAge] = 0.5
			case schema.ComplexityMode:
				customWeights[mode][schema.BreakdownSize] = 0.4
				customWeights[mode][schema.BreakdownLOC] = 0.4
				customWeights[mode][schema.BreakdownAge] = 0.2
			}

			customScore := computeScore(&metrics, mode, customWeights)

			// Both should be valid scores
			assert.True(t, defaultScore >= 0 && defaultScore <= 100, "Default score should be valid")
			assert.True(t, customScore >= 0 && customScore <= 100, "Custom score should be valid")

			// For modes where we changed weights significantly, scores should differ
			if mode == schema.HotMode {
				assert.NotEqual(t, defaultScore, customScore, "Hot mode custom weights should produce different score")
			}
		})
	}
}

// TestComputeScoreInvalidCustomWeights tests behavior with invalid custom weights.
func TestComputeScoreInvalidCustomWeights(t *testing.T) {
	metrics := schema.FileResult{
		Path:               "test.go",
		UniqueContributors: 5,
		Commits:            50,
		SizeBytes:          50 * 1024,
		AgeDays:            365,
		Churn:              250,
		Gini:               0.3,
		Mode:               schema.HotMode,
	}

	// Test with nil custom weights (should use defaults)
	score := computeScore(&metrics, schema.HotMode, nil)
	assert.True(t, score >= 0 && score <= 100, "Score with nil custom weights should be valid")

	// Test with empty custom weights map (should use defaults)
	emptyWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{}
	score = computeScore(&metrics, schema.HotMode, emptyWeights)
	assert.True(t, score >= 0 && score <= 100, "Score with empty custom weights should be valid")
}
