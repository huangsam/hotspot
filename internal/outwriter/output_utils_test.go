package outwriter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFormatters(t *testing.T) {
	tests := []struct {
		name      string
		precision int
		value     float64
		expected  string
	}{
		{
			name:      "precision 2",
			precision: 2,
			value:     3.14159,
			expected:  "3.14",
		},
		{
			name:      "precision 0",
			precision: 0,
			value:     3.14159,
			expected:  "3",
		},
		{
			name:      "precision 4",
			precision: 4,
			value:     3.14159,
			expected:  "3.1416",
		},
		{
			name:      "negative value",
			precision: 2,
			value:     -42.567,
			expected:  "-42.57",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmtFloat, intFmt := createFormatters(tt.precision)
			assert.Equal(t, tt.expected, fmtFloat(tt.value))
			assert.Equal(t, "%d", intFmt)
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected string
	}{
		{
			name: "simple object",
			data: map[string]any{
				"name":  "test",
				"value": 42,
			},
			expected: `{
  "name": "test",
  "value": 42
}
`,
		},
		{
			name: "array",
			data: []string{"a", "b", "c"},
			expected: `[
  "a",
  "b",
  "c"
]
`,
		},
		{
			name:     "string",
			data:     "hello",
			expected: `"hello"` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := writeJSON(&buf, tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestWriteJSONError(t *testing.T) {
	// Test with a value that can't be marshaled to JSON
	invalidData := make(chan int)
	var buf bytes.Buffer
	err := writeJSON(&buf, invalidData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to encode JSON")
}

func TestWriteCSVWithHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   []string
		rows     [][]string
		expected string
	}{
		{
			name:   "simple csv",
			header: []string{"name", "age", "city"},
			rows: [][]string{
				{"Alice", "30", "NYC"},
				{"Bob", "25", "LA"},
			},
			expected: "name,age,city\nAlice,30,NYC\nBob,25,LA\n",
		},
		{
			name:     "empty rows",
			header:   []string{"col1", "col2"},
			rows:     [][]string{},
			expected: "col1,col2\n",
		},
		{
			name:   "values with commas",
			header: []string{"name", "description"},
			rows: [][]string{
				{"Test", "A value, with comma"},
			},
			expected: "name,description\nTest,\"A value, with comma\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := writeCSVWithHeader(&buf, tt.header, func(w *csv.Writer) error {
				for _, row := range tt.rows {
					if err := w.Write(row); err != nil {
						return err
					}
				}
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestWriteCSVWithHeaderError(t *testing.T) {
	// Test CSV writer error propagation
	var buf bytes.Buffer
	err := writeCSVWithHeader(&buf, []string{"col"}, func(*csv.Writer) error {
		// Simulate an error in row writing
		return assert.AnError
	})
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestWriteWithOutputFileStdout(t *testing.T) {
	// Test writing to stdout (empty string means stdout)
	cfg := &contract.Config{UseEmojis: false, OutputFile: ""}
	called := false
	err := WriteWithOutputFile(cfg, func(w io.Writer) error {
		called = true
		_, err := w.Write([]byte("test"))
		return err
	}, "Test message")

	require.NoError(t, err)
	assert.True(t, called, "Writer function should have been called")
}

func TestWriteWithOutputFileActualFile(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// Test writing to an actual file
	testContent := "test content"
	cfg := &contract.Config{UseEmojis: false, OutputFile: tmpFile}
	err := WriteWithOutputFile(cfg, func(w io.Writer) error {
		_, err := w.Write([]byte(testContent))
		return err
	}, "Test message")

	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestWriteWithOutputFileError(t *testing.T) {
	// Test error propagation from writer function
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	cfg := &contract.Config{UseEmojis: false, OutputFile: tmpFile}
	err := WriteWithOutputFile(cfg, func(io.Writer) error {
		return assert.AnError
	}, "Test message")

	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestWriteWithOutputFileInvalidPath(t *testing.T) {
	// Test with an invalid file path (should fail on file open)
	cfg := &contract.Config{UseEmojis: false, OutputFile: "/nonexistent/path/file.txt"}
	err := WriteWithOutputFile(cfg, func(io.Writer) error {
		return nil
	}, "Test message")

	require.Error(t, err)
}

func TestWriteJSONIntegration(t *testing.T) {
	// Test full integration: write JSON to file using helpers
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.json")

	testData := map[string]any{
		"name":  "integration test",
		"count": 123,
	}

	cfg := &contract.Config{UseEmojis: false, OutputFile: tmpFile}
	err := WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writeJSON(w, testData)
	}, "Wrote JSON")

	require.NoError(t, err)

	// Read and verify
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(content, &result)
	require.NoError(t, err)

	assert.Equal(t, "integration test", result["name"])
	assert.Equal(t, float64(123), result["count"]) // JSON numbers are float64
}

func TestWriteCSVIntegration(t *testing.T) {
	// Test full integration: write CSV to file using helpers
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")

	header := []string{"name", "score"}
	rows := [][]string{
		{"Alice", "95"},
		{"Bob", "87"},
	}

	cfg := &contract.Config{UseEmojis: false, OutputFile: tmpFile}
	err := WriteWithOutputFile(cfg, func(w io.Writer) error {
		return writeCSVWithHeader(w, header, func(csvWriter *csv.Writer) error {
			for _, row := range rows {
				if err := csvWriter.Write(row); err != nil {
					return err
				}
			}
			return nil
		})
	}, "Wrote CSV")

	require.NoError(t, err)

	// Read and verify
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Equal(t, 3, len(lines)) // header + 2 rows
	assert.Equal(t, "name,score", lines[0])
	assert.Equal(t, "Alice,95", lines[1])
	assert.Equal(t, "Bob,87", lines[2])
}

func TestWriteJSONResults(t *testing.T) {
	files := []schema.FileResult{
		{
			Path:               "test.go",
			Score:              85.5,
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
			Score:              75.25,
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

func TestWriteJSONResultsForFolders(t *testing.T) {
	folders := []schema.FolderResult{
		{
			Path:     "src/",
			Score:    90.0,
			Commits:  20,
			Churn:    1000,
			TotalLOC: 5000,
			Owners:   []string{"Alice", "Bob"},
			Mode:     schema.HotMode,
		},
	}

	var buf bytes.Buffer
	err := writeJSONResultsForFolders(&buf, folders)
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, float64(1), result[0]["rank"])
	assert.Equal(t, "src/", result[0]["path"])
	assert.Equal(t, 90.0, result[0]["score"])
}

func TestWriteCSVResultsForFolders(t *testing.T) {
	fmtFloat, intFmt := createFormatters(2)
	folders := []schema.FolderResult{
		{
			Path:     "pkg/",
			Score:    65.5,
			Commits:  15,
			Churn:    750,
			TotalLOC: 3000,
			Owners:   []string{"Charlie"},
			Mode:     schema.ComplexityMode,
		},
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := writeCSVResultsForFolders(w, folders, fmtFloat, intFmt)
	require.NoError(t, err)
	w.Flush()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "folder")
	assert.Contains(t, lines[0], "total_commits")
	assert.Contains(t, lines[1], "pkg/")
	assert.Contains(t, lines[1], "65.5")
}

func TestWriteJSONResultsForComparison(t *testing.T) {
	comparison := schema.ComparisonResult{
		Results: []schema.ComparisonDetails{
			{
				Path:         "main.go",
				BeforeScore:  70.0,
				AfterScore:   80.0,
				Delta:        10.0,
				DeltaCommits: 5,
				DeltaChurn:   100,
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{"Alice", "Bob"},
				Mode:         schema.HotMode,
			},
		},
		Summary: schema.ComparisonSummary{
			NetScoreDelta:         10.0,
			NetChurnDelta:         100,
			TotalNewFiles:         0,
			TotalInactiveFiles:    0,
			TotalModifiedFiles:    1,
			TotalOwnershipChanges: 1,
		},
	}

	var buf bytes.Buffer
	err := writeJSONResultsForComparison(&buf, comparison)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "details")
	assert.Contains(t, result, "summary")
}

func TestWriteCSVResultsForComparison(t *testing.T) {
	fmtFloat, intFmt := createFormatters(2)
	comparison := schema.ComparisonResult{
		Results: []schema.ComparisonDetails{
			{
				Path:         "test.go",
				BeforeScore:  50.0,
				AfterScore:   60.0,
				Delta:        10.0,
				DeltaCommits: 3,
				DeltaChurn:   50,
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Charlie"},
				AfterOwners:  []string{"Charlie"},
				Mode:         schema.RiskMode,
			},
		},
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := writeCSVResultsForComparison(w, comparison, fmtFloat, intFmt)
	require.NoError(t, err)
	w.Flush()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "delta_score")
	assert.Contains(t, lines[1], "test.go")
	assert.Contains(t, lines[1], "10.00")
}

func TestWriteJSONResultsForTimeseries(t *testing.T) {
	result := schema.TimeseriesResult{
		Points: []schema.TimeseriesPoint{
			{
				Path:   "main.go",
				Period: "Current (30d)",
				Start:  time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
				Score:  85.5,
				Owners: []string{"Alice"},
				Mode:   schema.HotMode,
			},
		},
	}

	var buf bytes.Buffer
	err := writeJSONResultsForTimeseries(&buf, result)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Contains(t, parsed, "points")
}

func TestWriteCSVResultsForTimeseries(t *testing.T) {
	fmtFloat, _ := createFormatters(2)
	result := schema.TimeseriesResult{
		Points: []schema.TimeseriesPoint{
			{
				Path:   "test.go",
				Period: "30d to 60d Ago",
				Start:  time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
				Score:  70.0,
				Owners: []string{"Bob"},
				Mode:   schema.RiskMode,
			},
		},
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := writeCSVResultsForTimeseries(w, result, fmtFloat)
	require.NoError(t, err)
	w.Flush()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "period")
	assert.Contains(t, lines[1], "30d to 60d Ago")
	assert.Contains(t, lines[1], "70.00")
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
			Path:  "file1.go",
			Score: 90.0,
			Mode:  schema.HotMode,
		},
		{
			Path:  "file2.go",
			Score: 85.0,
			Mode:  schema.RiskMode,
		},
		{
			Path:  "file3.go",
			Score: 80.0,
			Mode:  schema.ComplexityMode,
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

func TestGetDisplayNameForMode(t *testing.T) {
	tests := []struct {
		name     string
		modeName string
		expected string
	}{
		{
			name:     "hot mode",
			modeName: "hot",
			expected: "Hot",
		},
		{
			name:     "risk mode",
			modeName: "risk",
			expected: "Risk",
		},
		{
			name:     "complexity mode",
			modeName: "complexity",
			expected: "Complexity",
		},
		{
			name:     "stale mode",
			modeName: "stale",
			expected: "Stale",
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

func TestWriteFileResultsTable(t *testing.T) {
	files := []schema.FileResult{
		{
			Path:               "main.go",
			Score:              85.5,
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
			Path:  "test.go",
			Score: 75.0,
			Mode:  schema.RiskMode,
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
			Score:              65.25,
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

func TestOutWriter_WriteFiles(t *testing.T) {
	ow := NewOutWriter()
	files := []schema.FileResult{
		{
			Path:  "file.go",
			Score: 90.0,
			Mode:  schema.HotMode,
		},
	}

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 10 * time.Millisecond
	err := ow.WriteFiles(&buf, files, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "file.go")
	assert.Contains(t, output, "90.00")
}

func TestOutWriter_WriteFolders(t *testing.T) {
	ow := NewOutWriter()
	folders := []schema.FolderResult{
		{
			Path:     "src/",
			Score:    80.0,
			Commits:  20,
			Churn:    1000,
			TotalLOC: 5000,
			Owners:   []string{"Alice"},
			Mode:     schema.HotMode,
		},
	}

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
		Detail:    true,
	}

	var buf bytes.Buffer
	duration := 20 * time.Millisecond
	err := ow.WriteFolders(&buf, folders, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "src/")
	assert.Contains(t, output, "80.00")
	assert.Contains(t, output, "20")
	assert.Contains(t, output, "1000")
	assert.Contains(t, output, "5000")
}

func TestOutWriter_WriteComparison(t *testing.T) {
	ow := NewOutWriter()
	comparison := schema.ComparisonResult{
		Results: []schema.ComparisonDetails{
			{
				Path:         "main.go",
				BeforeScore:  70.0,
				AfterScore:   80.0,
				Delta:        10.0,
				DeltaCommits: 5,
				DeltaChurn:   100,
				Status:       schema.ActiveStatus,
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{"Alice", "Bob"},
				Mode:         schema.HotMode,
			},
		},
		Summary: schema.ComparisonSummary{
			NetScoreDelta:         10.0,
			NetChurnDelta:         100,
			TotalNewFiles:         0,
			TotalInactiveFiles:    0,
			TotalModifiedFiles:    1,
			TotalOwnershipChanges: 1,
		},
	}

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
		Detail:    true,
		Owner:     true,
	}

	var buf bytes.Buffer
	duration := 30 * time.Millisecond
	err := ow.WriteComparison(&buf, comparison, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "main.go")
	assert.Contains(t, output, "70.00")
	assert.Contains(t, output, "80.00")
	assert.Contains(t, output, "+10.00")
	assert.Contains(t, output, "Net score delta: 10.00")
}

func TestOutWriter_WriteTimeseries(t *testing.T) {
	ow := NewOutWriter()
	result := schema.TimeseriesResult{
		Points: []schema.TimeseriesPoint{
			{
				Path:   "main.go",
				Period: "Current (30d)",
				Start:  time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
				Score:  85.5,
				Owners: []string{"Alice"},
				Mode:   schema.HotMode,
			},
		},
	}

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 40 * time.Millisecond
	err := ow.WriteTimeseries(&buf, result, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "main.go")
	assert.Contains(t, output, "Current (30d)")
	assert.Contains(t, output, "85.50")
	assert.Contains(t, output, "Alice")
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

func TestWriteFoldersResultsEmpty(t *testing.T) {
	var folders []schema.FolderResult

	cfg := &contract.Config{
		Output: schema.TextOut,
	}

	var buf bytes.Buffer
	duration := 10 * time.Millisecond
	err := WriteFolderResults(&buf, folders, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Showing top 0 folders")
}

func TestWriteComparisonResultsEmpty(t *testing.T) {
	comparison := schema.ComparisonResult{
		Results: []schema.ComparisonDetails{},
		Summary: schema.ComparisonSummary{},
	}

	cfg := &contract.Config{
		Output: schema.TextOut,
	}

	var buf bytes.Buffer
	duration := 15 * time.Millisecond
	err := WriteComparisonResults(&buf, comparison, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Showing top 0 changes")
}

func TestWriteTimeseriesResultsEmpty(t *testing.T) {
	result := schema.TimeseriesResult{
		Points: []schema.TimeseriesPoint{},
	}

	cfg := &contract.Config{
		Output: schema.TextOut,
	}

	var buf bytes.Buffer
	duration := 20 * time.Millisecond
	err := WriteTimeseriesResults(&buf, result, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Timeseries analysis completed in 20ms")
}

func TestOutWriter_WriteMetrics(t *testing.T) {
	ow := NewOutWriter()
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.HotMode: {
			schema.BreakdownCommits: 0.5,
			schema.BreakdownChurn:   0.5,
		},
	}

	cfg := &contract.Config{
		Output: schema.TextOut,
	}

	var buf bytes.Buffer
	err := ow.WriteMetrics(&buf, activeWeights, cfg)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Hotspot Scoring Modes")
	assert.Contains(t, output, "Activity hotspots")
	assert.Contains(t, output, "Knowledge risk")
	assert.Contains(t, output, "Technical debt")
	assert.Contains(t, output, "Maintenance debt")
}

func TestWriteMetricsDefinitions_Text(t *testing.T) {
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.HotMode: {
			schema.BreakdownCommits: 0.6,
			schema.BreakdownChurn:   0.4,
		},
	}

	cfg := &contract.Config{
		Output: schema.TextOut,
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Hotspot Scoring Modes")
	assert.Contains(t, output, "Activity hotspots")
	assert.Contains(t, output, "Factors:")
	assert.Contains(t, output, "Formula:")
}

func TestWriteMetricsDefinitions_JSON(t *testing.T) {
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.RiskMode: {
			schema.BreakdownInvContrib: 0.7,
			schema.BreakdownGini:       0.3,
		},
	}

	cfg := &contract.Config{
		Output: schema.JSONOut,
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"title"`)
	assert.Contains(t, output, `"modes"`)
	assert.Contains(t, output, `"hot"`)
	assert.Contains(t, output, `"risk"`)
	assert.Contains(t, output, `"complexity"`)
	assert.Contains(t, output, `"stale"`)

	// Verify it's valid JSON
	var result map[string]any
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
}

func TestWriteMetricsDefinitions_CSV(t *testing.T) {
	activeWeights := map[schema.ScoringMode]map[schema.BreakdownKey]float64{
		schema.ComplexityMode: {
			schema.BreakdownSize: 0.5,
			schema.BreakdownAge:  0.5,
		},
	}

	cfg := &contract.Config{
		Output: schema.CSVOut,
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Greater(t, len(lines), 1)

	// Check header
	assert.Contains(t, lines[0], "Mode")
	assert.Contains(t, lines[0], "Purpose")
	assert.Contains(t, lines[0], "Factors")
	assert.Contains(t, lines[0], "Formula")

	// Check that all modes are present
	foundModes := make(map[string]bool)
	for i := 1; i < len(lines); i++ {
		parts := strings.Split(lines[i], ",")
		if len(parts) >= 1 {
			foundModes[parts[0]] = true
		}
	}

	assert.True(t, foundModes["hot"])
	assert.True(t, foundModes["risk"])
	assert.True(t, foundModes["complexity"])
	assert.True(t, foundModes["stale"])
}

func TestWriteMetricsDefinitions_EmptyWeights(t *testing.T) {
	// Test with nil active weights (should use defaults)
	var activeWeights map[schema.ScoringMode]map[schema.BreakdownKey]float64

	cfg := &contract.Config{
		Output: schema.TextOut,
	}

	var buf bytes.Buffer
	err := WriteMetricsDefinitions(&buf, activeWeights, cfg)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Hotspot Scoring Modes")
	assert.Contains(t, output, "Activity hotspots")
	assert.Contains(t, output, "Knowledge risk")
}
