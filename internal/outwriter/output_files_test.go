package outwriter

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCSVResults(t *testing.T) {
	fmtFloat, intFmt := createFormatters(2)
	files := []schema.FileResult{
		{
			Path:               "file1.go",
			ModeScore:          75.25,
			UniqueContributors: 2,
			Commits:            5,
			SizeBytes:          2048,
			AgeDays:            60,
			Churn:              300,
			Gini:               0.5,
			LinesOfCode:        200,
			FirstCommit:        time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
			Owners:             []string{"Charlie"},
			Mode:               schema.RiskMode,
		},
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := writeCSVResultsForFiles(w, files, fmtFloat, intFmt)
	require.NoError(t, err)
	w.Flush()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2) // header + 1 row

	// Check header
	assert.Contains(t, lines[0], "rank")
	assert.Contains(t, lines[0], "file")
	assert.Contains(t, lines[0], "score")

	// Check data row
	assert.Contains(t, lines[1], "1")
	assert.Contains(t, lines[1], "file1.go")
	assert.Contains(t, lines[1], "75.25")
	assert.Contains(t, lines[1], "risk")
}

func TestWriteCSVResultsEmptyFiles(t *testing.T) {
	fmtFloat, intFmt := createFormatters(2)
	var files []schema.FileResult

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := writeCSVResultsForFiles(w, files, fmtFloat, intFmt)
	require.NoError(t, err)
	w.Flush()

	// Should only have header
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], "rank")
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
				ModeBreakdown: map[schema.BreakdownKey]float64{
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
				ModeBreakdown: map[schema.BreakdownKey]float64{
					schema.BreakdownCommits: 60.0,
					schema.BreakdownChurn:   40.0,
				},
			},
			expected: "commits > churn",
		},
		{
			name: "single contributor",
			file: &schema.FileResult{
				ModeBreakdown: map[schema.BreakdownKey]float64{
					schema.BreakdownAge: 100.0,
				},
			},
			expected: "age",
		},
		{
			name: "all below minimum threshold",
			file: &schema.FileResult{
				ModeBreakdown: map[schema.BreakdownKey]float64{
					schema.BreakdownCommits: 0.3,
					schema.BreakdownChurn:   0.2,
				},
			},
			expected: "Not applicable",
		},
		{
			name: "empty breakdown",
			file: &schema.FileResult{
				ModeBreakdown: map[schema.BreakdownKey]float64{},
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

func TestWriteFileResultsTable(t *testing.T) {
	files := []schema.FileResult{
		{
			Path:               "main.go",
			ModeScore:          85.5,
			UniqueContributors: 3,
			Commits:            10,
			SizeBytes:          1024,
			AgeDays:            30,
			Churn:              500,
			Gini:               0.6,
			LinesOfCode:        100,
			FirstCommit:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			Owners:             []string{"Alice", "Bob"},
			Mode:               schema.HotMode,
		},
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format:    schema.TextOut,
			Precision: 2,
			Detail:    true,
			Explain:   true,
			Owner:     true,
			UseColors: false,
			Width:     120,
		},
	}

	var buf bytes.Buffer
	duration := 100 * time.Millisecond
	err := WriteFileResults(&buf, files, cfg.Output, cfg.Runtime, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "main.go")
	assert.Contains(t, output, "85.50")
	assert.Contains(t, output, "3")
	assert.Contains(t, output, "10")
	assert.Contains(t, output, "100")
	assert.Contains(t, output, "500")
	assert.Contains(t, output, "30")
	assert.Contains(t, output, "0.60")
	assert.Contains(t, output, "Alice, Bob")
	assert.Contains(t, output, "Analysis completed in 100ms")
}

func TestWriteFileResultsCSV(t *testing.T) {
	files := []schema.FileResult{
		{
			Path:               "example.go",
			ModeScore:          65.25,
			UniqueContributors: 2,
			Commits:            5,
			SizeBytes:          2048,
			AgeDays:            60,
			Churn:              300,
			Gini:               0.5,
			LinesOfCode:        200,
			FirstCommit:        time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
			Owners:             []string{"Charlie"},
			Mode:               schema.ComplexityMode,
		},
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format:    schema.CSVOut,
			Precision: 2,
		},
	}

	var buf bytes.Buffer
	duration := 75 * time.Millisecond
	err := WriteFileResults(&buf, files, cfg.Output, cfg.Runtime, duration)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "rank")
	assert.Contains(t, lines[0], "file")
	assert.Contains(t, lines[0], "score")
	assert.Contains(t, lines[1], "example.go")
	assert.Contains(t, lines[1], "65.25")
	assert.Contains(t, lines[1], "complexity")
}

func TestWriteFileResultsEmpty(t *testing.T) {
	var files []schema.FileResult

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format:    schema.TextOut,
			Precision: 2,
		},
	}

	var buf bytes.Buffer
	duration := 5 * time.Millisecond
	err := WriteFileResults(&buf, files, cfg.Output, cfg.Runtime, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Showing top 0 files")
	assert.Contains(t, output, "Analysis completed in 5ms")
}
