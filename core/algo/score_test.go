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
	modes := []schema.ScoringMode{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.ROIMode}

	metrics := schema.FileResult{
		Path:               "test.go",
		UniqueContributors: schema.Metric(5),
		Commits:            schema.Metric(50),
		RecentCommits:      schema.Metric(10),
		SizeBytes:          50 * 1024,
		AgeDays:            schema.Metric(365),
		Churn:              schema.Metric(250),
		DecayedCommits:     schema.Metric(50),
		DecayedChurn:       schema.Metric(250),
		Gini:               0.3,
		FirstCommit:        time.Now().AddDate(0, 0, -365),
		Mode:               schema.HotMode,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			weights := getWeightsForMode(mode, nil)
			score := ComputeScore(&metrics, mode, weights, 0.1, 0.4)
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
		DecayedCommits:     50,
		DecayedChurn:       250,
		Gini:               0.3,
		Mode:               schema.HotMode,
	}

	weights := getWeightsForMode(schema.HotMode, nil)

	for b.Loop() {
		ComputeScore(&metrics, schema.HotMode, weights, 0.1, 0.4)
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
		DecayedCommits:     50,
		DecayedChurn:       250,
		Gini:               0.3,
		Mode:               schema.HotMode,
	}

	// Get default score
	defaultWeights := getWeightsForMode(schema.HotMode, nil)
	defaultScore := ComputeScore(&metrics, schema.HotMode, defaultWeights, 0.1, 0.4)

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
	customScore := ComputeScore(&metrics, schema.HotMode, customModeWeights, 0.1, 0.4)

	// Scores should be different
	assert.NotEqual(t, defaultScore, customScore, "Custom weights should produce different score than defaults")

	// Both should be valid scores
	assert.True(t, defaultScore >= 0 && defaultScore <= 100, "Default score should be valid")
	assert.True(t, customScore >= 0 && customScore <= 100, "Custom score should be valid")
}

