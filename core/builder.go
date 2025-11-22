package core

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// FileResultBuilder builds the file metric from Git output.
type FileResultBuilder struct {
	cfg    *contract.Config
	git    contract.GitClient
	result *schema.FileResult
	output *schema.AggregateOutput
	path   string
	ctx    context.Context

	// Internal data collected during the build process
	contribCount map[string]int
	totalCommits int
}

// NewFileMetricsBuilder is the starting point for building file metrics.
func NewFileMetricsBuilder(ctx context.Context, cfg *contract.Config, client contract.GitClient, path string, output *schema.AggregateOutput) *FileResultBuilder {
	return &FileResultBuilder{
		cfg:          cfg,
		git:          client,
		result:       &schema.FileResult{Path: path, Mode: cfg.Mode},
		output:       output,
		path:         path,
		ctx:          ctx,
		contribCount: make(map[string]int),
	}
}

// FetchAllGitMetrics populates basic metrics (commits, contributors) and churn from aggregated data or git log if follow is needed.
func (b *FileResultBuilder) FetchAllGitMetrics() *FileResultBuilder {
	path := b.path
	useFollow := shouldUseFollow(b.ctx)

	if !useFollow {
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
			useFollow,
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

	// First commit time is already collected during aggregation phase
	// No additional git calls needed! The reason is that finding the true
	// age of the file is very inefficient since we would need to run a git
	// command for each file with --follow which is very slow for large repos.
	// This means that the true age is not fully accurate but it's acceptable
	// for most use cases.

	return b
}

// FetchFileStats reads the file to populate SizeBytes and LinesOfCode (PLOC).
func (b *FileResultBuilder) FetchFileStats() *FileResultBuilder {
	fullPath := filepath.Join(b.cfg.RepoPath, b.path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return b
	}

	size := int64(len(content))
	lines := len(strings.Split(string(content), "\n"))

	b.result.SizeBytes = size
	b.result.LinesOfCode = lines

	return b
}

// CalculateDerivedMetrics computes metrics that depend on previously collected data.
func (b *FileResultBuilder) CalculateDerivedMetrics() *FileResultBuilder {
	// AgeDays
	if b.result.FirstCommit.IsZero() {
		b.result.AgeDays = 0
	} else {
		b.result.AgeDays = contract.CalculateDaysBetween(b.result.FirstCommit, time.Now())
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

	// Sort authors by commit count descending, then by author name ascending for stable ordering
	type authorCommits struct {
		author  string
		commits int
	}
	var authors []authorCommits
	for author, commits := range authorMap {
		authors = append(authors, authorCommits{author: author, commits: commits})
	}
	sort.Slice(authors, func(i, j int) bool {
		if authors[i].commits == authors[j].commits {
			return authors[i].author < authors[j].author
		}
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
	// Compute score for current mode
	b.result.ModeScore = computeScore(b.result, b.cfg.Mode, b.cfg.CustomWeights)

	// Compute scores and breakdowns for all modes
	b.result.AllScores = make(map[schema.ScoringMode]float64)
	b.result.AllBreakdowns = make(map[schema.ScoringMode]map[schema.BreakdownKey]float64)

	for _, mode := range []schema.ScoringMode{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.StaleMode} {
		mCopy := *b.result
		mCopy.Mode = mode
		score := computeScore(&mCopy, mode, b.cfg.CustomWeights)
		b.result.AllScores[mode] = score
		b.result.AllBreakdowns[mode] = make(map[schema.BreakdownKey]float64)
		maps.Copy(b.result.AllBreakdowns[mode], mCopy.ModeBreakdown)
	}

	// Ensure current mode's breakdown is included
	if b.result.ModeBreakdown != nil {
		b.result.AllBreakdowns[b.cfg.Mode] = make(map[schema.BreakdownKey]float64)
		maps.Copy(b.result.AllBreakdowns[b.cfg.Mode], b.result.ModeBreakdown)
	}

	return b
}

// Build finalizes the construction and returns the completed metrics object.
func (b *FileResultBuilder) Build() schema.FileResult {
	return *b.result
}
