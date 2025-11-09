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
			Mode:               schema.HotMode,
		},
		{
			Path:               "only_in_base.go",
			Score:              5.0,
			Commits:            2,
			Churn:              8,
			LinesOfCode:        50,
			UniqueContributors: 1,
			Mode:               schema.HotMode,
		},
	}

	targetResults := []schema.FileResult{
		{
			Path:               "existing_in_both.go",
			Score:              12.0,
			Commits:            7,
			Churn:              20,
			LinesOfCode:        110,
			UniqueContributors: 3,
			Mode:               schema.HotMode,
		},
		{
			Path:               "only_in_target.go",
			Score:              8.0,
			Commits:            4,
			Churn:              12,
			LinesOfCode:        80,
			UniqueContributors: 2,
			Mode:               schema.HotMode,
		},
	}

	result := compareFileResults(baseResults, targetResults, 10, string(schema.HotMode))

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
	assert.Equal(t, 12.0, bothFile.AfterScore)
	assert.Equal(t, 2.0, bothFile.Delta)      // 12.0 - 10.0
	assert.Equal(t, 2, bothFile.DeltaCommits) // 7 - 5
	assert.Equal(t, 5, bothFile.DeltaChurn)   // 20 - 15
	assert.NotNil(t, bothFile.FileComparison)
	assert.Equal(t, 10, bothFile.DeltaLOC)    // 110 - 100
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
	assert.Equal(t, 5.0, result.Summary.NetScoreDelta) // 2.0 + (-5.0) + 8.0
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

	result := compareFileResults(baseResults, targetResults, 10, string(schema.HotMode))

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
		expected     schema.Status
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

func TestCompareFileResults_OwnershipChanges(t *testing.T) {
	// Test that ownership changes are correctly detected and counted

	baseResults := []schema.FileResult{
		{
			Path:   "same_owner.go",
			Score:  10.0,
			Owners: []string{"Alice"},
		},
		{
			Path:   "changed_owner.go",
			Score:  10.0,
			Owners: []string{"Alice"},
		},
		{
			Path:   "multiple_owners_same.go",
			Score:  10.0,
			Owners: []string{"Alice", "Bob"},
		},
		{
			Path:   "multiple_owners_changed.go",
			Score:  10.0,
			Owners: []string{"Alice", "Bob"},
		},
	}

	targetResults := []schema.FileResult{
		{
			Path:   "same_owner.go",
			Score:  10.0,              // No significant change
			Owners: []string{"Alice"}, // Same owner
		},
		{
			Path:   "changed_owner.go",
			Score:  11.0,            // Significant change
			Owners: []string{"Bob"}, // Different owner
		},
		{
			Path:   "multiple_owners_same.go",
			Score:  12.0,                     // Significant change
			Owners: []string{"Bob", "Alice"}, // Same owners, different order
		},
		{
			Path:   "multiple_owners_changed.go",
			Score:  13.0,                         // Significant change
			Owners: []string{"Alice", "Charlie"}, // One owner changed
		},
	}

	result := compareFileResults(baseResults, targetResults, 10, string(schema.HotMode))

	// Should have 3 results with significant score changes
	assert.Len(t, result.Results, 3)

	// Check ownership changes count
	assert.Equal(t, 2, result.Summary.TotalOwnershipChanges) // changed_owner.go and multiple_owners_changed.go

	// Check the owners are populated correctly
	resultMap := make(map[string]schema.ComparisonDetails)
	for _, r := range result.Results {
		resultMap[r.Path] = r
	}

	changedOwner := resultMap["changed_owner.go"]
	assert.Equal(t, []string{"Alice"}, changedOwner.BeforeOwners)
	assert.Equal(t, []string{"Bob"}, changedOwner.AfterOwners)

	multipleSame := resultMap["multiple_owners_same.go"]
	assert.Equal(t, []string{"Alice", "Bob"}, multipleSame.BeforeOwners)
	assert.Equal(t, []string{"Bob", "Alice"}, multipleSame.AfterOwners) // Order doesn't matter for equality check
}
