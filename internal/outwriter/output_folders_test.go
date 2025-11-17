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

func TestWriteFoldersResultsTable(t *testing.T) {
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
		{
			Path:     "pkg/",
			Score:    75.5,
			Commits:  15,
			Churn:    750,
			TotalLOC: 3000,
			Owners:   []string{"Charlie"},
			Mode:     schema.RiskMode,
		},
	}

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
		Detail:    true,
		Owner:     true,
		UseColors: false,
		Width:     120,
	}

	var buf bytes.Buffer
	duration := 100 * time.Millisecond
	err := WriteFolderResults(&buf, folders, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "src/")
	assert.Contains(t, output, "90.00")
	assert.Contains(t, output, "20")
	assert.Contains(t, output, "1000")
	assert.Contains(t, output, "5000")
	assert.Contains(t, output, "Alice, Bob")
	assert.Contains(t, output, "pkg/")
	assert.Contains(t, output, "75.50")
	assert.Contains(t, output, "Charlie")
	assert.Contains(t, output, "Analysis completed in 100ms")
}

func TestWriteFoldersResultsJSON(t *testing.T) {
	folders := []schema.FolderResult{
		{
			Path:     "test/",
			Score:    85.0,
			Commits:  10,
			Churn:    500,
			TotalLOC: 2500,
			Owners:   []string{"Alice"},
			Mode:     schema.ComplexityMode,
		},
	}

	cfg := &contract.Config{
		Output:    schema.JSONOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 50 * time.Millisecond
	err := WriteFolderResults(&buf, folders, cfg, duration)
	require.NoError(t, err)

	var result []map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, "test/", result[0]["path"])
	assert.Equal(t, 85.0, result[0]["score"])
	assert.Equal(t, "Critical", result[0]["label"])
	assert.Equal(t, float64(10), result[0]["commits"])
	assert.Equal(t, float64(500), result[0]["churn"])
	assert.Equal(t, float64(2500), result[0]["total_loc"])
	assert.Equal(t, []any{"Alice"}, result[0]["owners"])
	assert.Equal(t, "complexity", result[0]["mode"])
}

func TestWriteFoldersResultsCSV(t *testing.T) {
	folders := []schema.FolderResult{
		{
			Path:     "example/",
			Score:    65.25,
			Commits:  8,
			Churn:    400,
			TotalLOC: 2000,
			Owners:   []string{"Bob", "Charlie"},
			Mode:     schema.StaleMode,
		},
	}

	cfg := &contract.Config{
		Output:    schema.CSVOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 75 * time.Millisecond
	err := WriteFolderResults(&buf, folders, cfg, duration)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "folder")
	assert.Contains(t, lines[0], "score")
	assert.Contains(t, lines[0], "label")
	assert.Contains(t, lines[0], "total_commits")
	assert.Contains(t, lines[0], "total_churn")
	assert.Contains(t, lines[0], "total_loc")
	assert.Contains(t, lines[0], "owner")
	assert.Contains(t, lines[0], "mode")
	assert.Contains(t, lines[1], "example/")
	assert.Contains(t, lines[1], "65.25")
	assert.Contains(t, lines[1], "8")
	assert.Contains(t, lines[1], "400")
	assert.Contains(t, lines[1], "2000")
	assert.Contains(t, lines[1], "Bob|Charlie")
	assert.Contains(t, lines[1], "stale")
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
