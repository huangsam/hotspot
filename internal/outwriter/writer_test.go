package outwriter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	err := writeJSONResults(&buf, files)
	require.NoError(t, err)

	// Parse the JSON to verify structure
	var result []map[string]interface{}
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
	err := writeCSVResults(w, files, fmtFloat, intFmt)
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

	var result []map[string]interface{}
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

	var result map[string]interface{}
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

	var parsed map[string]interface{}
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

	var result map[string]interface{}
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
	files := []schema.FileResult{}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	err := writeCSVResults(w, files, fmtFloat, intFmt)
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
	err := writeJSONResults(&buf, files)
	require.NoError(t, err)

	var result []map[string]interface{}
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
