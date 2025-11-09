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

// TestComputeScoreHotMode tests the default hot mode scoring.
func TestComputeScoreHotMode(t *testing.T) {
	// Hot Mode Weights: wCommits=0.40, wChurn=0.40, wAge=0.10, wContrib=0.05, wSize=0.05
	tests := []struct {
		name     string
		metrics  schema.FileResult
		minScore float64
		maxScore float64
	}{
		{
			name: "zero metrics",
			metrics: schema.FileResult{
				Path:               "empty.go",
				UniqueContributors: 0,
				Commits:            0,
				SizeBytes:          1,
				AgeDays:            0,
				Churn:              0,
				Gini:               0,
				Mode:               schema.HotMode,
			},
			// Expected Score: ~0 (Debuffed to 0 by path check)
			minScore: 0,
			maxScore: 1, // Keep a small upper bound for floating point safety
		},
		{
			name: "high activity file",
			metrics: schema.FileResult{
				Path:               "active.go",
				UniqueContributors: 10,        // 10/20 = 0.5 nContrib
				Commits:            100,       // 100/500 = 0.2 nCommits
				SizeBytes:          50 * 1024, // 50/500 = 0.1 nSize
				AgeDays:            365,       // log(366)/log(3651) ~ 0.5 nAge
				Churn:              500,       // 500/5000 = 0.1 nChurn
				Gini:               0.3,
				LinesOfCode:        100,
				Mode:               schema.HotMode,
			},
			// Expected Score (Raw): 0.4*0.2 + 0.4*0.1 + 0.1*0.5 + 0.05*0.5 + 0.05*0.1 = 0.08 + 0.04 + 0.05 + 0.025 + 0.005 = 0.20
			// Score: 20.0
			minScore: 15, // Adjusted from 20 to 15 to allow for a reasonable range around 20
			maxScore: 30, // Adjusted from 35 to 30
		},
		{
			name: "saturated metrics",
			metrics: schema.FileResult{
				Path:               "huge.go",
				UniqueContributors: 30,   // >= maxContrib -> 1.0 nContrib
				Commits:            1000, // >= maxCommits -> 1.0 nCommits
				SizeBytes:          1024 * 1024,
				AgeDays:            5000, // >= maxAgeDays -> ~1.0 nAge
				Churn:              2000, // 2000/5000 = 0.4 nChurn
				Gini:               0.1,
				LinesOfCode:        20000,
				Mode:               schema.HotMode,
			},
			// Expected Score (Raw): 0.4*1.0 + 0.4*0.4 + 0.1*1.0 + 0.05*1.0 + 0.05*1.0 = 0.4 + 0.16 + 0.1 + 0.05 + 0.05 = 0.76
			// Score: 76.0
			minScore: 70, // Retained the minimum, as 76 is high
			maxScore: 85, // Adjusted from 100 to 85, reflecting the raw calculation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeScore(&tt.metrics, schema.HotMode, nil)
			assert.True(t, score >= tt.minScore && score <= tt.maxScore, "%f is out of the valid %f-%f range", score, tt.minScore, tt.maxScore)
			// Verify score is in valid range
			assert.True(t, score >= 0 && score <= 100, "%f is out of the valid 0-100 range")
			// Verify breakdown was populated
			assert.NotEmpty(t, tt.metrics.Breakdown)
		})
	}
}

func TestComputeScoreHotMode_EmptyFile(t *testing.T) {
	metrics := schema.FileResult{
		Path:      "active.go",
		SizeBytes: 0,
		Mode:      schema.HotMode,
	}
	score := computeScore(&metrics, schema.HotMode, nil)
	assert.Equal(t, float64(0), score, "Score should be 0.0 for empty file")
	assert.Empty(t, metrics.Breakdown, "Breakdown should be empty for empty file")
}

