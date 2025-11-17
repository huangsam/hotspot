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

func TestWriteTimeseriesResultsTable(t *testing.T) {
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
			{
				Path:   "main.go",
				Period: "30d to 60d Ago",
				Start:  time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
				Score:  70.0,
				Owners: []string{"Alice", "Bob"},
				Mode:   schema.HotMode,
			},
			{
				Path:   "main.go",
				Period: "60d to 90d Ago",
				Start:  time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				Score:  0.0,
				Owners: []string{},
				Mode:   schema.HotMode,
			},
		},
	}

	cfg := &contract.Config{
		Output:    schema.TextOut,
		Precision: 2,
		Detail:    true,
		UseColors: false,
		Width:     120,
	}

	var buf bytes.Buffer
	duration := 100 * time.Millisecond
	err := WriteTimeseriesResults(&buf, result, cfg, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "main.go")
	assert.Contains(t, output, "Current (30d)")
	assert.Contains(t, output, "85.50")
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "30d to 60d Ago")
	assert.Contains(t, output, "70.00")
	assert.Contains(t, output, "Alice, Bob")
	assert.Contains(t, output, "60d to 90d Ago")
	assert.Contains(t, output, "0.00")
	assert.Contains(t, output, "Timeseries analysis completed in 100ms")
}

func TestWriteTimeseriesResultsJSON(t *testing.T) {
	result := schema.TimeseriesResult{
		Points: []schema.TimeseriesPoint{
			{
				Path:   "test.go",
				Period: "Current (30d)",
				Start:  time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
				Score:  75.0,
				Owners: []string{"Bob"},
				Mode:   schema.RiskMode,
			},
		},
	}

	cfg := &contract.Config{
		Output:    schema.JSONOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 50 * time.Millisecond
	err := WriteTimeseriesResults(&buf, result, cfg, duration)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Contains(t, parsed, "points")

	points := parsed["points"].([]any)
	require.Len(t, points, 1)

	point := points[0].(map[string]any)
	assert.Equal(t, "test.go", point["path"])
	assert.Equal(t, "Current (30d)", point["period"])
	assert.Equal(t, 75.0, point["score"])
	assert.Equal(t, []any{"Bob"}, point["owners"])
	assert.Equal(t, "risk", point["mode"])
	assert.Contains(t, point, "start")
	assert.Contains(t, point, "end")
}

func TestWriteTimeseriesResultsCSV(t *testing.T) {
	result := schema.TimeseriesResult{
		Points: []schema.TimeseriesPoint{
			{
				Path:   "example.go",
				Period: "30d to 60d Ago",
				Start:  time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
				End:    time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
				Score:  65.25,
				Owners: []string{"Charlie", "Alice"},
				Mode:   schema.ComplexityMode,
			},
		},
	}

	cfg := &contract.Config{
		Output:    schema.CSVOut,
		Precision: 2,
	}

	var buf bytes.Buffer
	duration := 75 * time.Millisecond
	err := WriteTimeseriesResults(&buf, result, cfg, duration)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	assert.Contains(t, lines[0], "path")
	assert.Contains(t, lines[0], "period")
	assert.Contains(t, lines[0], "score")
	assert.Contains(t, lines[0], "owners")
	assert.Contains(t, lines[0], "mode")
	assert.Contains(t, lines[0], "start")
	assert.Contains(t, lines[0], "end")
	assert.Contains(t, lines[1], "example.go")
	assert.Contains(t, lines[1], "30d to 60d Ago")
	assert.Contains(t, lines[1], "65.25")
	assert.Contains(t, lines[1], "Charlie|Alice")
	assert.Contains(t, lines[1], "complexity")
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
