package core

import (
	"testing"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestCompareFileResults_StatusClassification(t *testing.T) {
	// Test that files are correctly classified based on existence in base vs target,
	// not based on activity. This prevents regression of the bug where files existing
	// in both refs were incorrectly marked as "new" instead of "active".

	baseResults := []schema.FileResult{
		{
			Path:               "existing_in_both.go",
			Score:              10.0,
			Commits:            5,
			Churn:              15,
			LinesOfCode:        100,
			UniqueContributors: 2,
		},
		{
			Path:               "only_in_base.go",
			Score:              5.0,
			Commits:            2,
			Churn:              8,
			LinesOfCode:        50,
			UniqueContributors: 1,
		},
	}

	targetResults := []schema.FileResult{
		{
			Path:               "existing_in_both.go",
			Score:              15.0, // Score increased
			Commits:            7,    // Commits increased
			Churn:              20,   // Churn increased
			LinesOfCode:        120,  // LOC increased
			UniqueContributors: 3,    // Contributors increased
		},
		{
			Path:               "only_in_target.go",
			Score:              8.0,
			Commits:            3,
			Churn:              12,
			LinesOfCode:        80,
			UniqueContributors: 2,
		},
	}

	result := compareFileResults(baseResults, targetResults, 10)

	// Verify we have results for all expected files
	assert.Len(t, result.Results, 3)

	// Create a map for easier lookup
	resultMap := make(map[string]schema.ComparisonDetails)
	for _, r := range result.Results {
		resultMap[r.Path] = r
	}

	// Test file that exists in both (should be "active")
	bothFile := resultMap["existing_in_both.go"]
	assert.Equal(t, schema.ActiveStatus, bothFile.Status)
	assert.Equal(t, 10.0, bothFile.BeforeScore)
	assert.Equal(t, 15.0, bothFile.AfterScore)
	assert.Equal(t, 5.0, bothFile.Delta)      // 15.0 - 10.0
	assert.Equal(t, 2, bothFile.DeltaCommits) // 7 - 5
	assert.Equal(t, 5, bothFile.DeltaChurn)   // 20 - 15
	assert.NotNil(t, bothFile.FileComparison)
	assert.Equal(t, 20, bothFile.DeltaLOC)    // 120 - 100
	assert.Equal(t, 1, bothFile.DeltaContrib) // 3 - 2

	// Test file that only exists in base (should be "inactive")
	baseOnlyFile := resultMap["only_in_base.go"]
	assert.Equal(t, schema.InactiveStatus, baseOnlyFile.Status)
	assert.Equal(t, 5.0, baseOnlyFile.BeforeScore)
	assert.Equal(t, 0.0, baseOnlyFile.AfterScore) // Default when not exists
	assert.Equal(t, -5.0, baseOnlyFile.Delta)
	assert.Equal(t, 0, baseOnlyFile.DeltaCommits) // No delta when target doesn't exist
	assert.Equal(t, 0, baseOnlyFile.DeltaChurn)
	assert.Nil(t, baseOnlyFile.FileComparison) // No file comparison when target doesn't exist

	// Test file that only exists in target (should be "new")
	targetOnlyFile := resultMap["only_in_target.go"]
	assert.Equal(t, schema.NewStatus, targetOnlyFile.Status)
	assert.Equal(t, 0.0, targetOnlyFile.BeforeScore) // Default when not exists
	assert.Equal(t, 8.0, targetOnlyFile.AfterScore)
	assert.Equal(t, 8.0, targetOnlyFile.Delta)
	assert.Equal(t, 0, targetOnlyFile.DeltaCommits) // No delta when base doesn't exist
	assert.Equal(t, 0, targetOnlyFile.DeltaChurn)
	assert.Nil(t, targetOnlyFile.FileComparison) // No file comparison when base doesn't exist

	// Verify summary counts
	assert.Equal(t, 1, result.Summary.TotalNewFiles)
	assert.Equal(t, 1, result.Summary.TotalInactiveFiles)
	assert.Equal(t, 1, result.Summary.TotalModifiedFiles)
	assert.Equal(t, 8.0, result.Summary.NetScoreDelta) // 5.0 + (-5.0) + 8.0
	assert.Equal(t, 5, result.Summary.NetChurnDelta)   // 5 + 0 + 0
}

func TestCompareFileResults_NoSignificantChanges(t *testing.T) {
	// Test that files with no significant score changes are excluded from results

	baseResults := []schema.FileResult{
		{
			Path:  "unchanged.go",
			Score: 10.0,
		},
		{
			Path:  "tiny_change.go",
			Score: 10.0,
		},
	}

	targetResults := []schema.FileResult{
		{
			Path:  "unchanged.go",
			Score: 10.0, // Exactly the same
		},
		{
			Path:  "tiny_change.go",
			Score: 10.001, // Change smaller than 0.01 threshold
		},
	}

	result := compareFileResults(baseResults, targetResults, 10)

	// Should have no results since changes are insignificant
	assert.Len(t, result.Results, 0)
	assert.Equal(t, 0, result.Summary.TotalNewFiles)
	assert.Equal(t, 0, result.Summary.TotalInactiveFiles)
	assert.Equal(t, 2, result.Summary.TotalModifiedFiles) // Both exist in both, so they're "modified" even with no significant change
	assert.InDelta(t, 0.001, result.Summary.NetScoreDelta, 0.0001)
}

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		name         string
		baseExists   bool
		targetExists bool
		expected     string
	}{
		{"new file", false, true, schema.NewStatus},
		{"active file", true, true, schema.ActiveStatus},
		{"inactive file", true, false, schema.InactiveStatus},
		{"unknown case", false, false, schema.UnknownStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineStatus(tt.baseExists, tt.targetExists)
			assert.Equal(t, tt.expected, result)
		})
	}
}
