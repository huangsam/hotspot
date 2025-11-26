package algo

import (
	"maps"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

// getWeightsForMode returns the weight map for a given scoring mode.
// If custom weights are provided for the mode, they override the defaults.
// This is a test helper function.
func getWeightsForMode(mode schema.ScoringMode, customWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64) map[schema.BreakdownKey]float64 {
	// Start with default weights
	defaultWeights := schema.GetDefaultWeights(mode)

	// Override with custom weights if provided
	weights := make(map[schema.BreakdownKey]float64)
	maps.Copy(weights, defaultWeights)

	if customWeights != nil {
		if modeWeights, ok := customWeights[mode]; ok {
			maps.Copy(weights, modeWeights)
		}
	}

	return weights
}

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
			result := Gini(tt.values)
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
			weights := getWeightsForMode(mode, nil)
			score := ComputeScore(&metrics, mode, weights)
			assert.True(t, score >= 0 && score <= 100)
			assert.NotEmpty(t, metrics.ModeBreakdown)
		})
	}
}

// BenchmarkGini benchmarks the Gini coefficient calculation.
func BenchmarkGini(b *testing.B) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	for b.Loop() {
		Gini(values)
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

	weights := getWeightsForMode(schema.HotMode, nil)

	for b.Loop() {
		ComputeScore(&metrics, schema.HotMode, weights)
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
	defaultWeights := getWeightsForMode(schema.HotMode, nil)
	defaultScore := ComputeScore(&metrics, schema.HotMode, defaultWeights)

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
	customModeWeights := getWeightsForMode(schema.HotMode, customWeights)
	customScore := ComputeScore(&metrics, schema.HotMode, customModeWeights)

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
			defaultWeights := getWeightsForMode(mode, nil)
			defaultScore := ComputeScore(&metrics, mode, defaultWeights)

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

			customModeWeights := getWeightsForMode(mode, customWeights)
			customScore := ComputeScore(&metrics, mode, customModeWeights)

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
	defaultWeights := getWeightsForMode(schema.HotMode, nil)
	score := ComputeScore(&metrics, schema.HotMode, defaultWeights)
	assert.True(t, score >= 0 && score <= 100, "Score with nil custom weights should be valid")

	// Test with empty custom weights map (should use defaults)
	emptyWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{}
	emptyModeWeights := getWeightsForMode(schema.HotMode, emptyWeights)
	score = ComputeScore(&metrics, schema.HotMode, emptyModeWeights)
	assert.True(t, score >= 0 && score <= 100, "Score with empty custom weights should be valid")
}

// FuzzComputeScore fuzzes the ComputeScore function with random FileResult inputs.
func FuzzComputeScore(f *testing.F) {
	seeds := []struct {
		result schema.FileResult
		mode   schema.ScoringMode
	}{
		{
			result: schema.FileResult{
				Path:               "main.go",
				SizeBytes:          1000,
				LinesOfCode:        50,
				Commits:            10,
				Churn:              100,
				AgeDays:            365,
				UniqueContributors: 2,
				Gini:               0.5,
				RecentCommits:      5,
				Mode:               schema.HotMode,
			},
			mode: schema.HotMode,
		},
		{
			result: schema.FileResult{
				Path:               "test.go",
				SizeBytes:          1000,
				LinesOfCode:        100,
				Commits:            10,
				Churn:              50,
				AgeDays:            365,
				UniqueContributors: 2,
				Gini:               0.5,
				RecentCommits:      5,
				Mode:               schema.HotMode,
			},
			mode: schema.HotMode,
		},
		{
			result: schema.FileResult{
				Path:               "test.go",
				SizeBytes:          0, // edge case
				LinesOfCode:        0,
				Commits:            0,
				Churn:              0,
				AgeDays:            0,
				UniqueContributors: 0,
				Gini:               0,
				RecentCommits:      0,
				Mode:               schema.RiskMode,
			},
			mode: schema.RiskMode,
		},
	}
	for _, seed := range seeds {
		f.Add(seed.result.Path, seed.result.SizeBytes, seed.result.LinesOfCode,
			seed.result.Commits, seed.result.Churn, seed.result.AgeDays,
			seed.result.UniqueContributors, seed.result.Gini, seed.result.RecentCommits,
			string(seed.mode))
	}

	f.Fuzz(func(_ *testing.T,
		path string,
		sizeBytes int64,
		linesOfCode int,
		commits int,
		churn int,
		ageDays int,
		uniqueContributors int,
		gini float64,
		recentCommits int,
		mode string,
	) {
		result := schema.FileResult{
			Path:               path,
			SizeBytes:          sizeBytes,
			LinesOfCode:        linesOfCode,
			Commits:            commits,
			Churn:              churn,
			AgeDays:            ageDays,
			UniqueContributors: uniqueContributors,
			Gini:               gini,
			RecentCommits:      recentCommits,
			Mode:               schema.ScoringMode(mode),
		}
		_ = ComputeScore(&result, schema.ScoringMode(mode), getWeightsForMode(schema.ScoringMode(mode), nil))
	})
}

// FuzzGini fuzzes the Gini function with random value arrays.
func FuzzGini(f *testing.F) {
	seeds := []string{
		"[1,2,3]",
		"[0,0,0]",
		"[100]",
		"[]",
		"[1,1,1,1]",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, valuesJSON string) {
		// Simple parsing, may fail but that's ok for fuzzing
		var values []float64
		if valuesJSON != "" && valuesJSON[0] == '[' && valuesJSON[len(valuesJSON)-1] == ']' {
			// Very basic parsing, just for fuzzing
			inner := valuesJSON[1 : len(valuesJSON)-1]
			if inner != "" {
				parts := strings.SplitSeq(inner, ",")
				for p := range parts {
					// Skip parsing errors, just try
					if f, err := strconv.ParseFloat(strings.TrimSpace(p), 64); err == nil {
						values = append(values, f)
					}
				}
			}
		}
		_ = Gini(values)
	})
}
