package outwriter

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestGetDisplayNameForMode(t *testing.T) {
	tests := []struct {
		name     string
		modeName string
		expected string
	}{
		{
			name:     "hot mode",
			modeName: "hot",
			expected: "🔥 HOT",
		},
		{
			name:     "risk mode",
			modeName: "risk",
			expected: "⚠️  RISK",
		},
		{
			name:     "complexity mode",
			modeName: "complexity",
			expected: "🧩 COMPLEXITY",
		},
		{
			name:     "stale mode",
			modeName: "stale",
			expected: "🕰️  STALE",
		},
		{
			name:     "unknown mode",
			modeName: "custom",
			expected: "CUSTOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDisplayNameForMode(tt.modeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDisplayWeightsForMode(t *testing.T) {
	tests := []struct {
		name          string
		mode          schema.ScoringMode
		activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64
		checkKeys     []string
	}{
		{
			name:          "hot mode default weights",
			mode:          schema.HotMode,
			activeWeights: nil,
			checkKeys:     []string{"commits", "churn", "contrib"},
		},
		{
			name: "hot mode with custom weights",
			mode: schema.HotMode,
			activeWeights: map[schema.ScoringMode]map[schema.BreakdownKey]float64{
				schema.HotMode: {
					schema.BreakdownCommits: 0.8,
					schema.BreakdownChurn:   0.2,
				},
			},
			checkKeys: []string{"commits", "churn"},
		},
		{
			name:          "risk mode default weights",
			mode:          schema.RiskMode,
			activeWeights: nil,
			checkKeys:     []string{"inv_contrib", "gini"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDisplayWeightsForMode(tt.mode, tt.activeWeights)
			assert.NotNil(t, result)
			// Check that expected keys exist
			for _, key := range tt.checkKeys {
				_, exists := result[key]
				assert.True(t, exists, "Expected key %s to exist in weights", key)
			}
		})
	}
}

func TestFormatWeights(t *testing.T) {
	tests := []struct {
		name       string
		weights    map[string]float64
		factorKeys []string
		expected   string
	}{
		{
			name: "simple weights",
			weights: map[string]float64{
				"commits": 0.5,
				"churn":   0.5,
			},
			factorKeys: []string{"commits", "churn"},
			expected:   "0.50*commits+0.50*churn",
		},
		{
			name: "single weight",
			weights: map[string]float64{
				"age": 1.0,
			},
			factorKeys: []string{"age"},
			expected:   "1.00*age",
		},
		{
			name: "zero weight ignored",
			weights: map[string]float64{
				"commits": 0.7,
				"churn":   0.0,
				"age":     0.3,
			},
			factorKeys: []string{"commits", "churn", "age"},
			expected:   "0.70*commits+0.30*age",
		},
		{
			name:       "empty weights",
			weights:    map[string]float64{},
			factorKeys: []string{},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWeights(tt.weights, tt.factorKeys)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTopMetricBreakdown(t *testing.T) {
	tests := []struct {
		name     string
		file     *schema.FileResult
		expected string
	}{
		{
			name: "top 3 contributors",
			file: &schema.FileResult{
				Breakdown: map[schema.BreakdownKey]float64{
					schema.BreakdownCommits: 40.0,
					schema.BreakdownChurn:   30.0,
					schema.BreakdownAge:     20.0,
					schema.BreakdownSize:    10.0,
				},
			},
			expected: "commits > churn > age",
		},
		{
			name: "less than 3 contributors",
			file: &schema.FileResult{
				Breakdown: map[schema.BreakdownKey]float64{
					schema.BreakdownCommits: 60.0,
					schema.BreakdownChurn:   40.0,
				},
			},
			expected: "commits > churn",
		},
		{
			name: "single contributor",
			file: &schema.FileResult{
				Breakdown: map[schema.BreakdownKey]float64{
					schema.BreakdownAge: 100.0,
				},
			},
			expected: "age",
		},
		{
			name: "all below minimum threshold",
			file: &schema.FileResult{
				Breakdown: map[schema.BreakdownKey]float64{
					schema.BreakdownCommits: 0.3,
					schema.BreakdownChurn:   0.2,
				},
			},
			expected: "Not applicable",
		},
		{
			name: "empty breakdown",
			file: &schema.FileResult{
				Breakdown: map[schema.BreakdownKey]float64{},
			},
			expected: "Not applicable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTopMetricBreakdown(tt.file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildMetricsRenderModel(t *testing.T) {
	// Test with nil active weights
	model := buildMetricsRenderModel(nil)
	assert.NotNil(t, model)
	assert.Equal(t, "Hotspot Scoring Modes", model.Title)
	assert.Len(t, model.Modes, 4) // hot, risk, complexity, stale

	// Verify each mode has expected structure
	for _, mode := range model.Modes {
		assert.NotEmpty(t, mode.Name)
		assert.NotEmpty(t, mode.Purpose)
		assert.NotEmpty(t, mode.Factors)
		assert.NotEmpty(t, mode.Formula)
		assert.NotNil(t, mode.Weights)
	}

	// Test with custom active weights
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.HotMode: {
			schema.BreakdownCommits: 0.9,
			schema.BreakdownChurn:   0.1,
		},
	}
	model = buildMetricsRenderModel(activeWeights)
	assert.NotNil(t, model)
	
	// Find hot mode and verify custom weights were applied
	for _, mode := range model.Modes {
		if mode.Name == "hot" {
			assert.Equal(t, 0.9, mode.Weights["commits"])
			assert.Equal(t, 0.1, mode.Weights["churn"])
		}
	}
}
