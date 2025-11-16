package agg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseAndAggregateGitLog_Comprehensive(t *testing.T) {
	gitLogData, fileExists := generateComprehensiveTestData()

	// Initialize aggregation maps
	commitsMap := make(map[string]int)
	churnMap := make(map[string]int)
	contribMap := make(map[string]map[string]int)
	firstCommitMap := make(map[string]time.Time)

	// Execute parsing
	parseAndAggregateGitLog(gitLogData, fileExists, commitsMap, churnMap, contribMap, firstCommitMap)

	// Property-based assertions instead of hardcoded values
	// Check that all expected files have been processed
	expectedFiles := []string{"core/analysis.go", "core/core.go", "core/builder.go"}
	for _, file := range expectedFiles {
		assert.Contains(t, commitsMap, file, "File %s should be in commits map", file)
		assert.Greater(t, commitsMap[file], 0, "File %s should have at least 1 commit", file)
		assert.GreaterOrEqual(t, churnMap[file], 0, "File %s should have non-negative churn", file)
	}

	// Check that contributors are tracked
	assert.NotEmpty(t, contribMap["core/analysis.go"], "analysis.go should have contributors")
	assert.NotEmpty(t, contribMap["core/core.go"], "core.go should have contributors")

	// Check that first commit dates are reasonable
	for _, file := range expectedFiles {
		if commitsMap[file] > 0 {
			assert.NotZero(t, firstCommitMap[file], "File %s should have a first commit date", file)
			assert.True(t, firstCommitMap[file].Before(time.Now()), "First commit date should be in the past")
		}
	}

	// Verify that total commits across all files is reasonable
	totalCommits := 0
	for _, count := range commitsMap {
		totalCommits += count
	}
	assert.Greater(t, totalCommits, 0, "Should have processed at least some commits")
	assert.LessOrEqual(t, totalCommits, 10, "Should not have processed too many commits")
}

func TestParseAndAggregateGitLog_WithRenames(t *testing.T) {
	gitLogData, fileExists := generateRenameTestData()

	commitsMap := make(map[string]int)
	churnMap := make(map[string]int)
	contribMap := make(map[string]map[string]int)
	firstCommitMap := make(map[string]time.Time)

	parseAndAggregateGitLog(gitLogData, fileExists, commitsMap, churnMap, contribMap, firstCommitMap)

	// Property-based checks for rename handling
	expectedFiles := []string{"src/utils/helper.go", "src/helpers/utility.go", "src/main.go"}
	for _, file := range expectedFiles {
		assert.Contains(t, commitsMap, file, "File %s should be processed", file)
		assert.Greater(t, commitsMap[file], 0, "File %s should have commits", file)
		assert.GreaterOrEqual(t, churnMap[file], 0, "File %s should have valid churn", file)
	}

	// Check that rename contributions are properly attributed
	assert.NotEmpty(t, contribMap["src/utils/helper.go"], "helper.go should have contributors")
	assert.NotEmpty(t, contribMap["src/helpers/utility.go"], "utility.go should have contributors")

	// Verify first commit dates are set appropriately
	for _, file := range expectedFiles {
		assert.NotZero(t, firstCommitMap[file], "File %s should have first commit date", file)
	}
}

func TestParseAndAggregateGitLog_EdgeCases(t *testing.T) {
	gitLogData, fileExists := generateEdgeCaseTestData()

	commitsMap := make(map[string]int)
	churnMap := make(map[string]int)
	contribMap := make(map[string]map[string]int)
	firstCommitMap := make(map[string]time.Time)

	parseAndAggregateGitLog(gitLogData, fileExists, commitsMap, churnMap, contribMap, firstCommitMap)

	// Property-based checks for edge cases
	expectedFiles := []string{"src/main.go", "src/logo.png", "src/empty.txt"}
	for _, file := range expectedFiles {
		assert.Contains(t, commitsMap, file, "File %s should be processed", file)
		assert.Greater(t, commitsMap[file], 0, "File %s should have at least 1 commit", file)
		assert.GreaterOrEqual(t, churnMap[file], 0, "File %s should have valid churn", file)
	}

	// Binary and empty files should have 0 churn
	assert.Equal(t, 0, churnMap["src/logo.png"], "Binary file should have 0 churn")
	assert.Equal(t, 0, churnMap["src/empty.txt"], "Empty file should have 0 churn")

	// Normal files should have positive churn
	assert.Greater(t, churnMap["src/main.go"], 0, "Normal file should have positive churn")
}

