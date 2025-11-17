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

func TestWriteComparisonResultsTable(t *testing.T) {
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
			{
				Path:         "utils.go",
				BeforeScore:  60.0,
				AfterScore:   50.0,
				Delta:        -10.0,
				DeltaCommits: -2,
				DeltaChurn:   -50,
				Status:       schema.InactiveStatus,
				BeforeOwners: []string{"Charlie"},
				AfterOwners:  []string{"Charlie"},
				Mode:         schema.RiskMode,
			},
		},
		Summary: schema.ComparisonSummary{
			NetScoreDelta:         0.0,
			NetChurnDelta:         50,
			TotalNewFiles:         0,
			TotalInactiveFiles:    1,
			TotalModifiedFiles:    2,
			TotalOwnershipChanges: 1,
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
	err := WriteComparisonResults(&buf, comparison, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "main.go")
	assert.Contains(t, output, "70.00")
	assert.Contains(t, output, "80.00")
	assert.Contains(t, output, "+10.00")
	assert.Contains(t, output, "active")
	assert.Contains(t, output, "100")
	assert.Contains(t, output, "Alice, Bob")
	assert.Contains(t, output, "utils.go")
	assert.Contains(t, output, "60.00")
	assert.Contains(t, output, "50.00")
	assert.Contains(t, output, "-10.00")
	assert.Contains(t, output, "inactive")
	assert.Contains(t, output, "-50")
	assert.Contains(t, output, "Removed: Charlie")
	assert.Contains(t, output, "Net score delta: 0.00")
	assert.Contains(t, output, "Net churn delta: 50")
	assert.Contains(t, output, "New files: 0")
	assert.Contains(t, output, "Inactive files: 1")
	assert.Contains(t, output, "Modified files: 2")
	assert.Contains(t, output, "Ownership changes: 1")
}

func TestWriteComparisonResultsJSON(t *testing.T) {
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
				BeforeOwners: []string{"Alice"},
				AfterOwners:  []string{"Alice"},
				Mode:         schema.RiskMode,
			},
		},
		Summary: schema.ComparisonSummary{
			NetScoreDelta:         10.0,
			NetChurnDelta:         50,
			TotalNewFiles:         0,
			TotalInactiveFiles:    0,
			TotalModifiedFiles:    1,
			TotalOwnershipChanges: 0,
		},
	}

	cfg := &contract.Config{
		Output:    schema.JSONOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 50 * time.Millisecond
	err := WriteComparisonResults(&buf, comparison, cfg, duration)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "details")
	assert.Contains(t, result, "summary")

	details := result["details"].([]any)
	require.Len(t, details, 1)

	detail := details[0].(map[string]any)
	assert.Equal(t, "test.go", detail["path"])
	assert.Equal(t, 50.0, detail["before_score"])
	assert.Equal(t, 60.0, detail["after_score"])
	assert.Equal(t, 10.0, detail["delta"])
	assert.Equal(t, float64(3), detail["delta_commits"])
	assert.Equal(t, float64(50), detail["delta_churn"])
	assert.Equal(t, "active", detail["status"])
	assert.Equal(t, []any{"Alice"}, detail["before_owners"])
	assert.Equal(t, []any{"Alice"}, detail["after_owners"])
	assert.Equal(t, "risk", detail["mode"])

	summary := result["summary"].(map[string]any)
	assert.Equal(t, 10.0, summary["net_score_delta"])
	assert.Equal(t, float64(50), summary["net_churn_delta"])
	assert.Equal(t, float64(0), summary["total_new_files"])
	assert.Equal(t, float64(0), summary["total_inactive_files"])
	assert.Equal(t, float64(1), summary["total_modified_files"])
	assert.Equal(t, float64(0), summary["total_ownership_changes"])
}

func TestWriteComparisonResultsCSV(t *testing.T) {
	comparison := schema.ComparisonResult{
		Results: []schema.ComparisonDetails{
			{
				Path:         "example.go",
				BeforeScore:  40.0,
				AfterScore:   45.0,
				Delta:        5.0,
				DeltaCommits: 1,
				DeltaChurn:   25,
				Status:       schema.NewStatus,
				BeforeOwners: []string{},
				AfterOwners:  []string{"Bob"},
				Mode:         schema.ComplexityMode,
			},
		},
		Summary: schema.ComparisonSummary{
			NetScoreDelta:         5.0,
			NetChurnDelta:         25,
			TotalNewFiles:         1,
			TotalInactiveFiles:    0,
			TotalModifiedFiles:    0,
			TotalOwnershipChanges: 1,
		},
	}

	cfg := &contract.Config{
		Output:    schema.CSVOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 75 * time.Millisecond
	err := WriteComparisonResults(&buf, comparison, cfg, duration)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "path")
	assert.Contains(t, lines[0], "base_score")
	assert.Contains(t, lines[0], "comp_score")
	assert.Contains(t, lines[0], "delta_score")
	assert.Contains(t, lines[0], "delta_commits")
	assert.Contains(t, lines[0], "delta_churn")
	assert.Contains(t, lines[0], "before_owners")
	assert.Contains(t, lines[0], "after_owners")
	assert.Contains(t, lines[0], "mode")
	assert.Contains(t, lines[1], "example.go")
	assert.Contains(t, lines[1], "40.00")
	assert.Contains(t, lines[1], "45.00")
	assert.Contains(t, lines[1], "5.00")
	assert.Contains(t, lines[1], "1")
	assert.Contains(t, lines[1], "25")
	assert.Contains(t, lines[1], ",Bob,")
	assert.Contains(t, lines[1], "complexity")
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
