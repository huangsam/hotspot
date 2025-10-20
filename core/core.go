// Package core has core logic for analysis, scoring and ranking.
package core

import (
	"bytes"
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// AnalyzeRepo processes all files in parallel using a worker pool.
// It spawns cfg.Workers number of goroutines to analyze files concurrently
// and aggregates their results into a single slice of schema.FileMetrics.
func AnalyzeRepo(cfg *schema.Config, files []string) []schema.FileMetrics {
	// Filter files according to excludes and path filter before analysis
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if internal.ShouldIgnore(f, cfg.Excludes) {
			continue
		}
		filtered = append(filtered, f)
	}

	results := make([]schema.FileMetrics, 0, len(filtered))
	fileCh := make(chan string, len(filtered))
	resultCh := make(chan schema.FileMetrics, len(files))
	var wg sync.WaitGroup

	for range cfg.Workers {
		wg.Go(func() {
			for f := range fileCh {
				metrics := AnalyzeFileCommon(cfg, f, false)
				resultCh <- metrics
			}
		})
	}

	for _, f := range filtered {
		fileCh <- f
	}
	close(fileCh)

	wg.Wait()
	close(resultCh)

	for r := range resultCh {
		results = append(results, r)
	}

	return results
}

// AnalyzeFileCommon computes all metrics for a single file in the repository.
// It gathers Git history data (commits, authors, dates), file size, and calculates
// derived metrics like churn and the Gini coefficient of author contributions.
// The analysis is constrained by the time range in cfg if specified.
// If useFollow is true, git --follow is used to track file renames.
func AnalyzeFileCommon(cfg *schema.Config, path string, useFollow bool) schema.FileMetrics {
	repo := cfg.RepoPath
	var metrics schema.FileMetrics
	metrics.Path = path

	// Build git log command with optional --follow
	args := []string{"-C", repo, "log"}
	if useFollow {
		args = append(args, "--follow")
	}
	if !cfg.StartTime.IsZero() {
		args = append(args, "--since="+cfg.StartTime.Format(time.RFC3339))
	}
	args = append(args, "--pretty=format:%an,%ad", "--date=iso", "--", path)

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return metrics
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	contribCount := map[string]int{}
	totalCommits := 0
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
		date, _ := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if !cfg.StartTime.IsZero() && date.Before(cfg.StartTime) {
			continue
		}
		if !cfg.EndTime.IsZero() && date.After(cfg.EndTime) {
			continue
		}
		contribCount[author]++
		totalCommits++
		if firstCommit.IsZero() || date.Before(firstCommit) {
			firstCommit = date
		}
	}

	metrics.UniqueContributors = len(contribCount)
	metrics.Commits = totalCommits
	metrics.FirstCommit = firstCommit
	// If we couldn't determine a first commit (empty history or parsing issues),
	// set AgeDays to 0 instead of a huge value to avoid skewing the score.
	if firstCommit.IsZero() {
		metrics.AgeDays = 0
	} else {
		metrics.AgeDays = int(time.Since(firstCommit).Hours() / 24)
	}

	// File size
	info, err := os.Stat(filepath.Join(repo, path))
	if err == nil {
		metrics.SizeBytes = info.Size()
	}

	// Churn
	churnArgs := []string{"-C", repo, "log"}
	if useFollow {
		churnArgs = append(churnArgs, "--follow")
	}
	if !cfg.StartTime.IsZero() {
		churnArgs = append(churnArgs, "--since="+cfg.StartTime.Format(time.RFC3339))
	}
	churnArgs = append(churnArgs, "--numstat", "--", path)

	cmd = exec.Command("git", churnArgs...)
	out, _ = cmd.Output()
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
	metrics.Churn = totalChanges + totalCommits

	// If we have global recent maps (populated by a single repo-wide pass),
	// use them to populate recent metrics for this file without doing per-file
	// git --follow invocations.
	if recentCommitsMapGlobal := schema.GetRecentCommitsMapGlobal(); recentCommitsMapGlobal != nil {
		if v, ok := recentCommitsMapGlobal[path]; ok {
			metrics.RecentCommits = v
		}
	}
	if recentChurnMapGlobal := schema.GetRecentChurnMapGlobal(); recentChurnMapGlobal != nil {
		if v, ok := recentChurnMapGlobal[path]; ok {
			metrics.RecentChurn = v
		}
	}
	if recentContribMapGlobal := schema.GetRecentContribMapGlobal(); recentContribMapGlobal != nil {
		if m, ok := recentContribMapGlobal[path]; ok {
			metrics.RecentContributors = len(m)
		}
	}

	// Gini coefficient for author diversity
	values := make([]float64, 0, len(contribCount))
	for _, c := range contribCount {
		values = append(values, float64(c))
	}
	metrics.Gini = gini(values)

	// Composite score (0–100)
	metrics.Score = computeScore(&metrics, cfg.Mode)
	return metrics
}