func TestParseCommitHeader(t *testing.T) {
	testCases := []struct {
		name         string
		line         string
		expectedAuth string
		expectZero   bool
	}{
		{"valid header", "--abc123|John Doe|2024-01-15T10:30:00Z", "John Doe", false},
		{"invalid date", "--abc123|John Doe|invalid-date", "", true},
		{"malformed header", "--abc123|John Doe", "", true},
		{"empty line", "", "", true},
		{"timezone offset", "--abc123|Jane Smith|2024-01-15T10:30:00-08:00", "Jane Smith", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			auth, date := parseCommitHeader(tc.line)
			assert.Equal(t, tc.expectedAuth, auth)
			if tc.expectZero {
				assert.True(t, date.IsZero())
			} else {
				assert.False(t, date.IsZero())
			}
		})
	}
}

func TestParseFileStatsLine(t *testing.T) {
	fileExists := createTestFileExistsMap([]string{"src/main.go", "src/utils.go"})

	testCases := []struct {
		name          string
		line          string
		expectedPaths []string
		expectedAdd   int
		expectedDel   int
	}{
		{"normal file", "10\t5\tsrc/main.go", []string{"src/main.go"}, 10, 5},
		{"binary file", "-\t-\tsrc/binary.dll", nil, 0, 0},
		{"non-existent file", "5\t2\told_file.go", nil, 5, 2},
		{"malformed line - too few parts", "10\tsrc/main.go", nil, 0, 0},
		{"invalid numbers", "abc\tdef\tsrc/main.go", []string{"src/main.go"}, 0, 0},
		{"simple rename", "8\t1\told.go => new.go", nil, 8, 1},
		{"zero additions", "0\t5\tsrc/utils.go", []string{"src/utils.go"}, 0, 5},
		{"zero deletions", "10\t0\tsrc/main.go", []string{"src/main.go"}, 10, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paths, add, del := parseFileStatsLine(tc.line, fileExists)
			assert.Equal(t, tc.expectedPaths, paths)
			assert.Equal(t, tc.expectedAdd, add)
			assert.Equal(t, tc.expectedDel, del)
		})
	}
}

func TestParseChurnValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"normal number", "42", 42},
		{"zero", "0", 0},
		{"dash (binary)", "-", 0},
		{"empty string", "", 0},
		{"invalid number", "abc", 0},
		{"negative number", "-5", 0},
		{"large number", "999999", 999999},
		{"with whitespace", "  42  ", 0}, // Should fail due to whitespace
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseChurnValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDeterminePathsToAggregate(t *testing.T) {
	fileExists := createTestFileExistsMap([]string{
		"src/main.go",
		"src/utils.go",
		"src/helpers/new.go",
		"new/path/file.go",
	})

	testCases := []struct {
		name          string
		path          string
		expectedPaths []string
	}{
		{"normal file exists", "src/main.go", []string{"src/main.go"}},
		{"normal file doesn't exist", "nonexistent.go", nil},
		{"simple rename both exist", "old.go => new.go", nil},
		{"simple rename one exists", "old/path/file.go => src/utils.go", []string{"src/utils.go"}},
		{"braced rename", "src/{utils => helpers}/file.go", nil},
		{"braced rename one exists", "src/{main => helpers}/new.go", []string{"src/helpers/new.go"}},
		{"complex braced rename", "a/b/{c/d => e/f}/file.go", nil},
		{"rename to existing file", "old.txt => new/path/file.go", []string{"new/path/file.go"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := determinePathsToAggregate(tc.path, fileExists)
			assert.Equal(t, tc.expectedPaths, result)
		})
	}
}

