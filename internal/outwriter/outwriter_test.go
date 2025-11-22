package outwriter

import (
	"bytes"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutWriter_WriteFiles(t *testing.T) {
	ow := NewOutWriter()
	files := []schema.FileResult{
		{
			Path:      "file.go",
			ModeScore: 90.0,
			Mode:      schema.HotMode,
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
