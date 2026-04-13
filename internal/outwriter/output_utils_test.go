package outwriter

import (
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/config"
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
	cfg := &config.Config{Output: config.OutputConfig{OutputFile: ""}}
	called := false
	err := WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
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
	cfg := &config.Config{Output: config.OutputConfig{OutputFile: tmpFile}}
	err := WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
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

	cfg := &config.Config{Output: config.OutputConfig{OutputFile: tmpFile}}
	err := WriteWithOutputFile(cfg.Output, func(io.Writer) error {
		return assert.AnError
	}, "Test message")

	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestWriteWithOutputFileInvalidPath(t *testing.T) {
	// Test with an invalid file path (should fail on file open)
	cfg := &config.Config{Output: config.OutputConfig{OutputFile: "/nonexistent/path/file.txt"}}
	err := WriteWithOutputFile(cfg.Output, func(io.Writer) error {
		return nil
	}, "Test message")

	require.Error(t, err)
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

	cfg := &config.Config{Output: config.OutputConfig{OutputFile: tmpFile}}
	err := WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
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

func TestWriteFoldersResultsEmpty(t *testing.T) {
	var folders []schema.FolderResult

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format: schema.TextOut,
		},
	}

	var buf bytes.Buffer
	duration := 10 * time.Millisecond
	err := WriteFolderResults(&buf, folders, cfg.Output, cfg.Runtime, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Showing top 0 folders")
}

func TestWriteComparisonResultsEmpty(t *testing.T) {
	comparison := schema.ComparisonResult{
		Details: []schema.ComparisonDetail{},
		Summary: schema.ComparisonSummary{},
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format: schema.TextOut,
		},
	}

	var buf bytes.Buffer
	duration := 15 * time.Millisecond
	err := WriteComparisonResults(&buf, comparison, cfg.Output, cfg.Runtime, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Showing top 0 changes")
}

func TestWriteTimeseriesResultsEmpty(t *testing.T) {
	result := schema.TimeseriesResult{
		Points: []schema.TimeseriesPoint{},
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Format: schema.TextOut,
		},
	}

	var buf bytes.Buffer
	duration := 20 * time.Millisecond
	err := WriteTimeseriesResults(&buf, result, cfg.Output, cfg.Runtime, duration)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Timeseries analysis completed in 20ms")
}

func TestGetColorLabel(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		label string
	}{
		{"low", 30, schema.LowValue},
		{"moderate", 50, schema.ModerateValue},
		{"high", 70, schema.HighValue},
		{"critical", 90, schema.CriticalValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getColorLabel(tt.score)
			// Should contain the plain label
			assert.Contains(t, result, tt.label)
		})
	}
}

func TestSelectOutputFile(t *testing.T) {
	t.Run("empty path returns stdout", func(t *testing.T) {
		file, err := selectOutputFile("")
		require.NoError(t, err)
		assert.Equal(t, os.Stdout, file)
	})

	t.Run("valid path creates file", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "test_output.txt")
		file, err := selectOutputFile(tempFile)
		require.NoError(t, err)
		assert.NotNil(t, file)
		_ = file.Close()

		// Verify file was created
		_, err = os.Stat(tempFile)
		assert.NoError(t, err)
	})
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		maxLen   int
		expected string
	}{
		{
			name:     "short path no truncation",
			path:     "src/main.go",
			maxLen:   20,
			expected: "src/main.go",
		},
		{
			name:     "exactly max length no truncation",
			path:     "pkg/utils/helper.go",
			maxLen:   19,
			expected: "pkg/utils/helper.go",
		},
		{
			name:     "truncate from start",
			path:     "cmd/hotspot/commands/analysis/export.go",
			maxLen:   20,
			expected: "...nalysis/export.go",
		},
		{
			name:     "very long path with minimal maxLen",
			path:     "some/very/long/path/to/a/file/deep/in/the/repo/structure.go",
			maxLen:   15,
			expected: "...structure.go",
		},
		{
			name:     "maxLen too short for ellipses",
			path:     "long/path/to/file.go",
			maxLen:   5,
			expected: "...go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, truncatePath(tt.path, tt.maxLen))
		})
	}
}
