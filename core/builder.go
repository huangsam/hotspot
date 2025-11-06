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

// FileResultBuilder builds the file metric from Git output.
type FileResultBuilder struct {
	cfg       *internal.Config
	git       internal.GitClient
	result    *schema.FileResult
	output    *schema.AggregateOutput
	path      string
	useFollow bool

	// Internal data collected during the build process
	contribCount map[string]int
	totalCommits int
}

// NewFileMetricsBuilder is the starting point for building file metrics.
func NewFileMetricsBuilder(cfg *internal.Config, client internal.GitClient, path string, output *schema.AggregateOutput, useFollow bool) *FileResultBuilder {
	return &FileResultBuilder{
		cfg:          cfg,
		git:          client,
		result:       &schema.FileResult{Path: path},
		output:       output,
		path:         path,
		useFollow:    useFollow,
		contribCount: make(map[string]int),
	}
}

// FetchAllGitMetrics runs 'git log' once to populate basic metrics (commits, contributors) and churn.
func (b *FileResultBuilder) FetchAllGitMetrics() *FileResultBuilder {
	const CommitDelimiter = "DELIMITER_COMMIT_START"

	// --- Data Collection: Use the new explicit method ---
	// The GitClient is now b.git (since it's an embedded/member field)
	out, err := b.git.GetFileActivityLog(
		b.cfg.RepoPath,
		b.path,
		b.cfg.StartTime,
		b.cfg.EndTime,
		b.useFollow,
	)
	if err != nil {
		internal.LogWarning(fmt.Sprintf("Failed to get metrics for %s. Error: %v", b.path, err))
		return b
	}

	// --- Parsing Logic (Remains identical) ---
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var firstCommit time.Time
	totalChanges := 0

	// ... (rest of the detailed parsing loop for Commit Metadata and Churn Data remains the same) ...
	for scanner.Scan() {
		// ... (Parsing logic from original function) ...
		line := scanner.Text()

		// 1. Process Commit Metadata
		if after, ok := strings.CutPrefix(line, CommitDelimiter); ok {
			// ... (Metadata parsing logic) ...
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
			continue
		}

		// 2. Process Churn Data (Numstat Line)
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			addStr := strings.TrimSpace(parts[0])
			delStr := strings.TrimSpace(parts[1])

			add, errA := strconv.Atoi(addStr)
			del, errD := strconv.Atoi(delStr)

			if errA == nil && errD == nil {
				totalChanges += add + del
			}
		}
	}

	// Finalize metrics after the loop
	b.result.UniqueContributors = len(b.contribCount)
	b.result.Commits = b.totalCommits
	b.result.FirstCommit = firstCommit
	b.result.Churn = totalChanges

	return b
}

// FetchFileStats reads the file once to populate SizeBytes and LinesOfCode (PLOC).
func (b *FileResultBuilder) FetchFileStats() *FileResultBuilder {
	fullPath := filepath.Join(b.cfg.RepoPath, b.path)

	// 1. Read the entire file content as a byte slice. This is the main disk I/O.
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return b
	}

	// 2. Get Size from the already-read byte slice length (instant).
	b.result.SizeBytes = int64(len(content))

	// 3. Count the number of newline characters (extremely fast byte operation).
	lineCount := bytes.Count(content, []byte{'\n'})
	b.result.LinesOfCode = lineCount

	return b
}

// CalculateDerivedMetrics computes metrics that depend on previously collected data.
func (b *FileResultBuilder) CalculateDerivedMetrics() *FileResultBuilder {
	// AgeDays
	if b.result.FirstCommit.IsZero() {
		b.result.AgeDays = 0
	} else {
		b.result.AgeDays = int(time.Since(b.result.FirstCommit).Hours() / 24)
	}

	// Gini coefficient for author diversity
	values := make([]float64, 0, len(b.contribCount))
	for _, c := range b.contribCount {
		values = append(values, float64(c))
	}
	b.result.Gini = gini(values) // Assuming gini() is a helper function

	return b
}

// FetchRecentInfo populates recent metrics from recent info if available.
func (b *FileResultBuilder) FetchRecentInfo() *FileResultBuilder {
	if v, ok := b.output.CommitMap[b.path]; ok {
		b.result.RecentCommits = v
	}
	if v, ok := b.output.ChurnMap[b.path]; ok {
		b.result.RecentChurn = v
	}
	if m, ok := b.output.ContribMap[b.path]; ok {
		b.result.RecentContributors = len(m)
	}
	return b
}

// CalculateOwner identifies the owner based on commit volume.
func (b *FileResultBuilder) CalculateOwner() *FileResultBuilder {
	authorMap, ok := b.output.ContribMap[b.path]
	if !ok || len(authorMap) == 0 {
		return b
	}

	var owner string
	var maxCommits int

	for author, commits := range authorMap {
		if maxCommits < commits {
			maxCommits = commits
			owner = author
		}
	}

	b.result.Owner = owner
	return b
}

// CalculateScore computes the final composite score.
func (b *FileResultBuilder) CalculateScore() *FileResultBuilder {
	b.result.Score = computeScore(b.result, b.cfg.Mode) // Assuming computeScore() is a helper function
	return b
}

// Build finalizes the construction and returns the completed metrics object.
func (b *FileResultBuilder) Build() schema.FileResult {
	return *b.result
}
