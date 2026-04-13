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

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format:    schema.TextOut,
			Precision: 2,
			Detail:    true,
			Owner:     true,
			UseColors: false,
			Width:     120,
		},
	}

	var buf bytes.Buffer
	duration := 100 * time.Millisecond
	err := WriteFolderResults(&buf, folders, cfg.Output, cfg.Runtime, duration)
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

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format:    schema.CSVOut,
			Precision: 2,
		},
	}

	var buf bytes.Buffer
	duration := 75 * time.Millisecond
	err := WriteFolderResults(&buf, folders, cfg.Output, cfg.Runtime, duration)
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
