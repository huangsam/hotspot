package agg

import (
	"testing"
	"time"

	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestParseAndAggregateGitLog_Comprehensive(t *testing.T) {
	gitLogData, fileExists := generateComprehensiveTestData()

	// Initialize aggregation maps
	output := initializeAggregateOutput(time.Now())
	recentThreshold := time.Now().AddDate(0, 0, -30)

	// Execute parsing
	parseAndAggregateGitLog(gitLogData, fileExists, output, recentThreshold)

	// Property-based assertions instead of hardcoded values
	// Check that all expected files have been processed
	expectedFiles := []string{"core/analysis.go", "core/core.go", "core/builder.go"}
	for _, file := range expectedFiles {
		stat, ok := output.FileStats[file]
		assert.True(t, ok, "File %s should be in FileStats map", file)
		assert.Greater(t, stat.Commits.Float64(), 0.0, "File %s should have at least 1 commit", file)
		assert.GreaterOrEqual(t, stat.Churn.Float64(), 0.0, "File %s should have non-negative churn", file)
	}

	// Check that contributors are tracked
	assert.NotEmpty(t, output.FileStats["core/analysis.go"].Contributors, "analysis.go should have contributors")
	assert.NotEmpty(t, output.FileStats["core/core.go"].Contributors, "core.go should have contributors")

	// Check that first commit dates are reasonable
	for _, file := range expectedFiles {
		stat := output.FileStats[file]
		if stat.Commits > 0 {
			assert.NotZero(t, stat.FirstCommit, "File %s should have a first commit date", file)
			assert.True(t, stat.FirstCommit.Before(time.Now()), "First commit date should be in the past")
		}
	}

	// Verify that total commits across all files is reasonable
	var totalCommits schema.Metric
	for _, stat := range output.FileStats {
		totalCommits += stat.Commits
	}
	assert.Greater(t, totalCommits.Float64(), 0.0, "Should have processed at least some commits")
	assert.LessOrEqual(t, totalCommits.Float64(), 10.0, "Should not have processed too many commits")
}

func TestParseAndAggregateGitLog_WithRenames(t *testing.T) {
	gitLogData, fileExists := generateRenameTestData()

	output := initializeAggregateOutput(time.Now())
	recentThreshold := time.Now().AddDate(0, 0, -30)

	parseAndAggregateGitLog(gitLogData, fileExists, output, recentThreshold)

	// Property-based checks for rename handling
	expectedFiles := []string{"src/utils/helper.go", "src/helpers/utility.go", "src/main.go"}
	for _, file := range expectedFiles {
		stat, ok := output.FileStats[file]
		assert.True(t, ok, "File %s should be processed", file)
		assert.Greater(t, stat.Commits.Float64(), 0.0, "File %s should have commits", file)
		assert.GreaterOrEqual(t, stat.Churn.Float64(), 0.0, "File %s should have valid churn", file)
	}

	// Check that rename contributions are properly attributed
	assert.NotEmpty(t, output.FileStats["src/utils/helper.go"].Contributors, "helper.go should have contributors")
	assert.NotEmpty(t, output.FileStats["src/helpers/utility.go"].Contributors, "utility.go should have contributors")

	// Verify first commit dates are set appropriately
	for _, file := range expectedFiles {
		assert.NotZero(t, output.FileStats[file].FirstCommit, "File %s should have first commit date", file)
	}
}

func TestParseAndAggregateGitLog_EdgeCases(t *testing.T) {
	gitLogData, fileExists := generateEdgeCaseTestData()

	output := initializeAggregateOutput(time.Now())
	recentThreshold := time.Now().AddDate(0, 0, -30)

	parseAndAggregateGitLog(gitLogData, fileExists, output, recentThreshold)

	// Property-based checks for edge cases
	expectedFiles := []string{"src/main.go", "src/logo.png", "src/empty.txt"}
	for _, file := range expectedFiles {
		stat, ok := output.FileStats[file]
		assert.True(t, ok, "File %s should be processed", file)
		assert.Greater(t, stat.Commits.Float64(), 0.0, "File %s should have at least 1 commit", file)
		assert.GreaterOrEqual(t, stat.Churn.Float64(), 0.0, "File %s should have valid churn", file)
	}

	// Binary and empty files should have 0 churn
	assert.Equal(t, schema.Metric(0), output.FileStats["src/logo.png"].Churn, "Binary file should have 0 churn")
	assert.Equal(t, schema.Metric(0), output.FileStats["src/empty.txt"].Churn, "Empty file should have 0 churn")

	// Normal files should have positive churn
	assert.Greater(t, output.FileStats["src/main.go"].Churn.Float64(), 0.0, "Normal file should have positive churn")
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
			auth, date := parseCommitHeader([]byte(tc.line), make(map[string]string))
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
		expectedAdd   schema.Metric
		expectedDel   schema.Metric
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
			paths, add, del := parseFileStatsLine([]byte(tc.line), fileExists)
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
		expected schema.Metric
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
			result := parseChurnValue([]byte(tc.input))
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
		output := initializeAggregateOutput(time.Now())

		aggregateForPath("src/main.go", 10, 5, "Alice", testTime, output, time.Time{})

		stat := output.FileStats["src/main.go"]
		assert.Equal(t, schema.Metric(1), stat.Commits)
		assert.Equal(t, schema.Metric(15), stat.Churn)
		assert.Equal(t, schema.Metric(10), stat.LinesAdded)
		assert.Equal(t, schema.Metric(5), stat.LinesDeleted)
		assert.Equal(t, map[string]schema.Metric{"Alice": 1}, stat.Contributors)
		assert.Equal(t, testTime, stat.FirstCommit)
	})

	t.Run("multiple aggregations same file", func(t *testing.T) {
		output := initializeAggregateOutput(time.Now())

		// First aggregation
		aggregateForPath("src/main.go", 10, 5, "Alice", testTime, output, time.Time{})
		// Second aggregation
		aggregateForPath("src/main.go", 8, 2, "Alice", laterTime, output, time.Time{})

		stat := output.FileStats["src/main.go"]
		assert.Equal(t, schema.Metric(2), stat.Commits)
		assert.Equal(t, schema.Metric(25), stat.Churn)
		assert.Equal(t, map[string]schema.Metric{"Alice": 2}, stat.Contributors)
		assert.Equal(t, testTime, stat.FirstCommit) // Should keep earliest time
		assert.Equal(t, schema.Metric(18), stat.LinesAdded)
		assert.Equal(t, schema.Metric(7), stat.LinesDeleted)
		assert.Equal(t, schema.Metric(2), stat.RecentCommits)
		assert.Equal(t, schema.Metric(25), stat.RecentChurn)
		assert.Equal(t, schema.Metric(18), stat.RecentLinesAdded)
		assert.Equal(t, schema.Metric(7), stat.RecentLinesDeleted)
	})

	t.Run("multiple authors", func(t *testing.T) {
		output := initializeAggregateOutput(time.Now())

		aggregateForPath("src/main.go", 10, 5, "Alice", testTime, output, time.Time{})
		aggregateForPath("src/main.go", 5, 5, "Bob", laterTime, output, time.Time{})

		stat := output.FileStats["src/main.go"]
		assert.Equal(t, schema.Metric(2), stat.Commits)
		assert.Equal(t, schema.Metric(25), stat.Churn)
		assert.Equal(t, map[string]schema.Metric{"Alice": 1, "Bob": 1}, stat.Contributors)
	})

	t.Run("empty author", func(t *testing.T) {
		output := initializeAggregateOutput(time.Now())

		aggregateForPath("src/utils.go", 4, 4, "", laterTime, output, time.Time{})

		stat := output.FileStats["src/utils.go"]
		assert.Equal(t, schema.Metric(1), stat.Commits)
		assert.Equal(t, schema.Metric(8), stat.Churn)
		assert.Empty(t, stat.Contributors) // No contributors added
	})

	t.Run("zero time", func(t *testing.T) {
		output := initializeAggregateOutput(time.Now())

		aggregateForPath("src/zero.go", 1, 2, "Charlie", time.Time{}, output, time.Time{})

		stat := output.FileStats["src/zero.go"]
		assert.Equal(t, schema.Metric(1), stat.Commits)
		assert.Equal(t, schema.Metric(3), stat.Churn)
		assert.Equal(t, map[string]schema.Metric{"Charlie": 1}, stat.Contributors)
		assert.True(t, stat.FirstCommit.IsZero()) // No time recorded
	})
}
