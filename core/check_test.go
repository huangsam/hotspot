package core

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w
	os.Stderr = w

	fn()

	assert.NoError(t, w.Close())
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	out, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.NoError(t, r.Close())
	return string(out)
}

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
				},
				MaxScores: map[schema.ScoringMode]float64{
					schema.HotMode:        30.0,
					schema.RiskMode:       25.0,
					schema.ComplexityMode: 40.0,
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
				},
				MaxScores: map[schema.ScoringMode]float64{
					schema.HotMode:        75.5,
					schema.RiskMode:       45.0,
					schema.ComplexityMode: 60.0,
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

func TestPrintCheckResult_ShowsReasoningOnFailures(t *testing.T) {
	result := schema.CheckResult{
		Passed: false,
		FailedFiles: []schema.CheckFailedFile{
			{
				Path:      "main.go",
				Mode:      schema.HotMode,
				Score:     75.5,
				Threshold: 50.0,
				Reasoning: []string{"High Churn", "Concentrated ownership"},
			},
		},
		TotalFiles:   5,
		CheckedModes: []schema.ScoringMode{schema.HotMode, schema.RiskMode},
		BaseRef:      "main",
		TargetRef:    "HEAD",
		Thresholds: map[schema.ScoringMode]float64{
			schema.HotMode:  50.0,
			schema.RiskMode: 50.0,
		},
		MaxScores: map[schema.ScoringMode]float64{
			schema.HotMode: 75.5,
		},
		Lookback: 180 * 24 * time.Hour,
	}

	output := captureOutput(t, func() {
		printCheckResult(&result, time.Second)
	})

	assert.Contains(t, output, "reason: High Churn")
	assert.Contains(t, output, "reason: Concentrated ownership")
}

func TestPrintCheckResult_FailureOutputGolden(t *testing.T) {
	result := schema.CheckResult{
		Passed: false,
		FailedFiles: []schema.CheckFailedFile{
			{
				Path:      "cmd/a.go",
				Mode:      schema.HotMode,
				Score:     81.0,
				Threshold: 50.0,
				Reasoning: []string{"High Churn", "Active Frontier"},
			},
			{
				Path:      "cmd/b.go",
				Mode:      schema.HotMode,
				Score:     70.0,
				Threshold: 50.0,
				Reasoning: []string{"Development Bottleneck"},
			},
		},
		TotalFiles:   3,
		CheckedModes: []schema.ScoringMode{schema.HotMode, schema.RiskMode},
		BaseRef:      "main",
		TargetRef:    "HEAD",
		Thresholds: map[schema.ScoringMode]float64{
			schema.HotMode:  50.0,
			schema.RiskMode: 45.0,
		},
		MaxScores: map[schema.ScoringMode]float64{
			schema.HotMode: 81.0,
		},
		Lookback: 180 * 24 * time.Hour,
	}

	output := captureOutput(t, func() {
		printCheckResult(&result, 2*time.Second)
	})

	expected := "" +
		"Policy Check Results:\n" +
		"  Base:        main\n" +
		"  Target:      HEAD\n" +
		"  Lookback:    4320h0m0s\n" +
		"  Thresholds:  hot=50.0, risk=45.0\n\n" +
		"Checked 3 files in 2s\n\n" +
		"FAIL: Policy check failed: 2 violation(s) found across 3 files\n\n" +
		"Mode: hot (2 violations)\n" +
		"  - cmd/a.go (score: 81.0 > threshold: 50.0)\n" +
		"      reason: High Churn\n" +
		"      reason: Active Frontier\n" +
		"  - cmd/b.go (score: 70.0 > threshold: 50.0)\n" +
		"      reason: Development Bottleneck\n\n"

	assert.Equal(t, expected, output)
}
