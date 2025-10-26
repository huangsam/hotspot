package core

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// FileMetricsBuilder builds the file metric from Git output.
type FileMetricsBuilder struct {
	metrics   schema.FileMetrics
	cfg       *internal.Config
	path      string
	useFollow bool

	// Internal data collected during the build process
	contribCount map[string]int
	totalCommits int
}

// NewFileMetricsBuilder is the starting point for building file metrics.
func NewFileMetricsBuilder(cfg *internal.Config, path string, useFollow bool) *FileMetricsBuilder {
	return &FileMetricsBuilder{
		cfg:          cfg,
		path:         path,
		useFollow:    useFollow,
		metrics:      schema.FileMetrics{Path: path},
		contribCount: make(map[string]int),
	}
}

// fetchCommitHistory runs 'git log' and populates basic metrics and internal counts.
func (b *FileMetricsBuilder) fetchCommitHistory() *FileMetricsBuilder {
	repo := b.cfg.RepoPath
	args := []string{"log"}
	if b.useFollow {
		args = append(args, "--follow")
	}
	if !b.cfg.StartTime.IsZero() {
		args = append(args, "--since="+b.cfg.StartTime.Format(internal.DateTimeFormat))
	}
	args = append(args, "--pretty=format:%an,%ad", "--date=iso", "--", b.path)

	out, err := internal.RunGitCommand(repo, args...)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Failed to get commit history for %s. Commits will be zeroed.", b.path))
		return b
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var firstCommit time.Time

	for _, l := range lines {
		if l == "" {
			continue
		}
		parts := strings.SplitN(l, ",", 2)
		if len(parts) < 2 {
			continue
		}
		author := parts[0]
		dateStr := parts[1]
		date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}

		if (!b.cfg.StartTime.IsZero() && date.Before(b.cfg.StartTime)) ||
			(!b.cfg.EndTime.IsZero() && date.After(b.cfg.EndTime)) {
			continue
		}

		b.contribCount[author]++
		b.totalCommits++
		if firstCommit.IsZero() || date.Before(firstCommit) {
			firstCommit = date
		}
	}
	b.metrics.UniqueContributors = len(b.contribCount)
	b.metrics.Commits = b.totalCommits
	b.metrics.FirstCommit = firstCommit

	return b
}

// fetchFileSize runs os.Stat to get the file size.
func (b *FileMetricsBuilder) fetchFileSize() *FileMetricsBuilder {
	if info, err := os.Stat(filepath.Join(b.cfg.RepoPath, b.path)); err == nil {
		b.metrics.SizeBytes = info.Size()
	}
	return b
}

// calculateChurn runs 'git log --numstat' to get lines added/deleted.
func (b *FileMetricsBuilder) calculateChurn() *FileMetricsBuilder {
	churnArgs := []string{"log"}
	if b.useFollow {
		churnArgs = append(churnArgs, "--follow")
	}
	if !b.cfg.StartTime.IsZero() {
		churnArgs = append(churnArgs, "--since="+b.cfg.StartTime.Format(internal.DateTimeFormat))
	}
	churnArgs = append(churnArgs, "--numstat", "--", b.path)

	out, err := internal.RunGitCommand(b.cfg.RepoPath, churnArgs...)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Failed to get churn data for %s.", b.path))
		return b
	}

	reader := csv.NewReader(bytes.NewReader(out))
	reader.Comma = '\t'
	totalChanges := 0
	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}
		if len(rec) >= 2 {
			add, _ := strconv.Atoi(rec[0])
			del, _ := strconv.Atoi(rec[1])
			totalChanges += add + del
		}
	}
	b.metrics.Churn = totalChanges + b.totalCommits

	return b
}

// calculateDerivedMetrics computes metrics that depend on previously collected data.
func (b *FileMetricsBuilder) calculateDerivedMetrics() *FileMetricsBuilder {
	// AgeDays
	if b.metrics.FirstCommit.IsZero() {
		b.metrics.AgeDays = 0
	} else {
		b.metrics.AgeDays = int(time.Since(b.metrics.FirstCommit).Hours() / 24)
	}

	// Gini coefficient for author diversity
	values := make([]float64, 0, len(b.contribCount))
	for _, c := range b.contribCount {
		values = append(values, float64(c))
	}
	b.metrics.Gini = gini(values) // Assuming gini() is a helper function

	return b
}

// applyGlobalMaps populates recent metrics from global maps if available.
func (b *FileMetricsBuilder) applyGlobalMaps() *FileMetricsBuilder {
	if recentCommitsMapGlobal := schema.GetRecentCommitsMapGlobal(); recentCommitsMapGlobal != nil {
		if v, ok := recentCommitsMapGlobal[b.path]; ok {
			b.metrics.RecentCommits = v
		}
	}
	if recentChurnMapGlobal := schema.GetRecentChurnMapGlobal(); recentChurnMapGlobal != nil {
		if v, ok := recentChurnMapGlobal[b.path]; ok {
			b.metrics.RecentChurn = v
		}
	}
	if recentContribMapGlobal := schema.GetRecentContribMapGlobal(); recentContribMapGlobal != nil {
		if m, ok := recentContribMapGlobal[b.path]; ok {
			b.metrics.RecentContributors = len(m)
		}
	}
	return b
}

// calculateScore computes the final composite score.
func (b *FileMetricsBuilder) calculateScore() *FileMetricsBuilder {
	b.metrics.Score = computeScore(&b.metrics, b.cfg.Mode) // Assuming computeScore() is a helper function
	return b
}

// Build finalizes the construction and returns the completed metrics object.
func (b *FileMetricsBuilder) Build() schema.FileMetrics {
	return b.metrics
}
