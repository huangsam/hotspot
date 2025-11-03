package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlainLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{
			name:     "smallest value possible",
			input:    0.0,
			expected: lowValue,
		},
		{
			name:     "just before moderate",
			input:    39.9,
			expected: lowValue,
		},
		{
			name:     "exactly moderate",
			input:    40.0,
			expected: moderateValue,
		},
		{
			name:     "just before high",
			input:    59.9,
			expected: moderateValue,
		},
		{
			name:     "exactly high",
			input:    60.0,
			expected: highValue,
		},
		{
			name:     "just before critical",
			input:    79.9,
			expected: highValue,
		},
		{
			name:     "exactly critical",
			input:    80.0,
			expected: criticalValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getPlainLabel(tt.input))
		})
	}
}

func TestSelectOutputFile_Fallback(t *testing.T) {
	file, err := selectOutputFile("")
	require.NoError(t, err)
	assert.Equal(t, os.Stdout, file)
}
