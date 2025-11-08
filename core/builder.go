package core

import (
	"bytes"
	"context"
	"maps"
	"os"
	"path/filepath"
	"sort"
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
	ctx       context.Context

	// Internal data collected during the build process
	contribCount map[string]int
	totalCommits int
}

// NewFileMetricsBuilder is the starting point for building file metrics.
func NewFileMetricsBuilder(ctx context.Context, cfg *internal.Config, client internal.GitClient, path string, output *schema.AggregateOutput, useFollow bool) *FileResultBuilder {
	return &FileResultBuilder{
		cfg:          cfg,
		git:          client,
		result:       &schema.FileResult{Path: path},
		output:       output,
		path:         path,
		useFollow:    useFollow,
		ctx:          ctx,
		contribCount: make(map[string]int),
	}
}

// FetchAllGitMetrics populates basic metrics (commits, contributors) and churn from aggregated data or git log if follow is needed.
func (b *FileResultBuilder) FetchAllGitMetrics() *FileResultBuilder {
	path := b.path

	if !b.useFollow {
		// Use aggregated data for initial analysis (no follow needed)
		if commits, ok := b.output.CommitMap[path]; ok {
			b.totalCommits = commits
			b.result.Commits = commits
		}
		if churn, ok := b.output.ChurnMap[path]; ok {
			b.result.Churn = churn
		}
		if contribMap, ok := b.output.ContribMap[path]; ok {
			b.contribCount = make(map[string]int)
			maps.Copy(b.contribCount, contribMap)
			b.result.UniqueContributors = len(b.contribCount)
		}
		if firstCommit, ok := b.output.FirstCommitMap[path]; ok {
			b.result.FirstCommit = firstCommit
		}
	} else {
		// For follow analysis, run git log with --follow to get complete history
		out, err := b.git.GetFileActivityLog(
			b.ctx,
			b.cfg.RepoPath,
			b.path,
			b.cfg.StartTime,
			b.cfg.EndTime,
			b.useFollow,
		)
		if err == nil {
			// Parse the output to populate metrics
			lines := strings.Split(string(out), "\n")
			var firstCommit time.Time
			totalChanges := 0
			authorCommits := make(map[string]int)

			for _, line := range lines {
				line = strings.Trim(line, " \t\r\n'")
				if strings.HasPrefix(line, "DELIMITER_COMMIT_START") {
					// Commit line: DELIMITER_COMMIT_STARTauthor|date
					metadata := line[len("DELIMITER_COMMIT_START"):]
					parts := strings.SplitN(metadata, "|", 2)
					if len(parts) == 2 {
						author := strings.TrimSpace(parts[0])
						dateStr := strings.TrimSpace(parts[1])
						authorCommits[author]++
						b.totalCommits++
						if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
							if firstCommit.IsZero() || date.Before(firstCommit) {
								firstCommit = date
							}
						}
					}
				} else if parts := strings.Split(line, "\t"); len(parts) >= 3 {
					// Numstat line
					if add, errA := strconv.Atoi(strings.TrimSpace(parts[0])); errA == nil {
						if del, errD := strconv.Atoi(strings.TrimSpace(parts[1])); errD == nil {
							totalChanges += add + del
						}
					}
				}
			}

			b.contribCount = authorCommits
			b.result.UniqueContributors = len(b.contribCount)
			b.result.Commits = b.totalCommits
			b.result.Churn = totalChanges
			b.result.FirstCommit = firstCommit
		}
	}

	// Always get the absolute first commit date for accurate age calculation
	absFirst, absErr := b.git.GetFileFirstCommitTime(b.ctx, b.cfg.RepoPath, b.path, true)
	if absErr == nil {
		b.result.FirstCommit = absFirst
	}

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

	// Sort authors by commit count descending
	type authorCommits struct {
		author  string
		commits int
	}
	var authors []authorCommits
	for author, commits := range authorMap {
		authors = append(authors, authorCommits{author: author, commits: commits})
	}
	sort.Slice(authors, func(i, j int) bool {
		return authors[i].commits > authors[j].commits
	})

	// Set top owner and top 2 owners
	if len(authors) > 0 {
		b.result.Owners = make([]string, 0, 2)
		for i := 0; i < len(authors) && i < 2; i++ {
			b.result.Owners = append(b.result.Owners, authors[i].author)
		}
	}
	return b
}

// CalculateScore computes the final composite score.
func (b *FileResultBuilder) CalculateScore() *FileResultBuilder {
	b.result.Score = computeScore(b.result, b.cfg.Mode, b.cfg.CustomWeights) // Assuming computeScore() is a helper function
	return b
}

// Build finalizes the construction and returns the completed metrics object.
func (b *FileResultBuilder) Build() schema.FileResult {
	return *b.result
}
