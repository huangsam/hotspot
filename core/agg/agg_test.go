package agg

import (
	"context"
	_ "embed"
	"strings"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

//go:embed testdata/git_log_basic.txt
var gitLogBasicFixture []byte

//go:embed testdata/file_list.txt
var fileListFixture string

func TestAggregateActivity(t *testing.T) {
	ctx := context.Background()

	// Create mock client
	mockClient := &contract.MockGitClient{}

	// Setup expectations
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return(strings.Split(strings.TrimSpace(fileListFixture), "\n"), nil)
	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(gitLogBasicFixture, nil)

	// Create config
	cfg := &contract.Config{
		RepoPath:  "/test/repo",
		StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	// Execute
	output, err := aggregateActivity(ctx, cfg, mockClient)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)

	// Verify commit counts (number of commits affecting each file)
	assert.Equal(t, 1, output.CommitMap["AGENTS.md"])
	assert.Equal(t, 2, output.CommitMap["integration/verification_test.go"])
	assert.Equal(t, 2, output.CommitMap["core/core.go"])
	assert.Equal(t, 2, output.CommitMap["internal/configs.go"])
	assert.Equal(t, 1, output.CommitMap["internal/output_timeseries.go"])
	assert.Equal(t, 1, output.CommitMap["internal/time.go"])
	assert.Equal(t, 1, output.CommitMap["internal/time_fuzz_test.go"])
	assert.Equal(t, 1, output.CommitMap["internal/time_test.go"])
	assert.Equal(t, 1, output.CommitMap["internal/writer_timeseries.go"])
	assert.Equal(t, 1, output.CommitMap["main.go"])
	assert.Equal(t, 1, output.CommitMap["schema/schema.go"])

	// Verify contributor counts (all commits are by "Samuel Huang")
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["AGENTS.md"])
	assert.Equal(t, map[string]int{"Samuel Huang": 2}, output.ContribMap["integration/verification_test.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 2}, output.ContribMap["core/core.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 2}, output.ContribMap["internal/configs.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["internal/output_timeseries.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["internal/time.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["internal/time_fuzz_test.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["internal/time_test.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["internal/writer_timeseries.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["main.go"])
	assert.Equal(t, map[string]int{"Samuel Huang": 1}, output.ContribMap["schema/schema.go"])

	// Verify churn counts (additions + deletions)
	assert.Equal(t, 272, output.ChurnMap["AGENTS.md"])
	assert.Equal(t, 125, output.ChurnMap["integration/verification_test.go"])
	assert.Equal(t, 120, output.ChurnMap["core/core.go"])
	assert.Equal(t, 31, output.ChurnMap["internal/configs.go"])
	assert.Equal(t, 116, output.ChurnMap["internal/output_timeseries.go"])
	assert.Equal(t, 4, output.ChurnMap["internal/time.go"])
	assert.Equal(t, 2, output.ChurnMap["internal/time_fuzz_test.go"])
	assert.Equal(t, 2, output.ChurnMap["internal/time_test.go"])
	assert.Equal(t, 44, output.ChurnMap["internal/writer_timeseries.go"])
	assert.Equal(t, 27, output.ChurnMap["main.go"])
	assert.Equal(t, 13, output.ChurnMap["schema/schema.go"])

	mockClient.AssertExpectations(t)
}

func TestBuildFilteredFileList(t *testing.T) {
	// Create sample aggregate output
	output := &schema.AggregateOutput{
		CommitMap: map[string]int{
			"main.go":          10,
			"core/agg.go":      5,
			"core/analysis.go": 8,
			"test_main.go":     3,
			"README.md":        2,
		},
		ChurnMap: map[string]int{
			"main.go":          50,
			"core/agg.go":      25,
			"core/analysis.go": 40,
			"test_main.go":     15,
			"README.md":        10,
		},
		ContribMap: map[string]map[string]int{
			"main.go":          {"alice": 8, "bob": 2},
			"core/agg.go":      {"alice": 3, "charlie": 2},
			"core/analysis.go": {"bob": 5, "charlie": 3},
			"test_main.go":     {"alice": 3},
			"README.md":        {"alice": 2},
		},
	}

	cfg := &contract.Config{
		PathFilter: "",
		Excludes:   []string{"test_*", "*.md"},
	}

	files := BuildFilteredFileList(cfg, output)

	// Should include main.go, core/agg.go, core/analysis.go
	// Should exclude test_main.go (matches test_*), README.md (matches *.md)
	assert.Len(t, files, 3)
	assert.Contains(t, files, "main.go")
	assert.Contains(t, files, "core/agg.go")
	assert.Contains(t, files, "core/analysis.go")
	assert.NotContains(t, files, "test_main.go")
	assert.NotContains(t, files, "README.md")
}

func TestBuildFilteredFileList_WithPathFilter(t *testing.T) {
	// Create sample aggregate output
	output := &schema.AggregateOutput{
		CommitMap: map[string]int{
			"main.go":          10,
			"core/agg.go":      5,
			"core/analysis.go": 8,
			"schema/types.go":  3,
		},
		ChurnMap: map[string]int{
			"main.go":          50,
			"core/agg.go":      25,
			"core/analysis.go": 40,
			"schema/types.go":  15,
		},
		ContribMap: map[string]map[string]int{
			"main.go":          {"alice": 8, "bob": 2},
			"core/agg.go":      {"alice": 3, "charlie": 2},
			"core/analysis.go": {"bob": 5, "charlie": 3},
			"schema/types.go":  {"alice": 3},
		},
	}

	cfg := &contract.Config{
		PathFilter: "core/",
		Excludes:   []string{},
	}

	files := BuildFilteredFileList(cfg, output)

	// Should only include files starting with "core/"
	assert.Len(t, files, 2)
	assert.Contains(t, files, "core/agg.go")
	assert.Contains(t, files, "core/analysis.go")
	assert.NotContains(t, files, "main.go")
	assert.NotContains(t, files, "schema/types.go")
}
