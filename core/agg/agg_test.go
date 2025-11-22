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

// TestComputeFolderScore validates folder computation.
func TestComputeFolderScore(t *testing.T) {
	t.Run("divide by zero", func(t *testing.T) {
		results := &schema.FolderResult{
			Path:             ".",
			TotalLOC:         0,
			WeightedScoreSum: 100.0,
		}
		score := computeFolderScore(results)
		assert.Empty(t, score)
	})

	t.Run("valid calculation", func(t *testing.T) {
		results := &schema.FolderResult{
			Path:             ".",
			TotalLOC:         100,
			WeightedScoreSum: 92.0,
		}
		score := computeFolderScore(results)
		assert.InEpsilon(t, .92, score, 0.01)
	})
}

func TestAggregateAndScoreFolders(t *testing.T) {
	t.Run("basic aggregation", func(t *testing.T) {
		fileResults := []schema.FileResult{
			{
				Path:        "core/agg.go",
				ModeScore:   85.0,
				Commits:     10,
				Churn:       50,
				LinesOfCode: 100,
				Owners:      []string{"alice"},
			},
			{
				Path:        "core/analysis.go",
				ModeScore:   75.0,
				Commits:     8,
				Churn:       40,
				LinesOfCode: 80,
				Owners:      []string{"bob"},
			},
			{
				Path:        "core/main.go",
				ModeScore:   90.0,
				Commits:     15,
				Churn:       30,
				LinesOfCode: 50,
				Owners:      []string{"alice"},
			},
		}

		cfg := &contract.Config{
			PathFilter: "",
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		// Should have 1 folder: "core" (root files are skipped when PathFilter is empty)
		assert.Len(t, folders, 1)

		coreFolder := &folders[0]
		assert.Equal(t, "core", coreFolder.Path)
		assert.Equal(t, 33, coreFolder.Commits)   // 10 + 8 + 15
		assert.Equal(t, 120, coreFolder.Churn)    // 50 + 40 + 30
		assert.Equal(t, 230, coreFolder.TotalLOC) // 100 + 80 + 50
		// Score: (85.0*100 + 75.0*80 + 90.0*50) / (100+80+50) = (8500 + 6000 + 4500) / 230 = 19000/230 ≈ 82.61
		assert.InEpsilon(t, 82.61, coreFolder.Score, 0.01)
		assert.Equal(t, []string{"alice", "bob"}, coreFolder.Owners) // alice has most commits (10+15=25), then bob (8)
	})

	t.Run("owner calculation with multiple authors", func(t *testing.T) {
		fileResults := []schema.FileResult{
			{
				Path:        "src/file1.go",
				ModeScore:   80.0,
				Commits:     5,
				Churn:       25,
				LinesOfCode: 50,
				Owners:      []string{"alice"},
			},
			{
				Path:        "src/file2.go",
				ModeScore:   70.0,
				Commits:     10,
				Churn:       30,
				LinesOfCode: 60,
				Owners:      []string{"bob"},
			},
			{
				Path:        "src/file3.go",
				ModeScore:   75.0,
				Commits:     3,
				Churn:       15,
				LinesOfCode: 40,
				Owners:      []string{"alice"},
			},
		}

		cfg := &contract.Config{
			PathFilter: "",
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		assert.Len(t, folders, 1)
		folder := folders[0]

		assert.Equal(t, "src", folder.Path)
		assert.Equal(t, 18, folder.Commits)   // 5 + 10 + 3
		assert.Equal(t, 70, folder.Churn)     // 25 + 30 + 15
		assert.Equal(t, 150, folder.TotalLOC) // 50 + 60 + 40
		// Score: (80.0*50 + 70.0*60 + 75.0*40) / 150 = (4000 + 4200 + 3000) / 150 = 11200/150 ≈ 74.67
		assert.InEpsilon(t, 74.67, folder.Score, 0.01)
		// Owners: bob has most commits (10), then alice (5+3=8)
		assert.Equal(t, []string{"bob", "alice"}, folder.Owners)
	})

	t.Run("empty input", func(t *testing.T) {
		fileResults := []schema.FileResult{}

		cfg := &contract.Config{
			PathFilter: "",
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		assert.Len(t, folders, 0)
	})

	t.Run("single file", func(t *testing.T) {
		fileResults := []schema.FileResult{
			{
				Path:        "utils/helper.go",
				ModeScore:   65.0,
				Commits:     7,
				Churn:       20,
				LinesOfCode: 30,
				Owners:      []string{"charlie"},
			},
		}

		cfg := &contract.Config{
			PathFilter: "",
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		assert.Len(t, folders, 1)
		folder := folders[0]

		assert.Equal(t, "utils", folder.Path)
		assert.Equal(t, 7, folder.Commits)
		assert.Equal(t, 20, folder.Churn)
		assert.Equal(t, 30, folder.TotalLOC)
		assert.Equal(t, 65.0, folder.Score)
		assert.Equal(t, []string{"charlie"}, folder.Owners)
	})

	t.Run("files in root directory with path filter", func(t *testing.T) {
		fileResults := []schema.FileResult{
			{
				Path:        "main.go",
				ModeScore:   85.0,
				Commits:     12,
				Churn:       45,
				LinesOfCode: 75,
				Owners:      []string{"alice"},
			},
			{
				Path:        "README.md",
				ModeScore:   20.0,
				Commits:     2,
				Churn:       5,
				LinesOfCode: 25,
				Owners:      []string{"bob"},
			},
		}

		cfg := &contract.Config{
			PathFilter: "src/", // Non-empty enables root folder inclusion
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		// Root folder should be included since PathFilter is set
		assert.Len(t, folders, 1)
		rootFolder := folders[0]
		assert.Equal(t, ".", rootFolder.Path)
		assert.Equal(t, 14, rootFolder.Commits)   // 12 + 2
		assert.Equal(t, 50, rootFolder.Churn)     // 45 + 5
		assert.Equal(t, 100, rootFolder.TotalLOC) // 75 + 25
		// Score: (85.0*75 + 20.0*25) / 100 = (6375 + 500) / 100 = 6875/100 = 68.75
		assert.InEpsilon(t, 68.75, rootFolder.Score, 0.01)
		assert.Equal(t, []string{"alice", "bob"}, rootFolder.Owners) // alice has more commits (12 > 2)
	})

	t.Run("root folder included when path filter set", func(t *testing.T) {
		fileResults := []schema.FileResult{
			{
				Path:        "main.go",
				ModeScore:   85.0,
				Commits:     12,
				Churn:       45,
				LinesOfCode: 75,
				Owners:      []string{"alice"},
			},
			{
				Path:        "config.go",
				ModeScore:   70.0,
				Commits:     8,
				Churn:       25,
				LinesOfCode: 50,
				Owners:      []string{"bob"},
			},
		}

		cfg := &contract.Config{
			PathFilter: ".", // Non-empty enables root folder inclusion
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		// Root folder should be included when PathFilter is set
		assert.Len(t, folders, 1)
		rootFolder := folders[0]
		assert.Equal(t, ".", rootFolder.Path)
		assert.Equal(t, 20, rootFolder.Commits)   // 12 + 8
		assert.Equal(t, 70, rootFolder.Churn)     // 45 + 25
		assert.Equal(t, 125, rootFolder.TotalLOC) // 75 + 50
		// Score: (85.0*75 + 70.0*50) / 125 = (6375 + 3500) / 125 = 9875/125 = 79.0
		assert.InEpsilon(t, 79.0, rootFolder.Score, 0.01)
		assert.Equal(t, []string{"alice", "bob"}, rootFolder.Owners) // alice has more commits (12 > 8)
	})

	t.Run("mixed file sizes for weighted scoring", func(t *testing.T) {
		fileResults := []schema.FileResult{
			{
				Path:        "pkg/small.go",
				ModeScore:   90.0,
				Commits:     5,
				Churn:       10,
				LinesOfCode: 20, // Small file
				Owners:      []string{"alice"},
			},
			{
				Path:        "pkg/large.go",
				ModeScore:   70.0,
				Commits:     15,
				Churn:       50,
				LinesOfCode: 200, // Large file
				Owners:      []string{"bob"},
			},
		}

		cfg := &contract.Config{
			PathFilter: "",
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		assert.Len(t, folders, 1)
		folder := folders[0]

		assert.Equal(t, "pkg", folder.Path)
		assert.Equal(t, 20, folder.Commits)   // 5 + 15
		assert.Equal(t, 60, folder.Churn)     // 10 + 50
		assert.Equal(t, 220, folder.TotalLOC) // 20 + 200
		// Score: (90.0*20 + 70.0*200) / 220 = (1800 + 14000) / 220 = 15800/220 ≈ 71.82
		// Large file (70.0) weighted more heavily than small file (90.0)
		assert.InEpsilon(t, 71.82, folder.Score, 0.01)
		assert.Equal(t, []string{"bob", "alice"}, folder.Owners) // bob has more commits (15 > 5)
	})

	t.Run("no owners in files", func(t *testing.T) {
		fileResults := []schema.FileResult{
			{
				Path:        "lib/utils.go",
				ModeScore:   75.0,
				Commits:     8,
				Churn:       25,
				LinesOfCode: 45,
				Owners:      []string{}, // No owners
			},
			{
				Path:        "lib/helpers.go",
				ModeScore:   80.0,
				Commits:     6,
				Churn:       20,
				LinesOfCode: 35,
				Owners:      []string{}, // No owners
			},
		}

		cfg := &contract.Config{
			PathFilter: "",
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		assert.Len(t, folders, 1)
		folder := folders[0]

		assert.Equal(t, "lib", folder.Path)
		assert.Equal(t, 14, folder.Commits)  // 8 + 6
		assert.Equal(t, 45, folder.Churn)    // 25 + 20
		assert.Equal(t, 80, folder.TotalLOC) // 45 + 35
		// Score: (75.0*45 + 80.0*35) / 80 = (3375 + 2800) / 80 = 6175/80 = 77.19
		assert.InEpsilon(t, 77.19, folder.Score, 0.01)
		assert.Empty(t, folder.Owners) // No owners available
	})

	t.Run("multiple folders with different structures", func(t *testing.T) {
		fileResults := []schema.FileResult{
			// api folder
			{
				Path:        "api/handlers.go",
				ModeScore:   85.0,
				Commits:     12,
				Churn:       40,
				LinesOfCode: 100,
				Owners:      []string{"alice"},
			},
			{
				Path:        "api/middleware.go",
				ModeScore:   75.0,
				Commits:     8,
				Churn:       25,
				LinesOfCode: 80,
				Owners:      []string{"bob"},
			},
			// db folder
			{
				Path:        "db/models.go",
				ModeScore:   80.0,
				Commits:     10,
				Churn:       35,
				LinesOfCode: 90,
				Owners:      []string{"charlie"},
			},
			// root file
			{
				Path:        "config.go",
				ModeScore:   70.0,
				Commits:     5,
				Churn:       15,
				LinesOfCode: 40,
				Owners:      []string{"alice"},
			},
		}

		cfg := &contract.Config{
			PathFilter: "",
			Mode:       schema.HotMode,
		}

		folders := AggregateAndScoreFolders(cfg, fileResults)

		assert.Len(t, folders, 2) // api and db (root excluded when PathFilter is empty)

		// Create a map for easier testing
		folderMap := make(map[string]*schema.FolderResult)
		for i := range folders {
			folderMap[folders[i].Path] = &folders[i]
		}

		// Test api folder
		apiFolder := folderMap["api"]
		assert.NotNil(t, apiFolder)
		assert.Equal(t, 20, apiFolder.Commits)           // 12 + 8
		assert.Equal(t, 65, apiFolder.Churn)             // 40 + 25
		assert.Equal(t, 180, apiFolder.TotalLOC)         // 100 + 80
		assert.InEpsilon(t, 80.56, apiFolder.Score, 0.1) // (85*100 + 75*80) / 180
		assert.Equal(t, []string{"alice", "bob"}, apiFolder.Owners)

		// Test db folder
		dbFolder := folderMap["db"]
		assert.NotNil(t, dbFolder)
		assert.Equal(t, 10, dbFolder.Commits)
		assert.Equal(t, 35, dbFolder.Churn)
		assert.Equal(t, 90, dbFolder.TotalLOC)
		assert.Equal(t, 80.0, dbFolder.Score)
		assert.Equal(t, []string{"charlie"}, dbFolder.Owners)

		// Root folder should not be present when PathFilter is empty
		assert.Nil(t, folderMap["."])
	})
}
