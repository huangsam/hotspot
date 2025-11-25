package core

import (
	"context"
	"testing"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/internal/iocache"
	"github.com/huangsam/hotspot/schema"
	"github.com/stretchr/testify/assert"
)

func TestCheckResultBuilder_ValidatePrerequisites_NoFilesChanged(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		CompareMode: true,
		BaseRef:     "main",
		TargetRef:   "feature",
		Excludes:    []string{},
	}

	mockGitClient := &contract.MockGitClient{}
	mockGitClient.On("GetChangedFilesBetweenRefs", ctx, "/test/repo", "main", "feature").Return([]string{}, nil)

	builder := NewCheckResultBuilder(ctx, cfg, &iocache.MockCacheManager{})
	builder.client = mockGitClient

	_, err := builder.ValidatePrerequisites()
	assert.NoError(t, err)
	assert.NotNil(t, builder.GetResult())
	assert.True(t, builder.GetResult().Passed)
}

func TestCheckResultBuilder_ValidatePrerequisites_AllFilesExcluded(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		CompareMode: true,
		BaseRef:     "main",
		TargetRef:   "feature",
		Excludes:    []string{"*.go"},
	}

	mockGitClient := &contract.MockGitClient{}
	mockGitClient.On("GetChangedFilesBetweenRefs", ctx, "/test/repo", "main", "feature").Return([]string{"main.go", "test.go"}, nil)

	builder := NewCheckResultBuilder(ctx, cfg, &iocache.MockCacheManager{})
	builder.client = mockGitClient

	_, err := builder.ValidatePrerequisites()
	assert.NoError(t, err)
	assert.NotNil(t, builder.GetResult())
	assert.True(t, builder.GetResult().Passed)
	assert.Equal(t, 0, len(builder.filesToAnalyze))
}

func TestCheckResultBuilder_ValidatePrerequisites_WithValidFiles(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		CompareMode: true,
		BaseRef:     "main",
		TargetRef:   "feature",
		Excludes:    []string{"*.md"},
	}

	mockGitClient := &contract.MockGitClient{}
	mockGitClient.On("GetChangedFilesBetweenRefs", ctx, "/test/repo", "main", "feature").Return([]string{"main.go", "README.md"}, nil)

	builder := NewCheckResultBuilder(ctx, cfg, &iocache.MockCacheManager{})
	builder.client = mockGitClient

	_, err := builder.ValidatePrerequisites()
	assert.NoError(t, err)
	assert.Nil(t, builder.GetResult()) // Should not be set for valid files
	assert.Equal(t, []string{"main.go"}, builder.filesToAnalyze)
}

func TestCheckResultBuilder_ValidatePrerequisites_MissingCompareMode(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		CompareMode: false,
	}

	builder := NewCheckResultBuilder(ctx, cfg, &iocache.MockCacheManager{})

	_, err := builder.ValidatePrerequisites()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base-ref and --target-ref")
}

func TestCheckResultBuilder_PrepareAnalysisConfig(t *testing.T) {
	ctx := context.Background()
	cfg := &contract.Config{
		RepoPath:    "/test/repo",
		CompareMode: true,
		BaseRef:     "main",
		TargetRef:   "feature",
		Lookback:    30 * 24 * time.Hour,
	}

	targetTime := time.Now()
	mockGitClient := &contract.MockGitClient{}
	mockGitClient.On("GetCommitTime", ctx, "/test/repo", "feature").Return(targetTime, nil)

	builder := NewCheckResultBuilder(ctx, cfg, &iocache.MockCacheManager{})
	builder.client = mockGitClient

	_, err := builder.PrepareAnalysisConfig()
	assert.NoError(t, err)
	assert.NotNil(t, builder.cfgTarget)
	assert.Equal(t, targetTime.Add(-30*24*time.Hour), builder.cfgTarget.StartTime)
	assert.Equal(t, targetTime, builder.cfgTarget.EndTime)
}

