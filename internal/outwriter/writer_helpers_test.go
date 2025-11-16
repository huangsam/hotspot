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
		data     interface{}
		expected string
	}{
		{
			name: "simple object",
			data: map[string]interface{}{
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
	err := writeCSVWithHeader(&buf, []string{"col"}, func(w *csv.Writer) error {
		// Simulate an error in row writing
		return assert.AnError
	})
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestWriteWithFileStdout(t *testing.T) {
	// Test writing to stdout (empty string means stdout)
	called := false
	err := writeWithFile("", func(w io.Writer) error {
		called = true
		_, err := w.Write([]byte("test"))
		return err
	}, "Test message")
	
	require.NoError(t, err)
	assert.True(t, called, "Writer function should have been called")
}

func TestWriteWithFileActualFile(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// Test writing to an actual file
	testContent := "test content"
	err := writeWithFile(tmpFile, func(w io.Writer) error {
		_, err := w.Write([]byte(testContent))
		return err
	}, "Test message")
	
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestWriteWithFileError(t *testing.T) {
	// Test error propagation from writer function
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	err := writeWithFile(tmpFile, func(w io.Writer) error {
		return assert.AnError
	}, "Test message")
	
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestWriteWithFileInvalidPath(t *testing.T) {
	// Test with an invalid file path (should fail on file open)
	err := writeWithFile("/nonexistent/path/file.txt", func(w io.Writer) error {
		return nil
	}, "Test message")
	
	require.Error(t, err)
}

func TestWriteJSONIntegration(t *testing.T) {
	// Test full integration: write JSON to file using helpers
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.json")

	testData := map[string]interface{}{
		"name":  "integration test",
		"count": 123,
	}

	err := writeWithFile(tmpFile, func(w io.Writer) error {
		return writeJSON(w, testData)
	}, "Wrote JSON")
	
	require.NoError(t, err)

	// Read and verify
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var result map[string]interface{}
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

	err := writeWithFile(tmpFile, func(w io.Writer) error {
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
