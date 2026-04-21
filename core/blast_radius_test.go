package core

import (
	"context"
	"testing"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetHotspotBlastRadiusResults(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}

	// Setup mock git log with co-changes
	// Commit 1: A and B changed together
	// Commit 2: A and C changed together
	// Commit 3: B and C changed together
	// Commit 4: A changed alone
	// Commit 5: B changed alone
	// Commit 6: C changed alone

	gitLog := "--hash1|author|2024-01-01 10:00:00 +0000\n" +
		"10\t5\tfile_a.go\n" +
		"20\t10\tfile_b.go\n" +
		"--hash2|author|2024-01-01 11:00:00 +0000\n" +
		"5\t2\tfile_a.go\n" +
		"15\t5\tfile_c.go\n" +
		"--hash3|author|2024-01-01 12:00:00 +0000\n" +
		"8\t3\tfile_b.go\n" +
		"12\t4\tfile_c.go\n" +
		"--hash4|author|2024-01-01 13:00:00 +0000\n" +
		"1\t1\tfile_a.go\n" +
		"--hash5|author|2024-01-01 14:00:00 +0000\n" +
		"1\t1\tfile_b.go\n" +
		"--hash6|author|2024-01-01 15:00:00 +0000\n" +
		"1\t1\tfile_c.go\n"

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
	}

	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return([]byte(gitLog), nil)
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return([]string{"file_a.go", "file_b.go", "file_c.go"}, nil)

	// Threshold 0.2 to catch all pairs:
	// A: 3 commits (1,2,4)
	// B: 3 commits (1,3,5)
	// C: 3 commits (2,3,6)
	// A&B: 1 co-change. Jaccard = 1 / (3+3-1) = 1/5 = 0.2
	// A&C: 1 co-change. Jaccard = 1 / (3+3-1) = 1/5 = 0.2
	// B&C: 1 co-change. Jaccard = 1 / (3+3-1) = 1/5 = 0.2

	result, err := GetHotspotBlastRadiusResults(ctx, cfg, mockClient, 10, 0.1)

	assert.NoError(t, err)
	assert.Len(t, result.Pairs, 3)
	assert.Equal(t, 6, result.Summary.TotalCommits)

	// Check one pair
	foundAB := false
	for _, p := range result.Pairs {
		if (p.Source == "file_a.go" && p.Target == "file_b.go") || (p.Source == "file_b.go" && p.Target == "file_a.go") {
			assert.InEpsilon(t, 0.2, p.Score, 0.0001)
			assert.Equal(t, 1, p.CoChange)
			foundAB = true
		}
	}
	assert.True(t, foundAB)
}

func TestJaccardWithHigherCoupling(t *testing.T) {
	ctx := context.Background()
	mockClient := &git.MockGitClient{}

	// Commit 1: A and B
	// Commit 2: A and B
	// Commit 3: A
	gitLog := "--hash1|author|2024-01-01 10:00:00 +0000\n" +
		"10\t5\tfile_a.go\n" +
		"20\t10\tfile_b.go\n" +
		"--hash2|author|2024-01-01 11:00:00 +0000\n" +
		"5\t2\tfile_a.go\n" +
		"15\t5\tfile_b.go\n" +
		"--hash3|author|2024-01-01 12:00:00 +0000\n" +
		"1\t1\tfile_a.go\n"

	cfg := &config.Config{
		Git: config.GitConfig{
			RepoPath: "/test/repo",
		},
	}

	mockClient.On("GetActivityLog", ctx, "/test/repo", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return([]byte(gitLog), nil)
	mockClient.On("ListFilesAtRef", ctx, "/test/repo", "HEAD").Return([]string{"file_a.go", "file_b.go"}, nil)

	// A: 3 commits
	// B: 2 commits
	// A&B: 2 co-changes
	// Score = 2 / (3 + 2 - 2) = 2/3 = 0.666...
	result, err := GetHotspotBlastRadiusResults(ctx, cfg, mockClient, 10, 0.5)

	assert.NoError(t, err)
	assert.Len(t, result.Pairs, 1)
	assert.InEpsilon(t, 0.6666, result.Pairs[0].Score, 0.001)
}
