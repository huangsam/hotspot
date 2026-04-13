package outwriter

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/huangsam/hotspot/internal/config"
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

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format:    schema.TextOut,
			Precision: 2,
			UseColors: false,
			Width:     120,
		},
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg.Output)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Hotspot Scoring Modes")
	assert.Contains(t, output, "Hot: Activity hotspots")
	assert.Contains(t, output, "Risk: Knowledge risk/bus factor")
	assert.Contains(t, output, "Formula: Score = 0.50*commits+0.50*churn")
	assert.Contains(t, output, "Formula: Score = 0.60*inv_contrib+0.40*gini")
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

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format:    schema.CSVOut,
			Precision: 2,
		},
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg.Output)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 6) // header + 5 rows (all modes)

	assert.Contains(t, lines[0], "Mode")
	assert.Contains(t, lines[0], "Purpose")
	assert.Contains(t, lines[0], "Factors")
	assert.Contains(t, lines[0], "Formula")
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
