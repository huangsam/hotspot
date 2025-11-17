package outwriter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMetricsTable(t *testing.T) {
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.HotMode: {
			schema.BreakdownCommits: 0.5,
			schema.BreakdownChurn:   0.5,
		},
		schema.RiskMode: {
			schema.BreakdownInvContrib: 0.6,
			schema.BreakdownGini:       0.4,
		},
	}

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
		UseColors: false,
		Width:     120,
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Hotspot Scoring Modes")
	assert.Contains(t, output, "Hot: Activity hotspots")
	assert.Contains(t, output, "Risk: Knowledge risk/bus factor")
	assert.Contains(t, output, "Formula: Score = 0.50*commits+0.50*churn")
	assert.Contains(t, output, "Formula: Score = 0.60*inv_contrib+0.40*gini")
}

func TestWriteMetricsJSON(t *testing.T) {
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.HotMode: {
			schema.BreakdownCommits: 0.5,
			schema.BreakdownChurn:   0.5,
		},
	}

	cfg := &contract.Config{
		Output:    schema.JSONOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "Hotspot Scoring Modes", result["title"])
	assert.Contains(t, result, "modes")
}

func TestWriteMetricsCSV(t *testing.T) {
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.RiskMode: {
			schema.BreakdownInvContrib: 0.6,
			schema.BreakdownGini:       0.4,
		},
		schema.ComplexityMode: {
			schema.BreakdownSize: 0.7,
			schema.BreakdownAge:  0.3,
		},
	}

	cfg := &contract.Config{
		Output:    schema.CSVOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 5) // header + 4 rows (all modes)

	assert.Contains(t, lines[0], "Mode")
	assert.Contains(t, lines[0], "Purpose")
	assert.Contains(t, lines[0], "Factors")
	assert.Contains(t, lines[0], "Formula")
}

func TestWriteJSONMetrics(t *testing.T) {
	renderModel := &schema.MetricsRenderModel{
		Title:       "Test Metrics",
		Description: "Test description",
		Modes: []schema.MetricsModeWithData{
			{
				MetricsMode: schema.MetricsMode{
					Name:    "hot",
					Purpose: "Activity hotspots",
					Factors: []string{"Commits", "Churn"},
				},
				Weights: map[string]float64{
					"commits": 0.5,
					"churn":   0.5,
				},
				Formula: "0.5*commits+0.5*churn",
			},
		},
	}

	var buf bytes.Buffer
	err := writeJSONMetrics(&buf, renderModel)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "Test Metrics", result["title"])
	assert.Contains(t, result, "modes")
}

func TestWriteCSVMetrics(t *testing.T) {
	renderModel := &schema.MetricsRenderModel{
		Modes: []schema.MetricsModeWithData{
			{
				MetricsMode: schema.MetricsMode{
					Name:    "risk",
					Purpose: "Knowledge risk",
					Factors: []string{"InvContributors", "Gini"},
				},
				Formula: "0.6*inv_contrib+0.4*gini",
			},
		},
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := writeCSVMetrics(w, renderModel)
	require.NoError(t, err)
	w.Flush()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "Mode")
	assert.Contains(t, lines[0], "Purpose")
	assert.Contains(t, lines[1], "risk")
	assert.Contains(t, lines[1], "Knowledge risk")
}
