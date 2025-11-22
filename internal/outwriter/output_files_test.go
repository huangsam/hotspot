package outwriter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSONResults(t *testing.T) {
	files := []schema.FileResult{
		{
			Path:               "test.go",
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

	var buf bytes.Buffer
	err := writeJSONResultsForFiles(&buf, files)
	require.NoError(t, err)

	// Parse the JSON to verify structure
	var result []map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, float64(1), result[0]["rank"])
	assert.Equal(t, "test.go", result[0]["path"])
	assert.Equal(t, 85.5, result[0]["score"])
	assert.Equal(t, "Critical", result[0]["label"])
}

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

func TestWriteJSONResultsMultipleFiles(t *testing.T) {
	files := []schema.FileResult{
		{
			Path:      "file1.go",
			ModeScore: 90.0,
			Mode:      schema.HotMode,
		},
		{
			Path:      "file2.go",
			ModeScore: 85.0,
			Mode:      schema.RiskMode,
		},
		{
			Path:      "file3.go",
			ModeScore: 80.0,
			Mode:      schema.ComplexityMode,
		},
	}

	var buf bytes.Buffer
	err := writeJSONResultsForFiles(&buf, files)
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Verify ranks are sequential
	assert.Equal(t, float64(1), result[0]["rank"])
	assert.Equal(t, float64(2), result[1]["rank"])
	assert.Equal(t, float64(3), result[2]["rank"])

	// Verify labels are computed correctly
	assert.Equal(t, "Critical", result[0]["label"]) // 90.0 is critical
	assert.Equal(t, "Critical", result[1]["label"]) // 85.0 is critical
	assert.Equal(t, "Critical", result[2]["label"]) // 80.0 is critical
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

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
		Detail:    true,
		Explain:   true,
		Owner:     true,
		UseColors: false,
		Width:     120,
	}

	var buf bytes.Buffer
	duration := 100 * time.Millisecond
	err := WriteFileResults(&buf, files, cfg, duration)
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

func TestWriteFileResultsJSON(t *testing.T) {
	files := []schema.FileResult{
		{
			Path:      "test.go",
			ModeScore: 75.0,
			Mode:      schema.RiskMode,
		},
	}

	cfg := &contract.Config{
		Output:    schema.JSONOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 50 * time.Millisecond
	err := WriteFileResults(&buf, files, cfg, duration)
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, "test.go", result[0]["path"])
	assert.Equal(t, 75.0, result[0]["score"])
	assert.Equal(t, "High", result[0]["label"])
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

	cfg := &contract.Config{
		Output:    schema.CSVOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 75 * time.Millisecond
	err := WriteFileResults(&buf, files, cfg, duration)
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

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 5 * time.Millisecond
	err := WriteFileResults(&buf, files, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Showing top 0 files")
	assert.Contains(t, output, "Analysis completed in 5ms")
}