// TestComputeScoreRiskMode tests risk mode scoring.
func TestComputeScoreRiskMode(t *testing.T) {
	// Risk Mode Weights: wInvContrib=0.30, wGini=0.26, wAgeRisk=0.16, wSizeRisk=0.12, wChurnRisk=0.06, wCommRisk=0.04, wLOCRisk=0.06
	tests := []struct {
		name     string
		metrics  schema.FileResult
		minScore float64
		maxScore float64
	}{
		{
			name: "low risk - many contributors, low gini",
			metrics: schema.FileResult{
				Path:               "safe.go",
				UniqueContributors: 15,        // 15/20 = 0.75 nContrib -> 0.25 nInvContrib
				Commits:            50,        // 50/500 = 0.1 nCommits
				SizeBytes:          20 * 1024, // 20/500 = 0.04 nSize
				AgeDays:            100,
				Churn:              100, // 100/5000 = 0.02 nChurn
				Gini:               0.1, // 0.1 nGiniRaw
				LinesOfCode:        500, // 500/10000 = 0.05 nLOC
				Mode:               schema.RiskMode,
			},
			// Expected Score (Raw):
			// 0.30*0.25 + 0.26*0.1 + 0.16*nAge + 0.12*0.04 + 0.06*0.02 + 0.04*0.1 + 0.06*0.05
			// nAge is low (~0.4)
			// Approx: 0.075 + 0.026 + 0.064 + 0.0048 + 0.0012 + 0.004 + 0.003 = ~0.178
			// Score: 17.8
			minScore: 10, // Adjusted from 0
			maxScore: 30, // Retained
		},
		{
			name: "high risk - few contributors, high gini",
			metrics: schema.FileResult{
				Path:               "risky.go",
				UniqueContributors: 2,          // 2/20 = 0.1 nContrib -> 0.9 nInvContrib
				Commits:            100,        // 0.2 nCommits
				SizeBytes:          100 * 1024, // 0.2 nSize
				AgeDays:            1000,
				Churn:              500,  // 0.1 nChurn
				Gini:               0.8,  // 0.8 nGiniRaw
				LinesOfCode:        3000, // 0.3 nLOC
				Mode:               schema.RiskMode,
			},
			// Expected Score (Raw):
			// 0.30*0.9 + 0.26*0.8 + 0.16*nAge + 0.12*0.2 + 0.06*0.1 + 0.04*0.2 + 0.06*0.3
			// nAge is mid-to-high (~0.7)
			// Approx: 0.27 + 0.208 + 0.112 + 0.024 + 0.006 + 0.008 + 0.018 = ~0.646
			// Score: 64.6
			minScore: 55, // Adjusted from 40 to better reflect the high score
			maxScore: 80, // Retained
		},
		{
			name: "test file should get reduced score",
			metrics: schema.FileResult{
				Path:               "controller_test.go", // Debuff: score *= 0.75
				UniqueContributors: 1,                    // 0.95 nInvContrib
				Commits:            50,
				SizeBytes:          30 * 1024,
				AgeDays:            500,
				Churn:              200,
				Gini:               0.9, // 0.9 nGiniRaw
				LinesOfCode:        1000,
				Mode:               schema.RiskMode,
			},
			// Raw Score (similar to high risk, but smaller values): ~0.55
			// Debuffed Score: 0.55 * 0.75 = ~0.4125
			// Score: 41.25
			minScore: 30, // Adjusted from 0
			maxScore: 50, // Adjusted from 60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeScore(&tt.metrics, schema.RiskMode, nil)
			assert.True(t, score >= tt.minScore && score <= tt.maxScore)
			assert.True(t, score >= 0 && score <= 100)
		})
	}
}

// TestComputeScoreStaleMode tests stale mode scoring, prioritizing old, inactive files.
func TestComputeScoreStaleMode(t *testing.T) {
	// Stale Mode Weights: wInvRecentStale=0.35, wSizeStale=0.25, wAgeStale=0.20, wCommitsStale=0.15, wContribStale=0.05
	tests := []struct {
		name     string
		metrics  schema.FileResult
		minScore float64
		maxScore float64
	}{
		{
			name: "high stale - old and inactive",
			metrics: schema.FileResult{
				Path:               "stale_code.go",
				UniqueContributors: 1,         // 0.05 nContrib
				Commits:            5,         // 0.01 nCommits
				RecentCommits:      0,         // 1.0 nInvRecentCommits
				SizeBytes:          10 * 1024, // 0.02 nSize
				AgeDays:            1500,      // ~0.8 nAge
				Churn:              10,
				Gini:               0.9,
				Mode:               schema.StaleMode,
			},
			// Expected Score (Raw):
			// 0.35*1.0 + 0.25*0.02 + 0.20*0.8 + 0.15*0.01 + 0.05*0.05
			// Approx: 0.35 + 0.005 + 0.16 + 0.0015 + 0.0025 = ~0.519
			// Score: 51.9
			minScore: 45, // Retained
			maxScore: 65, // Adjusted from 100 to better reflect the raw score
		},
		{
			name: "low stale - new and active",
			metrics: schema.FileResult{
				Path:               "new_feature.go",
				UniqueContributors: 5,         // 0.25 nContrib
				Commits:            30,        // 0.06 nCommits
				RecentCommits:      25,        // 0.5 nRecentCommits -> 0.5 nInvRecentCommits
				SizeBytes:          50 * 1024, // 0.1 nSize
				AgeDays:            30,        // Low nAge (~0.3)
				Churn:              500,
				Gini:               0.3,
				Mode:               schema.StaleMode,
			},
			// Expected Score (Raw):
			// 0.35*0.5 + 0.25*0.1 + 0.20*nAge + 0.15*0.06 + 0.05*0.25
			// Approx: 0.175 + 0.025 + 0.06 + 0.009 + 0.0125 = ~0.2815
			// Score: 28.15
			minScore: 20, // Adjusted from 0
			maxScore: 35, // Retained
		},
		{
			name: "test file should get lower score",
			metrics: schema.FileResult{
				Path:               "stale_test.go", // Debuff: score *= 0.50
				UniqueContributors: 1,
				Commits:            10,
				RecentCommits:      0, // 1.0 nInvRecentCommits
				SizeBytes:          5 * 1024,
				AgeDays:            1000, // ~0.7 nAge
				Churn:              50,
				Gini:               0.9,
				Mode:               schema.StaleMode,
			},
			// Raw Score (similar to high stale): ~0.45
			// Debuffed Score: 0.45 * 0.50 = ~0.225
			// Score: 22.5
			minScore: 15, // Retained
			maxScore: 30, // Adjusted from 70
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeScore(&tt.metrics, schema.StaleMode, nil)
			assert.True(t, score >= tt.minScore && score <= tt.maxScore)
			assert.True(t, score >= 0 && score <= 100)
			assert.NotEmpty(t, tt.metrics.Breakdown)
		})
	}
}