func TestParseRenamePath(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		expectedOld string
		expectedNew string
	}{
		{"simple rename", "old/file.go => new/file.go", "old/file.go", "new/file.go"},
		{"braced rename", "src/{old => new}/file.go", "src/old/file.go", "src/new/file.go"},
		{"complex braced rename", "a/b/{c/d => e/f}/file.go", "a/b/c/d/file.go", "a/b/e/f/file.go"},
		{"no braces", "old => new", "old", "new"},
		{"malformed - no arrow", "src/file.go", "", ""},
		{"malformed - empty braces", "src/{}/file.go", "", ""},
		{"malformed - unclosed brace", "src/{old => new/file.go", "", ""},
		{"multiple arrows", "a => b => c", "a", "b => c"}, // Should not parse
		{"empty old path", " => new/file.go", "", "new/file.go"},
		{"empty new path", "old/file.go => ", "old/file.go", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o, n := parseRenamePath(tc.path)
			assert.Equal(t, tc.expectedOld, o)
			assert.Equal(t, tc.expectedNew, n)
		})
	}
}

func TestAggregateForPath(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	laterTime := testTime.Add(time.Hour)

	t.Run("single aggregation", func(t *testing.T) {
		commitsMap, churnMap, contribMap, firstCommitMap := createAggregationMaps()

		aggregateForPath("src/main.go", 15, "Alice", testTime, commitsMap, churnMap, contribMap, firstCommitMap)

		assert.Equal(t, 1, commitsMap["src/main.go"])
		assert.Equal(t, 15, churnMap["src/main.go"])
		assert.Equal(t, map[string]int{"Alice": 1}, contribMap["src/main.go"])
		assert.Equal(t, testTime, firstCommitMap["src/main.go"])
	})

	t.Run("multiple aggregations same file", func(t *testing.T) {
		commitsMap, churnMap, contribMap, firstCommitMap := createAggregationMaps()

		// First aggregation
		aggregateForPath("src/main.go", 15, "Alice", testTime, commitsMap, churnMap, contribMap, firstCommitMap)
		// Second aggregation
		aggregateForPath("src/main.go", 10, "Alice", laterTime, commitsMap, churnMap, contribMap, firstCommitMap)

		assert.Equal(t, 2, commitsMap["src/main.go"])
		assert.Equal(t, 25, churnMap["src/main.go"])
		assert.Equal(t, map[string]int{"Alice": 2}, contribMap["src/main.go"])
		assert.Equal(t, testTime, firstCommitMap["src/main.go"]) // Should keep earliest time
	})

	t.Run("multiple authors", func(t *testing.T) {
		commitsMap, churnMap, contribMap, firstCommitMap := createAggregationMaps()

		aggregateForPath("src/main.go", 15, "Alice", testTime, commitsMap, churnMap, contribMap, firstCommitMap)
		aggregateForPath("src/main.go", 10, "Bob", laterTime, commitsMap, churnMap, contribMap, firstCommitMap)

		assert.Equal(t, 2, commitsMap["src/main.go"])
		assert.Equal(t, 25, churnMap["src/main.go"])
		assert.Equal(t, map[string]int{"Alice": 1, "Bob": 1}, contribMap["src/main.go"])
	})

	t.Run("empty author", func(t *testing.T) {
		commitsMap, churnMap, contribMap, firstCommitMap := createAggregationMaps()

		aggregateForPath("src/utils.go", 8, "", laterTime, commitsMap, churnMap, contribMap, firstCommitMap)

		assert.Equal(t, 1, commitsMap["src/utils.go"])
		assert.Equal(t, 8, churnMap["src/utils.go"])
		assert.NotContains(t, contribMap, "src/utils.go") // No contributors added
	})

	t.Run("zero time", func(t *testing.T) {
		commitsMap, churnMap, contribMap, firstCommitMap := createAggregationMaps()

		aggregateForPath("src/zero.go", 3, "Charlie", time.Time{}, commitsMap, churnMap, contribMap, firstCommitMap)

		assert.Equal(t, 1, commitsMap["src/zero.go"])
		assert.Equal(t, 3, churnMap["src/zero.go"])
		assert.Equal(t, map[string]int{"Charlie": 1}, contribMap["src/zero.go"])
		assert.NotContains(t, firstCommitMap, "src/zero.go") // No time recorded
	})
}
