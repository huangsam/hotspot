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

	"github.com/huangsam/hotspot/core/algo"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/schema"
)

// FileResultBuilder builds the file metric from Git output.
type FileResultBuilder struct {
	gitSettings     config.GitSettings
	scoringSettings config.ScoringSettings
	git             git.Client
	result          *schema.FileResult
	output          *schema.AggregateOutput
	path            string
	ctx             context.Context

	// Internal data collected during the build process
	contribCount map[string]schema.Metric
	totalCommits schema.Metric
}

// NewFileMetricsBuilder is the starting point for building file metrics.
func NewFileMetricsBuilder(
	ctx context.Context,
	gitSettings config.GitSettings,
	scoringSettings config.ScoringSettings,
	client git.Client,
	path string,
	output *schema.AggregateOutput,
) *FileResultBuilder {
	return &FileResultBuilder{
		gitSettings:     gitSettings,
		scoringSettings: scoringSettings,
		git:             client,
		result:          &schema.FileResult{Path: path, Mode: scoringSettings.GetMode()},
		output:          output,
		path:            path,
		ctx:             ctx,
		contribCount:    make(map[string]schema.Metric),
	}
}

// FetchAllGitMetrics populates basic metrics (commits, contributors) and churn from aggregated data or git log if follow is needed.
func (b *FileResultBuilder) FetchAllGitMetrics() *FileResultBuilder {
	path := b.path
	useFollow := shouldUseFollow(b.ctx)

	if !useFollow {
		// Use aggregated data for initial analysis (no follow needed)
		if stat, ok := b.output.FileStats[path]; ok {
			b.totalCommits = stat.Commits
			b.result.Commits = stat.Commits
			b.result.Churn = stat.Churn
			b.result.LinesAdded = stat.LinesAdded
			b.result.LinesDeleted = stat.LinesDeleted
			b.result.DecayedCommits = stat.DecayedCommits
			b.result.DecayedChurn = stat.DecayedChurn
			b.result.FirstCommit = stat.FirstCommit

			if len(stat.Contributors) > 0 {
				b.contribCount = make(map[string]schema.Metric)
				maps.Copy(b.contribCount, stat.Contributors)
				b.result.UniqueContributors = schema.Metric(len(b.contribCount))
			}
		}
	} else {
		// For follow analysis, run git log with --follow to get complete history
		out, err := b.git.GetFileActivityLog(
			b.ctx,
			b.gitSettings.GetRepoPath(),
			b.path,
			b.gitSettings.GetStartTime(),
			b.gitSettings.GetEndTime(),
			useFollow,
		)
		if err == nil {
			// Parse the output to populate metrics
			lines := strings.Split(string(out), "\n")
			var firstCommit time.Time
			totalAdd := schema.Metric(0)
			totalDel := schema.Metric(0)
			authorCommits := make(map[string]schema.Metric)

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
							totalAdd += schema.Metric(add)
							totalDel += schema.Metric(del)
						}
					}
				}
			}

			b.contribCount = authorCommits
			b.result.UniqueContributors = schema.Metric(len(b.contribCount))
			b.result.Commits = schema.Metric(b.totalCommits)
			b.result.LinesAdded = schema.Metric(totalAdd)
			b.result.LinesDeleted = schema.Metric(totalDel)
			b.result.Churn = schema.Metric(totalAdd + totalDel)
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
	fullPath := filepath.Join(b.gitSettings.GetRepoPath(), b.path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return b
	}

	size := int64(len(content))
	lines := bytes.Count(content, []byte("\n"))

	// If the file is not empty and doesn't end with a newline, it has
	// an "orphaned" last line that bytes.Count missed.
	if size > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		lines++
	}

	b.result.SizeBytes = size
	b.result.LinesOfCode = schema.Metric(lines)

	return b
}

// CalculateDerivedMetrics computes metrics that depend on previously collected data.
func (b *FileResultBuilder) CalculateDerivedMetrics() *FileResultBuilder {
	// AgeDays
	if b.result.FirstCommit.IsZero() {
		b.result.AgeDays = 0
	} else {
		// Calculate age relative to the analysis EndTime if available, else time.Now()
		refTime := time.Now()
		if b.output != nil && !b.output.EndTime.IsZero() {
			refTime = b.output.EndTime
		}
		b.result.AgeDays = schema.Metric(schema.CalculateDaysBetween(b.result.FirstCommit, refTime))
	}

	// Gini coefficient for author diversity
	values := make([]float64, 0, len(b.contribCount))
	for _, c := range b.contribCount {
		values = append(values, float64(c))
	}
	b.result.Gini = algo.Gini(values) // Assuming gini() is a helper function

	return b
}

// FetchRecentInfo populates recent metrics from recent info if available.
func (b *FileResultBuilder) FetchRecentInfo() *FileResultBuilder {
	b.result.RecentWindowDays = 30 // Fixed default window

	if b.output == nil {
		return b
	}

	if stat, ok := b.output.FileStats[b.path]; ok {
		b.result.RecentCommits = stat.RecentCommits
		b.result.RecentChurn = stat.RecentChurn
		b.result.RecentLinesAdded = stat.RecentLinesAdded
		b.result.RecentLinesDeleted = stat.RecentLinesDeleted
		b.result.RecentContributors = schema.Metric(len(stat.RecentContributors))
	}
	return b
}

// CalculateOwner identifies the owner based on commit volume.
func (b *FileResultBuilder) CalculateOwner() *FileResultBuilder {
	if b.output == nil {
		return b
	}

	stat, ok := b.output.FileStats[b.path]
	if !ok || len(stat.Contributors) == 0 {
		return b
	}

	authorMap := stat.Contributors

	// Sort authors by commit count descending, then by author name ascending for stable ordering
	type authorCommits struct {
		author  string
		commits schema.Metric
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
	mode := b.scoringSettings.GetMode()
	weights := b.scoringSettings.GetComputedWeights()[mode]
	thresholdLow := b.scoringSettings.GetRecencyThresholdLow()
	thresholdHigh := b.scoringSettings.GetRecencyThresholdHigh()
	b.result.ModeScore = algo.ComputeScore(b.result, mode, weights, thresholdLow, thresholdHigh)

	// Compute scores and breakdowns for all modes
	b.result.AllScores = make(map[schema.ScoringMode]float64)
	b.result.AllBreakdowns = make(map[schema.ScoringMode]map[schema.BreakdownKey]float64)

	computedWeights := b.scoringSettings.GetComputedWeights()
	for _, m := range []schema.ScoringMode{schema.HotMode, schema.RiskMode, schema.ComplexityMode, schema.ROIMode} {
		if m == mode {
			// Already computed
			b.result.AllScores[m] = b.result.ModeScore
			b.result.AllBreakdowns[m] = make(map[schema.BreakdownKey]float64, len(b.result.ModeBreakdown))
			maps.Copy(b.result.AllBreakdowns[m], b.result.ModeBreakdown)
		} else {
			mCopy := *b.result // Shallow copy of top-level fields
			// Crucially re-initialize the breakdown map to avoid stomping on the original
			mCopy.ModeBreakdown = make(map[schema.BreakdownKey]float64, 8)
			mCopy.Mode = m
			score := algo.ComputeScore(&mCopy, m, computedWeights[m], thresholdLow, thresholdHigh)
			b.result.AllScores[m] = score
			b.result.AllBreakdowns[m] = mCopy.ModeBreakdown
		}
	}

	return b
}

// Build finalizes the construction and returns the completed metrics object.
func (b *FileResultBuilder) Build() schema.FileResult {
	return *b.result
}