// TestComputeScoreCustomWeightsAllModes tests custom weights for all scoring modes.
func TestComputeScoreCustomWeightsAllModes(t *testing.T) {
	modes := []schema.ScoringMode{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.ROIMode}

	metrics := schema.FileResult{
		Path:               "test.go",
		UniqueContributors: schema.Metric(5),
		Commits:            schema.Metric(50),
		RecentCommits:      schema.Metric(10),
		SizeBytes:          50 * 1024,
		AgeDays:            schema.Metric(365),
		Churn:              schema.Metric(250),
		DecayedCommits:     schema.Metric(50),
		DecayedChurn:       schema.Metric(250),
		Gini:               0.3,
		LinesOfCode:        schema.Metric(500),
		Mode:               schema.HotMode,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			// Get default score
			defaultWeights := getWeightsForMode(mode, nil)
			defaultScore := ComputeScore(&metrics, mode, defaultWeights, 0.1, 0.4)

			// Create custom weights that emphasize different aspects
			customWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				mode: {},
			}

			// Add some custom weights based on mode
			switch mode {
			case schema.HotMode:
				customWeights[mode][schema.BreakdownCommits] = 0.6
				customWeights[mode][schema.BreakdownChurn] = 0.4
			case schema.ComplexityMode:
				customWeights[mode][schema.BreakdownSize] = 0.4
				customWeights[mode][schema.BreakdownLOC] = 0.4
				customWeights[mode][schema.BreakdownAge] = 0.2
			case schema.ROIMode:
				customWeights[mode][schema.BreakdownChurn] = 0.5
				customWeights[mode][schema.BreakdownLOC] = 0.5
			}

			customModeWeights := getWeightsForMode(mode, customWeights)
			customScore := ComputeScore(&metrics, mode, customModeWeights, 0.1, 0.4)

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

// TestComputeScoreRiskKnowledgeDecay verifies that staleness (LowRecent) impacts the Risk mode score.
func TestComputeScoreRiskKnowledgeDecay(t *testing.T) {
	// Case 1: Active file (RecentCommits = 50, max)
	activeFile := &schema.FileResult{
		Path:               "active.go",
		UniqueContributors: 1, // High base risk
		RecentCommits:      50,
		Gini:               1.0,
		SizeBytes:          1000,
	}

	// Case 2: Stale file (RecentCommits = 0)
	staleFile := &schema.FileResult{
		Path:               "stale.go",
		UniqueContributors: 1, // Same base risk
		RecentCommits:      0,
		Gini:               1.0,
		SizeBytes:          1000,
	}

	weights := getWeightsForMode(schema.RiskMode, nil)
	activeScore := ComputeScore(activeFile, schema.RiskMode, weights, 0.1, 0.4)
	staleScore := ComputeScore(staleFile, schema.RiskMode, weights, 0.1, 0.4)

	// Stale file should have a higher risk score due to LowRecent factor (15%)
	assert.Greater(t, staleScore, activeScore, "Stale file should have higher RISK score than active file with same ownership")

	// Verify breakdown contains low_recent
	assert.Greater(t, staleFile.ModeBreakdown[schema.BreakdownLowRecent], 0.0)
	assert.Equal(t, 0.0, activeFile.ModeBreakdown[schema.BreakdownLowRecent])
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
	score := ComputeScore(&metrics, schema.HotMode, defaultWeights, 0.1, 0.4)
	assert.True(t, score >= 0 && score <= 100, "Score with nil custom weights should be valid")

	// Test with empty custom weights map (should use defaults)
	emptyWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{}
	emptyModeWeights := getWeightsForMode(schema.HotMode, emptyWeights)
	score = ComputeScore(&metrics, schema.HotMode, emptyModeWeights, 0.1, 0.4)
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
				DecayedCommits:     10,
				DecayedChurn:       100,
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
				DecayedCommits:     10,
				DecayedChurn:       50,
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
		f.Add(
			seed.result.Path,
			seed.result.SizeBytes,
			seed.result.LinesOfCode.Float64(),
			seed.result.Commits.Float64(),
			seed.result.Churn.Float64(),
			seed.result.AgeDays.Float64(),
			seed.result.UniqueContributors.Float64(),
			seed.result.Gini,
			seed.result.RecentCommits.Float64(),
			string(seed.mode),
		)
	}

	f.Fuzz(func(_ *testing.T,
		path string,
		sizeBytes int64,
		linesOfCode float64,
		commits float64,
		churn float64,
		ageDays float64,
		uniqueContributors float64,
		gini float64,
		recentCommits float64,
		mode string,
	) {
		result := schema.FileResult{
			Path:               path,
			SizeBytes:          sizeBytes,
			LinesOfCode:        schema.Metric(linesOfCode),
			Commits:            schema.Metric(commits),
			Churn:              schema.Metric(churn),
			AgeDays:            schema.Metric(ageDays),
			UniqueContributors: schema.Metric(uniqueContributors),
			Gini:               gini,
			RecentCommits:      schema.Metric(recentCommits),
			Mode:               schema.ScoringMode(mode),
		}
		_ = ComputeScore(&result, schema.ScoringMode(mode), getWeightsForMode(schema.ScoringMode(mode), nil), 0.1, 0.4)
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

// TestComputeScore_EdgeCases tests boundary conditions and extreme metric values.
func TestComputeScore_EdgeCases(t *testing.T) {
	defaultWeights := schema.GetDefaultWeights(schema.HotMode)

	tests := []struct {
		name     string
		metrics  *schema.FileResult
		mode     schema.ScoringMode
		weights  map[schema.BreakdownKey]float64
		expected float64
	}{
		{
			name: "zero byte file always scores zero",
			metrics: &schema.FileResult{
				SizeBytes: 0,
				Commits:   100, // Should be ignored
				Churn:     1000,
			},
			mode:     schema.HotMode,
			weights:  defaultWeights,
			expected: 0.0,
		},
		{
			name: "extreme saturation - commits",
			metrics: &schema.FileResult{
				SizeBytes: 1000,
				Commits:   schema.Metric(1000000), // Far exceeds maxCommits
				AgeDays:   schema.Metric(10),
			},
			mode:    schema.HotMode,
			weights: defaultWeights,
		},
		{
			name: "roi mode with zero lines of code",
			metrics: &schema.FileResult{
				SizeBytes:   1000,
				LinesOfCode: 0, // Potential division by zero check
				Churn:       1000,
			},
			mode:    schema.ROIMode,
			weights: schema.GetDefaultWeights(schema.ROIMode),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ComputeScore(tt.metrics, tt.mode, tt.weights, 0.1, 0.4)
			if tt.name == "zero byte file always scores zero" {
				assert.Equal(t, tt.expected, score)
			} else {
				assert.True(t, score >= 0 && score <= 100)
			}
		})
	}
}

func TestIsConfigurationFile(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".yml", true},
		{".yaml", true},
		{"json", true},
		{".xml", true},
		{".lock", true},
		{".sum", true},
		{".md", true},
		{".txt", true},
		{".tfstate", true},
		{".go", false},
		{".rs", false},
		{".java", false},
		{".gradle", false},
		{"Makefile", false},
		{".csv", true},
		{".ini", true},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			assert.Equal(t, tt.expected, isConfigurationFile(tt.ext))
		})
	}
}
