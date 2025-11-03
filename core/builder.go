package core

import (
	"bufio"
	"bytes"
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
	cfg       *internal.Config
	metrics   *schema.FileMetrics
	output    *schema.AggregateOutput
	path      string
	useFollow bool

	// Internal data collected during the build process
	contribCount map[string]int
	totalCommits int
}

// NewFileMetricsBuilder is the starting point for building file metrics.
func NewFileMetricsBuilder(cfg *internal.Config, path string, output *schema.AggregateOutput, useFollow bool) *FileMetricsBuilder {
	return &FileMetricsBuilder{
		cfg:          cfg,
		metrics:      &schema.FileMetrics{Path: path},
		output:       output,
		path:         path,
		useFollow:    useFollow,
		contribCount: make(map[string]int),
	}
}

// FetchCommitHistory runs 'git log' and populates basic metrics and internal counts.
func (b *FileMetricsBuilder) FetchCommitHistory() *FileMetricsBuilder {
	repo := b.cfg.RepoPath
	historyArgs := []string{"log"}
	if b.useFollow {
		historyArgs = append(historyArgs, "--follow")
	}
	if !b.cfg.StartTime.IsZero() {
		historyArgs = append(historyArgs, "--since="+b.cfg.StartTime.Format(internal.DateTimeFormat))
	}
	historyArgs = append(historyArgs, "--pretty=format:%an,%ad", "--date=iso", "--", b.path)

	out, err := internal.RunGitCommand(repo, historyArgs...)
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

// FetchFileSize runs os.Stat to get the file size.
func (b *FileMetricsBuilder) FetchFileSize() *FileMetricsBuilder {
	if info, err := os.Stat(filepath.Join(b.cfg.RepoPath, b.path)); err == nil {
		b.metrics.SizeBytes = info.Size()
	}
	return b
}

// FetchLinesOfCode reads the file and counts the Physical Lines of Code (PLOC)
// by counting the number of newline characters, which replicates wc -l behavior.
func (b *FileMetricsBuilder) FetchLinesOfCode() *FileMetricsBuilder {
	fullPath := filepath.Join(b.cfg.RepoPath, b.path)

	// 1. Read the entire file content as a byte slice.
	content, err := os.ReadFile(fullPath)
	if err != nil {
		b.metrics.LinesOfCode = 0
		return b
	}

	// 2. Count the number of newline characters directly in the byte slice.
	// This is extremely fast as it avoids any string conversion or line iteration logic.
	lineCount := bytes.Count(content, []byte{'\n'})

	// 3. Assign the count. This lineCount is identical to the output of `wc -l`.
	b.metrics.LinesOfCode = lineCount

	return b
}

// CalculateChurn runs 'git log --numstat' to get lines added/deleted.
func (b *FileMetricsBuilder) CalculateChurn() *FileMetricsBuilder {
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
		internal.LogWarning(fmt.Sprintf("Failed to get churn data for %s. Error: %v", b.path, err))
		return b
	}

	// Use bufio.Scanner to process the output line by line,
	// filtering out all header/commit information.
	scanner := bufio.NewScanner(bytes.NewReader(out))
	totalChanges := 0

	for scanner.Scan() {
		line := scanner.Text()
		// Numstat lines are always tab-separated and typically start with a number or '-'
		parts := strings.Split(line, "\t")

		// A valid numstat line has at least three parts (additions, deletions, filename)
		if len(parts) >= 3 {
			addStr := strings.TrimSpace(parts[0])
			delStr := strings.TrimSpace(parts[1])

			add, errA := strconv.Atoi(addStr)
			del, errD := strconv.Atoi(delStr)

			// Ignore lines that aren't valid numbers (like the 'total' summary if it exists)
			// Or cases where the file was binary (represented by '-')
			if errA == nil && errD == nil {
				totalChanges += add + del
			}
		}
	}

	// Set Churn to ONLY the total line changes
	b.metrics.Churn = totalChanges

	return b
}

// CalculateDerivedMetrics computes metrics that depend on previously collected data.
func (b *FileMetricsBuilder) CalculateDerivedMetrics() *FileMetricsBuilder {
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

// FetchRecentInfo populates recent metrics from recent info if available.
func (b *FileMetricsBuilder) FetchRecentInfo() *FileMetricsBuilder {
	if recentCommitsMapGlobal := b.output.CommitMap; recentCommitsMapGlobal != nil {
		if v, ok := recentCommitsMapGlobal[b.path]; ok {
			b.metrics.RecentCommits = v
		}
	}
	if recentChurnMapGlobal := b.output.ChurnMap; recentChurnMapGlobal != nil {
		if v, ok := recentChurnMapGlobal[b.path]; ok {
			b.metrics.RecentChurn = v
		}
	}
	if recentContribMapGlobal := b.output.ContribMap; recentContribMapGlobal != nil {
		if m, ok := recentContribMapGlobal[b.path]; ok {
			b.metrics.RecentContributors = len(m)
		}
	}
	return b
}

// CalculateOwner identifies the owner based on commit volume.
func (b *FileMetricsBuilder) CalculateOwner() *FileMetricsBuilder {
	if recentContribGlobal := b.output.ContribMap; recentContribGlobal != nil {
		var owner string
		var maxCommits int
		if authorMap := recentContribGlobal[b.path]; len(authorMap) > 0 {
			for author, commits := range authorMap {
				if maxCommits < commits {
					maxCommits = commits
					owner = author
				}
			}
		}
		b.metrics.Owner = owner
	}
	return b
}

// CalculateScore computes the final composite score.
func (b *FileMetricsBuilder) CalculateScore() *FileMetricsBuilder {
	b.metrics.Score = computeScore(b.metrics, b.cfg.Mode) // Assuming computeScore() is a helper function
	return b
}

// Build finalizes the construction and returns the completed metrics object.
func (b *FileMetricsBuilder) Build() schema.FileMetrics {
	return *b.metrics
}
