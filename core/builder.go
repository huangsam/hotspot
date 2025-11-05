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
	git       internal.GitClient
	metrics   *schema.FileMetrics
	output    *schema.AggregateOutput
	path      string
	useFollow bool

	// Internal data collected during the build process
	contribCount map[string]int
	totalCommits int
}

// NewFileMetricsBuilder is the starting point for building file metrics.
func NewFileMetricsBuilder(cfg *internal.Config, client internal.GitClient, path string, output *schema.AggregateOutput, useFollow bool) *FileMetricsBuilder {
	return &FileMetricsBuilder{
		cfg:          cfg,
		git:          client,
		metrics:      &schema.FileMetrics{Path: path},
		output:       output,
		path:         path,
		useFollow:    useFollow,
		contribCount: make(map[string]int),
	}
}

// FetchAllGitMetrics runs 'git log' once to populate basic metrics (commits, contributors) and churn.
func (b *FileMetricsBuilder) FetchAllGitMetrics() *FileMetricsBuilder {
	const CommitDelimiter = "DELIMITER_COMMIT_START"

	repo := b.cfg.RepoPath
	historyArgs := []string{"log"}

	// Follow files in case they have been renamed
	if b.useFollow {
		historyArgs = append(historyArgs, "--follow")
	}

	// Let Git handle the time filtering
	if !b.cfg.StartTime.IsZero() {
		historyArgs = append(historyArgs, "--since="+b.cfg.StartTime.Format(internal.DateTimeFormat))
	}
	if !b.cfg.EndTime.IsZero() {
		historyArgs = append(historyArgs, "--until="+b.cfg.EndTime.Format(internal.DateTimeFormat))
	}

	// Use the combined format: custom delimiter, author/date, and numstat
	historyArgs = append(historyArgs, "--pretty=format:"+CommitDelimiter+"%an,%ad", "--date=iso", "--numstat", "--", b.path)

	out, err := b.git.Run(repo, historyArgs...)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Failed to get metrics for %s. Error: %v", b.path, err))
		return b
	}

	// Use bufio.Scanner for efficient line-by-line processing
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var firstCommit time.Time
	totalChanges := 0

	for scanner.Scan() {
		line := scanner.Text()

		// 1. Process Commit Metadata
		if after, ok := strings.CutPrefix(line, CommitDelimiter); ok {
			// Trim the delimiter prefix
			metadata := after
			parts := strings.SplitN(metadata, ",", 2)
			if len(parts) < 2 {
				continue
			}
			author := parts[0]
			dateStr := parts[1]

			date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
			if err != nil {
				continue
			}

			// Populate commit history metrics
			b.contribCount[author]++
			b.totalCommits++
			if firstCommit.IsZero() || date.Before(firstCommit) {
				firstCommit = date
			}
			continue // Move to the next line (which should be numstat)
		}

		// 2. Process Churn Data (Numstat Line)
		// This part is unchanged and correctly processes lines added/deleted.
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			addStr := strings.TrimSpace(parts[0])
			delStr := strings.TrimSpace(parts[1])

			add, errA := strconv.Atoi(addStr)
			del, errD := strconv.Atoi(delStr)

			// Ignore binary files ('-') or other non-numeric lines
			if errA == nil && errD == nil {
				totalChanges += add + del
			}
		}
	}

	// Finalize metrics after the loop
	b.metrics.UniqueContributors = len(b.contribCount)
	b.metrics.Commits = b.totalCommits
	b.metrics.FirstCommit = firstCommit
	b.metrics.Churn = totalChanges

	return b
}

// FetchFileStats reads the file once to populate SizeBytes and LinesOfCode (PLOC).
func (b *FileMetricsBuilder) FetchFileStats() *FileMetricsBuilder {
	fullPath := filepath.Join(b.cfg.RepoPath, b.path)

	// 1. Read the entire file content as a byte slice. This is the main disk I/O.
	content, err := os.ReadFile(fullPath)
	if err != nil {
		b.metrics.LinesOfCode = 0

		// If file read fails (e.g., deleted file), we still try os.Stat
		// in case the error was transient or specific (though usually nil here).
		// For robustness, we try to grab the size from stat if content reading failed.
		if info, statErr := os.Stat(fullPath); statErr == nil {
			b.metrics.SizeBytes = info.Size()
		}
		return b
	}

	// 2. Get Size from the already-read byte slice length (instant).
	b.metrics.SizeBytes = int64(len(content))

	// 3. Count the number of newline characters (extremely fast byte operation).
	lineCount := bytes.Count(content, []byte{'\n'})
	b.metrics.LinesOfCode = lineCount

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
	if v, ok := b.output.CommitMap[b.path]; ok {
		b.metrics.RecentCommits = v
	}
	if v, ok := b.output.ChurnMap[b.path]; ok {
		b.metrics.RecentChurn = v
	}
	if m, ok := b.output.ContribMap[b.path]; ok {
		b.metrics.RecentContributors = len(m)
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