func TestCheckResultBuilder_ComputeMetrics(t *testing.T) {
	builder := &CheckResultBuilder{}

	// Create mock file results with different scores
	fileResults := []schema.FileResult{
		{
			Path: "file1.go",
			AllScores: map[schema.ScoringMode]float64{
				schema.HotMode:        60.0,
				schema.RiskMode:       40.0,
				schema.ComplexityMode: 30.0,
				schema.StaleMode:      20.0,
			},
			Owners: []string{"alice"},
		},
		{
			Path: "file2.go",
			AllScores: map[schema.ScoringMode]float64{
				schema.HotMode:        50.0,
				schema.RiskMode:       70.0,
				schema.ComplexityMode: 25.0,
				schema.StaleMode:      15.0,
			},
			Owners: []string{"bob"},
		},
	}

	cfg := &contract.Config{
		RiskThresholds: map[schema.ScoringMode]float64{
			schema.HotMode:        55.0,
			schema.RiskMode:       45.0,
			schema.ComplexityMode: 35.0,
			schema.StaleMode:      25.0,
		},
	}

	builder.fileResults = fileResults
	builder.cfg = cfg

	builder.ComputeMetrics()

	// Check max scores
	assert.Equal(t, 60.0, builder.maxScores[schema.HotMode])
	assert.Equal(t, 70.0, builder.maxScores[schema.RiskMode])
	assert.Equal(t, 30.0, builder.maxScores[schema.ComplexityMode])
	assert.Equal(t, 20.0, builder.maxScores[schema.StaleMode])

	// Check avg scores
	assert.Equal(t, 55.0, builder.avgScores[schema.HotMode])        // (60+50)/2
	assert.Equal(t, 55.0, builder.avgScores[schema.RiskMode])       // (40+70)/2
	assert.Equal(t, 27.5, builder.avgScores[schema.ComplexityMode]) // (30+25)/2
	assert.Equal(t, 17.5, builder.avgScores[schema.StaleMode])      // (20+15)/2

	// Check max score files
	assert.Equal(t, "file1.go", builder.maxScoreFiles[schema.HotMode][0].Path)
	assert.Equal(t, "file2.go", builder.maxScoreFiles[schema.RiskMode][0].Path)

	// Check failed files
	assert.Len(t, builder.failedFiles, 2) // file1 hot, file2 risk

	expectedFailures := []schema.CheckFailedFile{
		{Path: "file1.go", Mode: schema.HotMode, Score: 60.0, Threshold: 55.0},
		{Path: "file2.go", Mode: schema.RiskMode, Score: 70.0, Threshold: 45.0},
	}

	for _, expected := range expectedFailures {
		found := false
		for _, actual := range builder.failedFiles {
			if actual.Path == expected.Path && actual.Mode == expected.Mode {
				assert.Equal(t, expected.Score, actual.Score)
				assert.Equal(t, expected.Threshold, actual.Threshold)
				found = true
				break
			}
		}
		assert.True(t, found, "Expected failure not found: %+v", expected)
	}
}

func TestCheckResultBuilder_BuildResult_Success(t *testing.T) {
	builder := &CheckResultBuilder{
		filesToAnalyze: []string{"file1.go", "file2.go"},
		cfg: &contract.Config{
			BaseRef:   "main",
			TargetRef: "feature",
			Lookback:  30 * 24 * time.Hour,
			RiskThresholds: map[schema.ScoringMode]float64{
				schema.HotMode:        50.0,
				schema.RiskMode:       50.0,
				schema.ComplexityMode: 50.0,
				schema.StaleMode:      50.0,
			},
		},
		failedFiles: []schema.CheckFailedFile{}, // No failures
		maxScores: map[schema.ScoringMode]float64{
			schema.HotMode:        40.0,
			schema.RiskMode:       30.0,
			schema.ComplexityMode: 20.0,
			schema.StaleMode:      10.0,
		},
		maxScoreFiles: map[schema.ScoringMode][]schema.CheckMaxScoreFile{
			schema.HotMode: {{Path: "file1.go", Owners: []string{"alice"}}},
		},
		avgScores: map[schema.ScoringMode]float64{
			schema.HotMode:        35.0,
			schema.RiskMode:       25.0,
			schema.ComplexityMode: 15.0,
			schema.StaleMode:      5.0,
		},
	}

	builder.BuildResult()

	result := builder.GetResult()
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Len(t, result.FailedFiles, 0)
	assert.Equal(t, 2, result.TotalFiles)
	assert.Equal(t, "main", result.BaseRef)
	assert.Equal(t, "feature", result.TargetRef)
	assert.Equal(t, 30*24*time.Hour, result.Lookback)
	assert.Equal(t, 35.0, result.AvgScores[schema.HotMode])
}

