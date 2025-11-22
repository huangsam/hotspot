package core

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

// TestRankFiles tests file ranking logic.
func TestRankFiles(t *testing.T) {
	files := []schema.FileResult{
		{Path: "low.go", SizeBytes: 1, ModeScore: 10, Mode: schema.HotMode},
		{Path: "high.go", SizeBytes: 1, ModeScore: 90, Mode: schema.HotMode},
		{Path: "medium.go", SizeBytes: 1, ModeScore: 50, Mode: schema.HotMode},
		{Path: "critical.go", SizeBytes: 1, ModeScore: 95, Mode: schema.HotMode},
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
			assert.LessOrEqual(t, ranked[i].ModeScore, ranked[i-1].ModeScore)
		}
	})
}
