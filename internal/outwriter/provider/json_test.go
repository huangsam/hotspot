// Package provider_test contains unit tests for the JSON provider.
package provider

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFiles(t *testing.T) {
	p := NewJSONProvider()
	files := []schema.FileResult{
		{
			Path:      "test.go",
			ModeScore: 85.5,
			Mode:      schema.HotMode,
		},
	}
	runtime := config.RuntimeConfig{
		Workers: 4,
	}

	var buf bytes.Buffer
	duration := 100 * time.Millisecond
	err := p.WriteFiles(&buf, files, config.OutputConfig{}, runtime, duration)
	require.NoError(t, err)

	var output schema.FileResultsOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	require.Len(t, output.Results, 1)
	assert.Equal(t, "test.go", output.Results[0].Path)
	assert.Equal(t, 85.5, output.Results[0].ModeScore)
	assert.Equal(t, "Critical", output.Results[0].Label)
	assert.Equal(t, 1, output.Results[0].Rank)

	assert.Equal(t, 4, output.Metadata.Workers)
	assert.Equal(t, duration, output.Metadata.AnalysisDuration)
}

func TestWriteFolders(t *testing.T) {
	p := NewJSONProvider()
	folders := []schema.FolderResult{
		{
			Path:  "cmd/",
			Score: 70.0,
			Mode:  schema.RiskMode,
		},
	}
	runtime := config.RuntimeConfig{Workers: 2}

	var buf bytes.Buffer
	duration := 50 * time.Millisecond
	err := p.WriteFolders(&buf, folders, config.OutputConfig{}, runtime, duration)
	require.NoError(t, err)

	var output schema.FolderResultsOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	require.Len(t, output.Results, 1)
	assert.Equal(t, "cmd/", output.Results[0].Path)
	assert.Equal(t, 70.0, output.Results[0].Score)
	assert.Equal(t, "High", output.Results[0].Label)
}

func TestWriteMetrics(t *testing.T) {
	p := NewJSONProvider()
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.HotMode: {
			schema.BreakdownCommits: 1.0,
		},
	}

	var buf bytes.Buffer
	err := p.WriteMetrics(&buf, activeWeights, config.OutputConfig{})
	require.NoError(t, err)

	var model schema.MetricsRenderModel
	err = json.Unmarshal(buf.Bytes(), &model)
	require.NoError(t, err)

	assert.Equal(t, "Hotspot Scoring Modes", model.Title)
	// Find hot mode
	var hotMode *schema.MetricsModeWithData
	for _, m := range model.Modes {
		if m.Name == "hot" {
			hotMode = &m
			break
		}
	}
	require.NotNil(t, hotMode)
	assert.Equal(t, 1.0, hotMode.Weights[string(schema.BreakdownCommits)])
}

func TestWriteComparison(t *testing.T) {
	p := NewJSONProvider()
	comparison := schema.ComparisonResult{
		Details: []schema.ComparisonDetail{
			{
				Path:        "main.go",
				BeforeScore: 70.0,
				AfterScore:  80.0,
				Delta:       10.0,
			},
		},
	}
	runtime := config.RuntimeConfig{Workers: 1}

	var buf bytes.Buffer
	duration := 150 * time.Millisecond
	err := p.WriteComparison(&buf, comparison, config.OutputConfig{}, runtime, duration)
	require.NoError(t, err)

	var output schema.ComparisonResultsOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	require.NoError(t, err)

	require.Len(t, output.Results.Details, 1)
	assert.Equal(t, "main.go", output.Results.Details[0].Path)
	assert.Equal(t, 10.0, output.Results.Details[0].Delta)
}
