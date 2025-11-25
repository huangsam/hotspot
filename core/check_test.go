package core

import (
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestPrintCheckResult(t *testing.T) {
	// Test that printCheckResult doesn't panic with various inputs
	tests := []struct {
		name   string
		result schema.CheckResult
	}{
		{
			name: "all passed",
			result: schema.CheckResult{
				Passed:       true,
				FailedFiles:  []schema.CheckFailedFile{},
				TotalFiles:   5,
				CheckedModes: []schema.ScoringMode{schema.HotMode, schema.RiskMode},
				BaseRef:      "main",
				TargetRef:    "HEAD",
				Thresholds: map[schema.ScoringMode]float64{
					schema.HotMode:        50.0,
					schema.RiskMode:       50.0,
					schema.ComplexityMode: 50.0,
					schema.StaleMode:      50.0,
				},
				MaxScores: map[schema.ScoringMode]float64{
					schema.HotMode:        30.0,
					schema.RiskMode:       25.0,
					schema.ComplexityMode: 40.0,
					schema.StaleMode:      20.0,
				},
				Lookback: 180 * 24 * time.Hour,
			},
		},
		{
			name: "some failed",
			result: schema.CheckResult{
				Passed: false,
				FailedFiles: []schema.CheckFailedFile{
					{
						Path:      "main.go",
						Mode:      schema.HotMode,
						Score:     75.5,
						Threshold: 50.0,
					},
				},
				TotalFiles:   5,
				CheckedModes: []schema.ScoringMode{schema.HotMode, schema.RiskMode},
				BaseRef:      "main",
				TargetRef:    "HEAD",
				Thresholds: map[schema.ScoringMode]float64{
					schema.HotMode:        50.0,
					schema.RiskMode:       50.0,
					schema.ComplexityMode: 50.0,
					schema.StaleMode:      50.0,
				},
				MaxScores: map[schema.ScoringMode]float64{
					schema.HotMode:        75.5,
					schema.RiskMode:       45.0,
					schema.ComplexityMode: 60.0,
					schema.StaleMode:      30.0,
				},
				Lookback: 180 * 24 * time.Hour,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure it doesn't panic
			assert.NotPanics(t, func() {
				printCheckResult(&tt.result, time.Second)
			})
		})
	}
}
