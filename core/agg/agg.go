// Package agg has aggregation logic for Git activity data.
package agg

import (
	"bufio"
	"bytes"
	"context"
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

// aggregateActivity performs a single repository-wide git log and aggregates per-file
// commits, churn and contributors. It runs over the entire history if
// gitSettings.GetStartTime() is zero, or runs since gitSettings.GetStartTime() otherwise.
// It filters out files that no longer exist in a single pass.
func aggregateActivity(ctx context.Context, gitSettings config.GitSettings, client git.Client, currentFiles []string) (*schema.AggregateOutput, error) {
	// 1. Get the list of currently existing files if not provided
	if currentFiles == nil {
		var err error
		currentFiles, err = client.ListFilesAtRef(ctx, gitSettings.GetRepoPath(), "HEAD")
		if err != nil {
			return nil, err
		}
	}
	fileExists := buildFileExistenceMap(currentFiles)

	// 2. Initialize aggregation maps
	endTime := gitSettings.GetEndTime()
	if endTime.IsZero() {
		endTime = time.Now()
	}
	output := initializeAggregateOutput(endTime)

	// 3. Determine recent window threshold (e.g., 30 days before EndTime or Now)
	recentThreshold := endTime.AddDate(0, 0, -30) // Fixed 30-day window for now

	// 4. Run the git log command
	out, err := client.GetActivityLog(ctx, gitSettings.GetRepoPath(), gitSettings.GetPathFilter(), gitSettings.GetStartTime(), gitSettings.GetEndTime())
	if err != nil {
		return nil, err
	}

	// 5. Parse and aggregate the git log output
	parseAndAggregateGitLog(out, fileExists, output, recentThreshold)

	return output, nil
}

// buildFileExistenceMap creates a lookup map for O(1) file existence checks.
// It returns a map where key and value are the same string object, allowing
// for zero-allocation string interning of paths during parsing.
func buildFileExistenceMap(currentFiles []string) map[string]string {
	fileExists := make(map[string]string, len(currentFiles))
	for _, file := range currentFiles {
		fileExists[file] = file
	}
	return fileExists
}

// initializeAggregateOutput creates the AggregateOutput and its internal maps.
func initializeAggregateOutput(endTime time.Time) *schema.AggregateOutput {
	return &schema.AggregateOutput{
		EndTime:   endTime,
		FileStats: make(map[string]*schema.FileAggregation),
	}
}

// parseAndAggregateGitLog processes the git log output and aggregates data into the output maps.
func parseAndAggregateGitLog(out []byte, fileExists map[string]string, output *schema.AggregateOutput, recentThreshold time.Time) {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var currentAuthor string
	var currentDate time.Time

	// authorCache interns strings to reuse author names across many commits
	authorCache := make(map[string]string)

	for scanner.Scan() {
		// We must trim whitespace and single quotes because git log format
		// --pretty=format:'--%H|%an|%ad' wraps the header in quotes.
		l := bytes.Trim(scanner.Bytes(), " \t\r\n'")
		if len(l) == 0 {
			continue
		}

		if bytes.HasPrefix(l, []byte("--")) {
			// Commit header line
			currentAuthor, currentDate = parseCommitHeader(l, authorCache)
			continue
		}

		// File stats line
		p1, p2, add, del := parseFileStatsLine(l, fileExists)
		if p1 != "" {
			aggregateForPath(p1, add, del, currentAuthor, currentDate, output, recentThreshold)
		}
		if p2 != "" {
			aggregateForPath(p2, add, del, currentAuthor, currentDate, output, recentThreshold)
		}
	}
}

// parseCommitHeader extracts author and date from a commit header line.
// It uses authorCache for string interning to avoid redundant allocations.
func parseCommitHeader(line []byte, authorCache map[string]string) (string, time.Time) {
	if !bytes.HasPrefix(line, []byte("--")) || len(line) < 5 { // --x|y|z minimum
		return "", time.Time{}
	}

	// Skip the leading "--"
	line = line[2:]

	// Manually find components to avoid SplitN allocations
	firstSep := bytes.IndexByte(line, '|')
	if firstSep == -1 {
		return "", time.Time{}
	}
	secondSep := bytes.IndexByte(line[firstSep+1:], '|')
	if secondSep == -1 {
		return "", time.Time{}
	}
	secondSep += firstSep + 1

	authorBytes := line[firstSep+1 : secondSep]
	dateBytes := line[secondSep+1:]

	// Optimization: compiler avoids allocation for string(authorBytes) when used as map key
	author, ok := authorCache[string(authorBytes)]
	if !ok {
		// Only allocate if not already in cache
		author = string(authorBytes)
		authorCache[author] = author
	}

	// We still allocate for the date string since time.Parse needs it,
	// but this is only once per commit header.
	if date, err := time.Parse(time.RFC3339, string(dateBytes)); err == nil {
		return author, date
	}

	return "", time.Time{}
}

// parseFileStatsLine parses a file stats line and returns paths to aggregate and churn values.
// It returns up to two paths (for renames) to avoid slice allocations.
func parseFileStatsLine(line []byte, fileExists map[string]string) (string, string, schema.Metric, schema.Metric) {
	firstTab := bytes.IndexByte(line, '\t')
	if firstTab == -1 {
		return "", "", 0, 0
	}
	secondTab := bytes.IndexByte(line[firstTab+1:], '\t')
	if secondTab == -1 {
		return "", "", 0, 0
	}
	secondTab += firstTab + 1

	add := parseChurnValue(line[:firstTab])
	del := parseChurnValue(line[firstTab+1 : secondTab])
	pathBytes := line[secondTab+1:]

	// Optimization: check if it's a simple path first to avoid allocations.
	if !bytes.Contains(pathBytes, []byte(" => ")) {
		if canonical, ok := fileExists[string(pathBytes)]; ok {
			return canonical, "", add, del
		}
		return "", "", add, del
	}

	// Renames still require string conversion for complex parsing
	p1, p2 := determinePathsToAggregate(string(pathBytes), fileExists)
	return p1, p2, add, del
}

// parseChurnValue converts a churn byte slice to Metric, handling "-" as 0.
// It uses a zero-allocation fast path for positive integers.
func parseChurnValue(b []byte) schema.Metric {
	if len(b) == 0 || (len(b) == 1 && b[0] == '-') {
		return 0
	}

	// Fast path for positive integers (common case in git log)
	var val float64
	for _, x := range b {
		if x < '0' || x > '9' {
			// Fallback for floats or invalid input
			v, err := strconv.ParseFloat(string(b), 64)
			if err != nil || v < 0 {
				return 0
			}
			return schema.Metric(v)
		}
		val = val*10 + float64(x-'0')
	}
	return schema.Metric(val)
}

// determinePathsToAggregate handles renames and determines which paths should be aggregated.
func determinePathsToAggregate(path string, fileExists map[string]string) (string, string) {
	if !strings.Contains(path, " => ") {
		if canonical, ok := fileExists[path]; ok {
			return canonical, ""
		}
		return "", ""
	}

	// Handle renames
	oldPath, newPath := parseRenamePath(path)
	var p1, p2 string
	if canonical, ok := fileExists[oldPath]; ok {
		p1 = canonical
	}
	if canonical, ok := fileExists[newPath]; ok {
		p2 = canonical
	}
	return p1, p2
}

// parseRenamePath extracts old and new paths from a rename string.
func parseRenamePath(path string) (string, string) {
	if !strings.Contains(path, "{") {
		// Simple format: "old => new"
		parts := strings.SplitN(path, " => ", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return "", ""
	}

	if !strings.Contains(path, "}") {
		// Malformed: has { but no }
		return "", ""
	}

	// Braced format: prefix{old => new}suffix
	braceStart := strings.Index(path, "{")
	braceEnd := strings.Index(path, "}")
	if braceStart == -1 || braceEnd == -1 || braceStart >= braceEnd {
		return "", ""
	}

	prefix := path[:braceStart]
	renamePart := path[braceStart+1 : braceEnd]
	suffix := path[braceEnd+1:]

	if !strings.Contains(renamePart, " => ") {
		return "", ""
	}

	renameParts := strings.SplitN(renamePart, " => ", 2)
	oldPath := prefix + renameParts[0] + suffix
	newPath := prefix + renameParts[1] + suffix
	return oldPath, newPath
}

// aggregateForPath updates the aggregation maps for a single path.
func aggregateForPath(path string, add schema.Metric, del schema.Metric, author string, date time.Time, output *schema.AggregateOutput, recentThreshold time.Time) {
	churn := add + del

	// Calculate decay factor (Half-life: 180 days)
	decayFactor := 1.0
	if !date.IsZero() {
		// Calculate age relative to the analysis end time
		ageDays := output.EndTime.Sub(date).Hours() / 24.0
		decayFactor = schema.CalculateDecayFactor(ageDays, 180.0)
	}

	// Get or create the aggregation struct for this path
	stat, ok := output.FileStats[path]
	if !ok {
		stat = &schema.FileAggregation{
			Contributors:       make(map[string]schema.Metric),
			RecentContributors: make(map[string]schema.Metric),
		}
		output.FileStats[path] = stat
	}

	// Base metrics
	stat.Churn += churn
	stat.Commits++
	stat.DecayedCommits += schema.Metric(decayFactor)
	stat.DecayedChurn += churn * schema.Metric(decayFactor)
	stat.LinesAdded += add
	stat.LinesDeleted += del

	if author != "" {
		stat.Contributors[author]++
	}

	if !date.IsZero() {
		if stat.FirstCommit.IsZero() || date.Before(stat.FirstCommit) {
			stat.FirstCommit = date
		}
	}

	// Recent metrics filtering
	if !date.IsZero() && !date.Before(recentThreshold) {
		stat.RecentChurn += churn
		stat.RecentCommits++
		stat.RecentLinesAdded += add
		stat.RecentLinesDeleted += del
		if author != "" {
			stat.RecentContributors[author]++
		}
	}
}

// BuildFilteredFileList creates a unified list of files from activity maps
// and filters them based on the configuration.
func BuildFilteredFileList(gitSettings config.GitSettings, output *schema.AggregateOutput) []string {
	// 1. Initialize result slice with a good guess for capacity
	files := make([]string, 0, len(output.FileStats))

	// 2. Apply filters and build final list
	// Pre-calculate filter state (only if PathFilter is set)
	pathFilter := gitSettings.GetPathFilter()
	pathFilterSet := pathFilter != ""
	excludes := gitSettings.GetExcludes()
	matcher := schema.NewPathMatcher(excludes)

	for f := range output.FileStats {
		// Apply path filter check only if the filter is set
		if pathFilterSet && !schema.IsPathInFilter(f, pathFilter) {
			continue
		}

		// Apply excludes filter
		if matcher.Match(f) {
			continue
		}

		files = append(files, f)
	}

	return files
}

// AggregateAndScoreFolders correctly aggregates file results into folders.
func AggregateAndScoreFolders(gitSettings config.GitSettings, scoringSettings config.ScoringSettings, fileResults []schema.FileResult) []schema.FolderResult {
	folderResults := make(map[string]*schema.FolderResult)

	// Map to track the aggregate commit count per author per folder:
	// folderPath -> authorName -> totalCommitsByAuthorInFolder
	folderAuthorContributions := make(map[string]map[string]schema.Metric)

	pathFilter := gitSettings.GetPathFilter()
	scoringMode := scoringSettings.GetMode()

	for _, fr := range fileResults {
		// 1. Determine the folder path
		folderPath := filepath.Dir(fr.Path)
		if pathFilter == "" && folderPath == "." {
			continue // Skip the root if not filtered
		}

		if _, ok := folderResults[folderPath]; !ok {
			folderResults[folderPath] = &schema.FolderResult{
				Path: folderPath,
				Mode: scoringMode,
			}
		}

		// 2. Aggregate simple metrics and score components
		folderResults[folderPath].Commits += fr.Commits
		folderResults[folderPath].Churn += fr.Churn
		folderResults[folderPath].DecayedCommits += fr.DecayedCommits
		folderResults[folderPath].DecayedChurn += fr.DecayedChurn
		folderResults[folderPath].TotalLOC += fr.LinesOfCode
		folderResults[folderPath].WeightedScoreSum += fr.ModeScore * fr.LinesOfCode.Float64()

		// 3. Aggregate author contributions for owner calculation
		if len(fr.Owners) > 0 {
			if folderAuthorContributions[folderPath] == nil {
				folderAuthorContributions[folderPath] = make(map[string]schema.Metric)
			}

			// Use the file's primary owner's total commits as the weight for its author's contribution
			// to the folder. This finds the author who has done the most work (measured by commits)
			// across all files in the folder.
			folderAuthorContributions[folderPath][fr.Owners[0]] += fr.Commits
		}
	}

	// Finalize: Calculate unique contributor count and the final score
	finalResults := make([]schema.FolderResult, 0, len(folderResults))
	for _, res := range folderResults {
		// Calculate the score (Average File Score, weighted by LOC)
		res.Score = computeFolderScore(res)

		// Determine the Most Frequent Author (Owner)
		if authorMap := folderAuthorContributions[res.Path]; len(authorMap) > 0 {
			// Sort authors by commit count descending
			type authorCommits struct {
				author  string
				commits schema.Metric
			}
			var authors []authorCommits
			for author, commits := range authorMap {
				authors = append(authors, authorCommits{author: author, commits: commits})
			}
			sort.Slice(authors, func(i, j int) bool {
				return authors[i].commits > authors[j].commits
			})

			// NEW: Calculate UniqueContributors and Gini for the folder
			res.UniqueContributors = schema.Metric(len(authorMap))
			giniValues := make([]float64, 0, len(authorMap))
			for _, commits := range authorMap {
				giniValues = append(giniValues, commits.Float64())
			}
			res.Gini = algo.Gini(giniValues)

			// Set top owner and top 2 owners
			if len(authors) > 0 {
				res.Owners = make([]string, 0, 2)
				for i := 0; i < len(authors) && i < 2; i++ {
					res.Owners = append(res.Owners, authors[i].author)
				}
			}
		}

		finalResults = append(finalResults, *res)
	}

	return finalResults
}

// computeFolderScore computes the final score for a folder as a weighted average.
// The weight for the average is Lines of Code (LOC).
func computeFolderScore(folderResult *schema.FolderResult) float64 {
	// Calculate Weighted Average Score
	if folderResult.TotalLOC == 0 {
		return 0.0
	}
	// Weighted Average Score = SUM(FileScore * FileLOC) / SUM(FileLOC)
	score := folderResult.WeightedScoreSum / folderResult.TotalLOC.Float64()

	// Apply optional debuffs if needed, similar to CalculateFileScore
	// For simplicity, we just return the raw weighted average here.
	return score
}
