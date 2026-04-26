package provider

import (
	"strings"
	"testing"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeatmapProvider_WriteFiles(t *testing.T) {
	p := NewHeatmapProvider()
	require.NotNil(t, p)

	files := []schema.FileResult{
		{Path: "core/analysis.go", ModeScore: 10.0, SizeBytes: 1000, Churn: 50},
		{Path: "core/core.go", ModeScore: 20.0, SizeBytes: 2000, Churn: 100},
		{Path: "cmd/main.go", ModeScore: 5.0, SizeBytes: 500, Churn: 25},
	}

	output := config.OutputConfig{}
	runtime := config.RuntimeConfig{}

	var buf strings.Builder
	err := p.WriteFiles(&buf, files, output, runtime, 0)
	require.NoError(t, err)

	svg := buf.String()
	assert.Contains(t, svg, "<svg")
	assert.Contains(t, svg, "</svg>")
	assert.Contains(t, svg, "analysis.go")
	assert.Contains(t, svg, "core.go")
	assert.Contains(t, svg, "main.go")
	assert.Contains(t, svg, "Score: 10.0")
	assert.Contains(t, svg, "Score: 20.0")
	assert.Contains(t, svg, "Score: 5.0")
}

func TestHeatmapProvider_WriteFolders(t *testing.T) {
	p := NewHeatmapProvider()
	require.NotNil(t, p)

	folders := []schema.FolderResult{
		{Path: "core", Score: 15.0, TotalLOC: 3000, Churn: 150},
		{Path: "cmd", Score: 7.0, TotalLOC: 1000, Churn: 50},
	}

	output := config.OutputConfig{}
	runtime := config.RuntimeConfig{}

	var buf strings.Builder
	err := p.WriteFolders(&buf, folders, output, runtime, 0)
	require.NoError(t, err)

	svg := buf.String()
	assert.Contains(t, svg, "<svg")
	assert.Contains(t, svg, "</svg>")
	assert.Contains(t, svg, "core")
	assert.Contains(t, svg, "cmd")
}

func TestHeatmapProvider_WriteFiles_Empty(t *testing.T) {
	p := NewHeatmapProvider()

	files := []schema.FileResult{}
	output := config.OutputConfig{}
	runtime := config.RuntimeConfig{}

	var buf strings.Builder
	err := p.WriteFiles(&buf, files, output, runtime, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no files to visualize")
}

func TestHeatmapProvider_WriteComparison_Error(t *testing.T) {
	p := NewHeatmapProvider()

	var buf strings.Builder
	err := p.WriteComparison(&buf, schema.ComparisonResult{}, config.OutputConfig{}, config.RuntimeConfig{}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "heatmap output not supported for comparison results")
}

func TestHeatmapProvider_WriteTimeseries_Error(t *testing.T) {
	p := NewHeatmapProvider()

	var buf strings.Builder
	err := p.WriteTimeseries(&buf, schema.TimeseriesResult{}, config.OutputConfig{}, config.RuntimeConfig{}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "heatmap output not supported for timeseries results")
}

func TestHeatmapProvider_WriteBlastRadius_Error(t *testing.T) {
	p := NewHeatmapProvider()

	var buf strings.Builder
	err := p.WriteBlastRadius(&buf, schema.BlastRadiusResult{}, config.OutputConfig{}, config.RuntimeConfig{}, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "heatmap output not supported for blast radius results")
}

func TestHeatmapProvider_WriteMetrics_Error(t *testing.T) {
	p := NewHeatmapProvider()

	err := p.WriteMetrics(nil, nil, config.OutputConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "heatmap output not supported for metrics")
}

func TestHeatmapProvider_WriteHistory_Error(t *testing.T) {
	p := NewHeatmapProvider()

	err := p.WriteHistory(nil, nil, config.OutputConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "heatmap output not supported for history")
}

func TestHeatmapProvider_GenerateSVG(t *testing.T) {
	p := NewHeatmapProvider()

	files := []schema.FileResult{
		{Path: "test.go", ModeScore: 8.0, SizeBytes: 100, Churn: 10},
	}

	output := config.OutputConfig{}
	svg, err := p.GenerateSVG(files, output)
	require.NoError(t, err)
	assert.Contains(t, svg, "<svg")
	assert.Contains(t, svg, "test.go")
	assert.Contains(t, svg, "Score: 8.0")
}
