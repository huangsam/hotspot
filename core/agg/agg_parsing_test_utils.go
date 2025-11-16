package agg

import (
	"fmt"
	"strings"
	"time"
)

// gitLogScenario represents a single commit scenario for test data generation.
type gitLogScenario struct {
	commitHash string
	author     string
	date       time.Time
	files      []fileChange
}

// fileChange represents a single file change in a commit.
type fileChange struct {
	path      string
	additions int
	deletions int
}

// generateTestGitLog creates a programmatic git log fixture for testing.
func generateTestGitLog(scenarios []gitLogScenario) []byte {
	var lines []string
	for _, scenario := range scenarios {
		lines = append(lines, fmt.Sprintf("--%s|%s|%s", scenario.commitHash, scenario.author, scenario.date.Format(time.RFC3339)))
		for _, file := range scenario.files {
			lines = append(lines, fmt.Sprintf("%d\t%d\t%s", file.additions, file.deletions, file.path))
		}
		lines = append(lines, "") // Empty line between commits
	}
	return []byte(strings.Join(lines, "\n"))
}

// generateComprehensiveTestData creates test data that covers various parsing scenarios.
func generateComprehensiveTestData() ([]byte, map[string]bool) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	scenarios := []gitLogScenario{
		{
			commitHash: "abc123def456",
			author:     "Alice Developer",
			date:       baseTime,
			files: []fileChange{
				{"core/analysis.go", 50, 10},
				{"core/core.go", 100, 5},
			},
		},
		{
			commitHash: "def456ghi789",
			author:     "Bob Tester",
			date:       baseTime.Add(time.Hour),
			files: []fileChange{
				{"core/analysis.go", 25, 5},
				{"core/builder.go", 75, 0},
			},
		},
		{
			commitHash: "ghi789jkl012",
			author:     "Alice Developer",
			date:       baseTime.Add(2 * time.Hour),
			files: []fileChange{
				{"core/core.go", 200, 50},
				{"core/analysis.go", 10, 2},
			},
		},
	}

	fileExists := map[string]bool{
		"core/analysis.go": true,
		"core/core.go":     true,
		"core/builder.go":  true,
	}

	return generateTestGitLog(scenarios), fileExists
}

// generateRenameTestData creates test data for rename scenarios.
func generateRenameTestData() ([]byte, map[string]bool) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	scenarios := []gitLogScenario{
		{
			commitHash: "rename123abc",
			author:     "Charlie Refactor",
			date:       baseTime,
			files: []fileChange{
				{"src/utils/helper.go", 10, 2},
			},
		},
		{
			commitHash: "rename456def",
			author:     "Charlie Refactor",
			date:       baseTime.Add(time.Hour),
			files: []fileChange{
				{"src/utils/helper.go => src/helpers/utility.go", 8, 1},
			},
		},
		{
			commitHash: "update789ghi",
			author:     "Charlie Refactor",
			date:       baseTime.Add(2 * time.Hour),
			files: []fileChange{
				{"src/helpers/utility.go", 15, 7},
			},
		},
		{
			commitHash: "main012jkl",
			author:     "Alice Developer",
			date:       baseTime.Add(3 * time.Hour),
			files: []fileChange{
				{"src/main.go", 12, 4},
			},
		},
	}

	fileExists := map[string]bool{
		"src/utils/helper.go":    true,
		"src/helpers/utility.go": true,
		"src/main.go":            true,
	}

	return generateTestGitLog(scenarios), fileExists
}

// generateEdgeCaseTestData creates test data for edge cases.
func generateEdgeCaseTestData() ([]byte, map[string]bool) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	scenarios := []gitLogScenario{
		{
			commitHash: "normal123abc",
			author:     "Alice Developer",
			date:       baseTime,
			files: []fileChange{
				{"src/main.go", 5, 2},
			},
		},
		{
			commitHash: "binary456def",
			author:     "Bob Tester",
			date:       baseTime.Add(time.Hour),
			files: []fileChange{
				{"src/logo.png", 0, 0}, // Binary file (represented as 0 churn)
			},
		},
		{
			commitHash: "empty789ghi",
			author:     "Charlie Dev",
			date:       baseTime.Add(2 * time.Hour),
			files: []fileChange{
				{"src/empty.txt", 0, 0}, // Empty file
			},
		},
	}

	fileExists := map[string]bool{
		"src/main.go":   true,
		"src/logo.png":  true,
		"src/empty.txt": true,
	}

	return generateTestGitLog(scenarios), fileExists
}

// createTestFileExistsMap creates a standard file existence map for testing.
func createTestFileExistsMap(files []string) map[string]bool {
	result := make(map[string]bool)
	for _, file := range files {
		result[file] = true
	}
	return result
}

// createAggregationMaps creates initialized aggregation maps for testing.
func createAggregationMaps() (map[string]int, map[string]int, map[string]map[string]int, map[string]time.Time) {
	return make(map[string]int), make(map[string]int), make(map[string]map[string]int), make(map[string]time.Time)
}
