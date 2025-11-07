package core

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

// TestRankFiles tests file ranking logic.
func TestRankFiles(t *testing.T) {
	files := []schema.FileResult{
		{Path: "low.go", SizeBytes: 1, Score: 10},
		{Path: "high.go", SizeBytes: 1, Score: 90},
		{Path: "medium.go", SizeBytes: 1, Score: 50},
		{Path: "critical.go", SizeBytes: 1, Score: 95},
	}

	t.Run("rank and limit", func(t *testing.T) {
		ranked := rankFiles(files, 2)
		assert.Equal(t, 2, len(ranked))
		assert.Equal(t, "critical.go", ranked[0].Path)
		assert.Equal(t, "high.go", ranked[1].Path)
	})

	t.Run("limit exceeds length", func(t *testing.T) {
		ranked := rankFiles(files, 10)
		assert.Equal(t, 4, len(ranked))
	})

	t.Run("scores in descending order", func(t *testing.T) {
		ranked := rankFiles(files, 10)
		for i := 1; i < len(ranked); i++ {
			assert.LessOrEqual(t, ranked[i].Score, ranked[i-1].Score)
		}
	})
}