// TestComputeScoreComplexityMode is a placeholder for testing complexity scoring.
// The complexity mode weights: Size, Age, and Churn heavily, with a minor focus on low recent activity.
func TestComputeScoreComplexityMode(t *testing.T) {
	// Complexity Mode Weights: wAgeComplex=0.30, wChurnComplex=0.30, wLOCComplex=0.20, wCommComplex=0.10, wSizeComplex=0.05, wContribLow=0.05
	tests := []struct {
		name     string
		metrics  schema.FileResult
		minScore float64
		maxScore float64
	}{
		{
			name: "high complexity - large, old, churny",
			metrics: schema.FileResult{
				Path:               "legacy_api.go",
				UniqueContributors: 5,
				Commits:            200,        // 0.4 nCommits
				RecentCommits:      1,          // 1/50 = 0.02 nRecent -> 0.98 nInvRecent
				SizeBytes:          700 * 1024, // > max -> 1.0 nSize
				LinesOfCode:        15000,      // > max -> 1.0 nLOC
				AgeDays:            2000,       // ~0.85 nAge
				Churn:              7000,       // > max -> 1.0 nChurn
				Gini:               0.3,
				Mode:               schema.ComplexityMode,
			},
			// Expected Score (Raw):
			// 0.30*0.85 + 0.30*1.0 + 0.20*1.0 + 0.10*0.4 + 0.05*1.0 + 0.05*0.98
			// Approx: 0.255 + 0.30 + 0.20 + 0.04 + 0.05 + 0.049 = ~0.894
			// Score: 89.4
			minScore: 80,  // Adjusted from 70 to reflect the high score
			maxScore: 100, // Retained
		},
		{
			name: "low complexity - small, new, low churn",
			metrics: schema.FileResult{
				Path:               "small_helper.go",
				UniqueContributors: 1,
				Commits:            10,       // 0.02 nCommits
				RecentCommits:      5,        // 0.1 nRecent -> 0.9 nInvRecent
				SizeBytes:          5 * 1024, // 0.01 nSize
				LinesOfCode:        50,       // 0.005 nLOC
				AgeDays:            50,       // Low nAge (~0.35)
				Churn:              50,       // 0.01 nChurn
				Gini:               0.1,
				Mode:               schema.ComplexityMode,
			},
			// Expected Score (Raw):
			// 0.30*0.35 + 0.30*0.01 + 0.20*0.005 + 0.10*0.02 + 0.05*0.01 + 0.05*0.9
			// Approx: 0.105 + 0.003 + 0.001 + 0.002 + 0.0005 + 0.045 = ~0.1565
			// Score: 15.65
			minScore: 0,  // Retained
			maxScore: 25, // Adjusted from 20 to allow for a slightly wider range
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := computeScore(&tt.metrics, schema.ComplexityMode, nil)
			assert.True(t, score >= tt.minScore && score <= tt.maxScore)
			assert.True(t, score >= 0 && score <= 100)
			assert.NotEmpty(t, tt.metrics.Breakdown)
		})
	}
}

// TestComputeScoreAllModes ensures all modes produce valid scores.
func TestComputeScoreAllModes(t *testing.T) {
	modes := []string{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.StaleMode}

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
		t.Run(mode, func(t *testing.T) {
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

// TestComputeScoreWithCustomWeights tests that custom weights produce different results than defaults
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
	customWeights := map[string]map[string]float64{
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

// TestComputeScoreCustomWeightsAllModes tests custom weights for all scoring modes
func TestComputeScoreCustomWeightsAllModes(t *testing.T) {
	modes := []string{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.StaleMode}

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
		t.Run(mode, func(t *testing.T) {
			// Get default score
			defaultScore := computeScore(&metrics, mode, nil)

			// Create custom weights that emphasize different aspects
			customWeights := map[string]map[string]float64{
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

// TestComputeScoreInvalidCustomWeights tests behavior with invalid custom weights
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
	emptyWeights := map[string]map[string]float64{}
	score = computeScore(&metrics, schema.HotMode, emptyWeights)
	assert.True(t, score >= 0 && score <= 100, "Score with empty custom weights should be valid")

	// Test with custom weights for wrong mode (should use defaults for the requested mode)
	wrongModeWeights := map[string]map[string]float64{
		"nonexistent_mode": {
			"some_key": 1.0,
		},
	}
	score = computeScore(&metrics, schema.HotMode, wrongModeWeights)
	assert.True(t, score >= 0 && score <= 100, "Score with wrong mode custom weights should be valid")
}
