package schema_test

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestGetPlainLabel(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{"Critical Score Upper", 100.0, "Critical"},
		{"Critical Score Lower", 80.0, "Critical"},
		{"High Score Upper", 79.9, "High"},
		{"High Score Lower", 60.0, "High"},
		{"Moderate Score Upper", 59.9, "Moderate"},
		{"Moderate Score Lower", 40.0, "Moderate"},
		{"Low Score Upper", 39.9, "Low"},
		{"Low Score Lower", 0.0, "Low"},
		{"Negative Score", -10.0, "Low"}, // Edge case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schema.GetPlainLabel(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnrichFiles(t *testing.T) {
	files := []schema.FileResult{
		{Path: "file1.go", ModeScore: 85.0}, // Critical
		{Path: "file2.go", ModeScore: 65.0}, // High
		{Path: "file3.go", ModeScore: 20.0}, // Low
	}

	enriched := schema.EnrichFiles(files)

	assert.Len(t, enriched, 3)

	assert.Equal(t, 1, enriched[0].Rank)
	assert.Equal(t, "Critical", enriched[0].Label)
	assert.Equal(t, "file1.go", enriched[0].Path)

	assert.Equal(t, 2, enriched[1].Rank)
	assert.Equal(t, "High", enriched[1].Label)
	assert.Equal(t, "file2.go", enriched[1].Path)

	assert.Equal(t, 3, enriched[2].Rank)
	assert.Equal(t, "Low", enriched[2].Label)
	assert.Equal(t, "file3.go", enriched[2].Path)
}

func TestEnrichFolders(t *testing.T) {
	folders := []schema.FolderResult{
		{Path: "folder1", Score: 45.0}, // Moderate
		{Path: "folder2", Score: 80.0}, // Critical
	}

	enriched := schema.EnrichFolders(folders)

	assert.Len(t, enriched, 2)

	assert.Equal(t, 1, enriched[0].Rank)
	assert.Equal(t, "Moderate", enriched[0].Label)
	assert.Equal(t, "folder1", enriched[0].Path)

	assert.Equal(t, 2, enriched[1].Rank)
	assert.Equal(t, "Critical", enriched[1].Label)
	assert.Equal(t, "folder2", enriched[1].Path)
}
