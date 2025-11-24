package core

import (
	"testing"

	"github.com/huangsam/hotspot/core/agg"
	"github.com/huangsam/hotspot/core/algo"
	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateAndScoreFolders(t *testing.T) {
	// Create test file results
	fileResults := []schema.FileResult{
		{
			Path:        "src/main.go",
			Commits:     10,
			Churn:       50,
			LinesOfCode: 100,
			ModeScore:   25.0,
			Owners:      []string{"Alice"},
		},
		{
			Path:        "src/utils.go",
			Commits:     5,
			Churn:       25,
			LinesOfCode: 50,
			ModeScore:   15.0,
			Owners:      []string{"Bob"},
		},
		{
			Path:        "tests/main_test.go",
			Commits:     8,
			Churn:       40,
			LinesOfCode: 80,
			ModeScore:   20.0,
			Owners:      []string{"Alice"},
		},
		{
			Path:        "README.md",
			Commits:     3,
			Churn:       15,
			LinesOfCode: 30,
			ModeScore:   10.0,
			Owners:      []string{"Charlie"},
		},
	}

	cfg := &contract.Config{
		Mode: schema.HotMode,
	}

	result := agg.AggregateAndScoreFolders(cfg, fileResults)

	// Should have 2 folders: src/, tests/ (root is skipped when PathFilter is empty)
	assert.Len(t, result, 2)

	// Find folders by path
	var srcFolder, testsFolder *schema.FolderResult
	for i := range result {
		switch result[i].Path {
		case "src":
			srcFolder = &result[i]
		case "tests":
			testsFolder = &result[i]
		}
	}

	// Check src/ folder
	require.NotNil(t, srcFolder)
	assert.Equal(t, "src", srcFolder.Path)
	assert.Equal(t, schema.HotMode, srcFolder.Mode)
	assert.Equal(t, 15, srcFolder.Commits)        // 10 + 5
	assert.Equal(t, 75, srcFolder.Churn)          // 50 + 25
	assert.Equal(t, 150, srcFolder.TotalLOC)      // 100 + 50
	assert.True(t, srcFolder.Score > 0)           // Should be calculated
	assert.Contains(t, srcFolder.Owners, "Alice") // Most commits by Alice

	// Check tests/ folder
	require.NotNil(t, testsFolder)
	assert.Equal(t, "tests", testsFolder.Path)
	assert.Equal(t, 8, testsFolder.Commits)
	assert.Equal(t, 40, testsFolder.Churn)
	assert.Equal(t, 80, testsFolder.TotalLOC)
	assert.True(t, testsFolder.Score > 0)
}

func TestAggregateAndScoreFolders_WithPathFilter(t *testing.T) {
	// Create test file results
	fileResults := []schema.FileResult{
		{
			Path:        "src/main.go",
			Commits:     10,
			Churn:       50,
			LinesOfCode: 100,
			ModeScore:   25.0,
			Owners:      []string{"Alice"},
		},
		{
			Path:        "src/utils.go",
			Commits:     5,
			Churn:       25,
			LinesOfCode: 50,
			ModeScore:   15.0,
			Owners:      []string{"Bob"},
		},
		{
			Path:        "tests/main_test.go",
			Commits:     8,
			Churn:       40,
			LinesOfCode: 80,
			ModeScore:   20.0,
			Owners:      []string{"Alice"},
		},
	}

	cfg := &contract.Config{
		PathFilter: "src/", // Only analyze files in src/
		Mode:       schema.HotMode,
	}

	result := agg.AggregateAndScoreFolders(cfg, fileResults)

	// PathFilter doesn't affect folder aggregation - should still have both folders
	assert.Len(t, result, 2)
}

func TestAggregateAndScoreFolders_EmptyInput(t *testing.T) {
	var fileResults []schema.FileResult
	cfg := &contract.Config{Mode: schema.HotMode}

	result := agg.AggregateAndScoreFolders(cfg, fileResults)

	assert.Empty(t, result)
}

func TestAggregateAndScoreFolders_SingleFileInRoot(t *testing.T) {
	fileResults := []schema.FileResult{
		{
			Path:        "main.go",
			Commits:     5,
			Churn:       25,
			LinesOfCode: 50,
			ModeScore:   12.0,
			Owners:      []string{"Alice"},
		},
	}

	cfg := &contract.Config{
		Mode: schema.HotMode,
	}

	result := agg.AggregateAndScoreFolders(cfg, fileResults)

	// Should not include root folder when PathFilter is empty
	assert.Empty(t, result)
}

func TestAggregateAndScoreFolders_OwnerCalculation(t *testing.T) {
	// Test owner calculation with multiple contributors
	fileResults := []schema.FileResult{
		{
			Path:        "src/feature1.go",
			Commits:     8,
			Churn:       40,
			LinesOfCode: 80,
			ModeScore:   20.0,
			Owners:      []string{"Alice"}, // Alice has 8 commits
		},
		{
			Path:        "src/feature2.go",
			Commits:     12,
			Churn:       60,
			LinesOfCode: 120,
			ModeScore:   25.0,
			Owners:      []string{"Bob"}, // Bob has 12 commits
		},
		{
			Path:        "src/feature3.go",
			Commits:     6,
			Churn:       30,
			LinesOfCode: 60,
			ModeScore:   15.0,
			Owners:      []string{"Alice"}, // Alice has another 6 commits
		},
	}

	cfg := &contract.Config{
		Mode: schema.HotMode,
	}

	result := agg.AggregateAndScoreFolders(cfg, fileResults)

	assert.Len(t, result, 1)
	folder := result[0]

	// Alice should be the primary owner (14 commits > Bob's 12)
	assert.Equal(t, []string{"Alice", "Bob"}, folder.Owners)
	assert.Equal(t, 26, folder.Commits)   // 8 + 12 + 6
	assert.Equal(t, 130, folder.Churn)    // 40 + 60 + 30
	assert.Equal(t, 260, folder.TotalLOC) // 80 + 120 + 60
}

func TestAggregateAndScoreFolders_NoOwners(t *testing.T) {
	fileResults := []schema.FileResult{
		{
			Path:        "src/main.go",
			Commits:     5,
			Churn:       25,
			LinesOfCode: 50,
			ModeScore:   12.0,
			Owners:      []string{}, // No owners
		},
	}

	cfg := &contract.Config{
		Mode: schema.HotMode,
	}

	result := agg.AggregateAndScoreFolders(cfg, fileResults)

	assert.Len(t, result, 1)
	folder := result[0]

	// Should have empty owners
	assert.Empty(t, folder.Owners)
	assert.True(t, folder.Score >= 0)
}

func TestRankFolders(t *testing.T) {
	folders := []schema.FolderResult{
		{Path: "src", Score: 15.0, Commits: 50},
		{Path: "tests", Score: 25.0, Commits: 30},
		{Path: "docs", Score: 10.0, Commits: 20},
		{Path: "utils", Score: 20.0, Commits: 40},
	}

	result := algo.RankFolders(folders, 10)

	// Should be sorted by score descending: tests (25), utils (20), src (15), docs (10)
	assert.Len(t, result, 4)
	assert.Equal(t, "tests", result[0].Path)
	assert.Equal(t, 25.0, result[0].Score)
	assert.Equal(t, "utils", result[1].Path)
	assert.Equal(t, 20.0, result[1].Score)
	assert.Equal(t, "src", result[2].Path)
	assert.Equal(t, 15.0, result[2].Score)
	assert.Equal(t, "docs", result[3].Path)
	assert.Equal(t, 10.0, result[3].Score)
}

func TestRankFolders_WithLimit(t *testing.T) {
	folders := []schema.FolderResult{
		{Path: "src", Score: 15.0},
		{Path: "tests", Score: 25.0},
		{Path: "docs", Score: 10.0},
		{Path: "utils", Score: 20.0},
	}

	result := algo.RankFolders(folders, 2)

	// Should return only top 2: tests (25), utils (20)
	assert.Len(t, result, 2)
	assert.Equal(t, "tests", result[0].Path)
	assert.Equal(t, 25.0, result[0].Score)
	assert.Equal(t, "utils", result[1].Path)
	assert.Equal(t, 20.0, result[1].Score)
}

func TestRankFolders_EmptyInput(t *testing.T) {
	var folders []schema.FolderResult

	result := algo.RankFolders(folders, 10)

	assert.Empty(t, result)
}

func TestRankFolders_LimitGreaterThanLength(t *testing.T) {
	folders := []schema.FolderResult{
		{Path: "src", Score: 15.0},
		{Path: "tests", Score: 25.0},
	}

	result := algo.RankFolders(folders, 10)

	// Should return all folders when limit > length
	assert.Len(t, result, 2)
	assert.Equal(t, "tests", result[0].Path)
	assert.Equal(t, "src", result[1].Path)
}

func TestRankFolders_ZeroLimit(t *testing.T) {
	folders := []schema.FolderResult{
		{Path: "src", Score: 15.0},
		{Path: "tests", Score: 25.0},
	}

	result := algo.RankFolders(folders, 0)

	// Should return empty slice when limit is 0
	assert.Empty(t, result)
}

func TestCompareFolderMetrics(t *testing.T) {
	baseFolders := []schema.FolderResult{
		{Path: "src", Score: 20.0, Commits: 50, Churn: 100, Owners: []string{"Alice"}},
		{Path: "tests", Score: 15.0, Commits: 30, Churn: 60, Owners: []string{"Bob"}},
		{Path: "docs", Score: 5.0, Commits: 10, Churn: 20, Owners: []string{"Charlie"}},
	}

	targetFolders := []schema.FolderResult{
		{Path: "src", Score: 25.0, Commits: 60, Churn: 120, Owners: []string{"Alice", "Dave"}},
		{Path: "tests", Score: 10.0, Commits: 25, Churn: 50, Owners: []string{"Bob"}},
		{Path: "utils", Score: 18.0, Commits: 40, Churn: 80, Owners: []string{"Eve"}}, // New folder
	}

	result := compareFolderMetrics(baseFolders, targetFolders, 10, "hot")

	assert.NotNil(t, result)
	assert.NotNil(t, result.Details)
	assert.NotNil(t, result.Summary)

	// Should have results for paths with significant score changes
	assert.True(t, len(result.Details) > 0)

	// Check summary
	assert.True(t, result.Summary.NetScoreDelta > 0)      // Overall score increase
	assert.True(t, result.Summary.NetChurnDelta > 0)      // Overall churn increase
	assert.Equal(t, 1, result.Summary.TotalNewFiles)      // utils folder is new
	assert.Equal(t, 1, result.Summary.TotalInactiveFiles) // docs folder became inactive
	assert.Equal(t, 2, result.Summary.TotalModifiedFiles) // src and tests were modified
}

func TestCompareFolderMetrics_NoChanges(t *testing.T) {
	baseFolders := []schema.FolderResult{
		{Path: "src", Score: 20.0, Commits: 50, Churn: 100, Owners: []string{"Alice"}},
	}

	targetFolders := []schema.FolderResult{
		{Path: "src", Score: 20.0, Commits: 50, Churn: 100, Owners: []string{"Alice"}},
	}

	result := compareFolderMetrics(baseFolders, targetFolders, 10, "hot")

	assert.NotNil(t, result)
	assert.Empty(t, result.Details) // No significant changes

	// Summary should be zero except for modified files (existing in both)
	assert.Equal(t, 0.0, result.Summary.NetScoreDelta)
	assert.Equal(t, 0, result.Summary.NetChurnDelta)
	assert.Equal(t, 0, result.Summary.TotalNewFiles)
	assert.Equal(t, 0, result.Summary.TotalInactiveFiles)
	assert.Equal(t, 1, result.Summary.TotalModifiedFiles) // Folder exists in both base and target
	assert.Equal(t, 0, result.Summary.TotalOwnershipChanges)
}

func TestCompareFolderMetrics_EmptyBase(t *testing.T) {
	var baseFolders []schema.FolderResult
	targetFolders := []schema.FolderResult{
		{Path: "src", Score: 20.0, Commits: 50, Churn: 100, Owners: []string{"Alice"}},
	}

	result := compareFolderMetrics(baseFolders, targetFolders, 10, "hot")

	assert.NotNil(t, result)
	assert.Len(t, result.Details, 1)
	assert.Equal(t, schema.NewStatus, result.Details[0].Status)
	assert.Equal(t, 1, result.Summary.TotalNewFiles)
}

func TestCompareFolderMetrics_EmptyTarget(t *testing.T) {
	baseFolders := []schema.FolderResult{
		{Path: "src", Score: 20.0, Commits: 50, Churn: 100, Owners: []string{"Alice"}},
	}
	var targetFolders []schema.FolderResult

	result := compareFolderMetrics(baseFolders, targetFolders, 10, "hot")

	assert.NotNil(t, result)
	assert.Len(t, result.Details, 1)
	assert.Equal(t, schema.InactiveStatus, result.Details[0].Status)
	assert.Equal(t, 1, result.Summary.TotalInactiveFiles)
}

func TestCompareFolderMetrics_WithLimit(t *testing.T) {
	baseFolders := []schema.FolderResult{
		{Path: "src", Score: 10.0, Commits: 20, Churn: 40, Owners: []string{"Alice"}},
		{Path: "tests", Score: 15.0, Commits: 30, Churn: 60, Owners: []string{"Bob"}},
	}

	targetFolders := []schema.FolderResult{
		{Path: "src", Score: 25.0, Commits: 50, Churn: 100, Owners: []string{"Alice"}}, // +15 delta
		{Path: "tests", Score: 20.0, Commits: 40, Churn: 80, Owners: []string{"Bob"}},  // +5 delta
	}

	result := compareFolderMetrics(baseFolders, targetFolders, 1, "hot")

	// Should return only the top 1 result (largest delta)
	assert.Len(t, result.Details, 1)
	assert.Equal(t, "src", result.Details[0].Path)
	assert.Equal(t, 15.0, result.Details[0].Delta)
}

func TestCompareFolderMetrics_OwnershipChange(t *testing.T) {
	baseFolders := []schema.FolderResult{
		{Path: "src", Score: 20.0, Commits: 50, Churn: 100, Owners: []string{"Alice"}},
	}

	targetFolders := []schema.FolderResult{
		{Path: "src", Score: 20.0, Commits: 50, Churn: 100, Owners: []string{"Bob"}}, // Same score, different owner
	}

	result := compareFolderMetrics(baseFolders, targetFolders, 10, "hot")

	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Summary.TotalOwnershipChanges)
	// No results since score delta is 0
	assert.Empty(t, result.Details)
}