func TestCheckResultBuilder_BuildResult_Failure(t *testing.T) {
	builder := &CheckResultBuilder{
		filesToAnalyze: []string{"file1.go"},
		cfg: &contract.Config{
			BaseRef:   "main",
			TargetRef: "feature",
			Lookback:  30 * 24 * time.Hour,
			RiskThresholds: map[schema.ScoringMode]float64{
				schema.HotMode: 50.0,
			},
		},
		failedFiles: []schema.CheckFailedFile{
			{Path: "file1.go", Mode: schema.HotMode, Score: 60.0, Threshold: 50.0},
		},
		maxScores: map[schema.ScoringMode]float64{
			schema.HotMode: 60.0,
		},
		maxScoreFiles: map[schema.ScoringMode][]schema.CheckMaxScoreFile{
			schema.HotMode: {{Path: "file1.go", Owners: []string{"alice"}}},
		},
		avgScores: map[schema.ScoringMode]float64{
			schema.HotMode: 60.0,
		},
	}

	builder.BuildResult()

	result := builder.GetResult()
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
	assert.Len(t, result.FailedFiles, 1)
	assert.Equal(t, "file1.go", result.FailedFiles[0].Path)
	assert.Equal(t, schema.HotMode, result.FailedFiles[0].Mode)
	assert.Equal(t, 60.0, result.FailedFiles[0].Score)
	assert.Equal(t, 50.0, result.FailedFiles[0].Threshold)
	assert.Equal(t, 60.0, result.AvgScores[schema.HotMode])
}

func TestFilterChangedFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		excludes []string
		expected []string
	}{
		{
			name:     "no excludes",
			files:    []string{"main.go", "core/check.go", "README.md"},
			excludes: []string{},
			expected: []string{"main.go", "core/check.go", "README.md"},
		},
		{
			name:     "exclude by extension",
			files:    []string{"main.go", "core/check.go", "README.md"},
			excludes: []string{".md"},
			expected: []string{"main.go", "core/check.go"},
		},
		{
			name:     "exclude by directory",
			files:    []string{"main.go", "vendor/lib.go", "dist/app.js"},
			excludes: []string{"vendor/", "dist/"},
			expected: []string{"main.go"},
		},
		{
			name:     "all files excluded",
			files:    []string{"README.md", "LICENSE"},
			excludes: []string{".md", "LICENSE"},
			expected: []string{},
		},
		{
			name:     "no excludes (core version)",
			files:    []string{"main.go", "core/agg.go", "README.md"},
			excludes: []string{},
			expected: []string{"main.go", "core/agg.go", "README.md"},
		},
		{
			name:     "exclude vendor",
			files:    []string{"main.go", "vendor/lib.go", "core/agg.go"},
			excludes: []string{"vendor/"},
			expected: []string{"main.go", "core/agg.go"},
		},
		{
			name:     "exclude multiple patterns",
			files:    []string{"main.go", "vendor/lib.go", "test_main.go", "core/agg.go"},
			excludes: []string{"vendor/", "test_"},
			expected: []string{"main.go", "core/agg.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterChangedFiles(tt.files, tt.excludes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
