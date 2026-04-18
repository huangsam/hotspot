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

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/schema"
)

// aggregateActivity performs a single repository-wide git log and aggregates per-file
// commits, churn and contributors. It runs over the entire history if
// gitSettings.GetStartTime() is zero, or runs since gitSettings.GetStartTime() otherwise.
// It filters out files that no longer exist in a single pass.
func aggregateActivity(ctx context.Context, gitSettings config.GitSettings, client git.Client) (*schema.AggregateOutput, error) {
	// 1. Get the list of currently existing files
	currentFiles, err := client.ListFilesAtRef(ctx, gitSettings.GetRepoPath(), "HEAD")
	if err != nil {
		return nil, err
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
	out, err := client.GetActivityLog(ctx, gitSettings.GetRepoPath(), gitSettings.GetStartTime(), gitSettings.GetEndTime())
	if err != nil {
		return nil, err
	}

	// 5. Parse and aggregate the git log output
	parseAndAggregateGitLog(out, fileExists, output, recentThreshold)

	return output, nil
}

// buildFileExistenceMap creates a lookup map for O(1) file existence checks.
func buildFileExistenceMap(currentFiles []string) map[string]bool {
	fileExists := make(map[string]bool, len(currentFiles))
	for _, file := range currentFiles {
		fileExists[file] = true
	}
	return fileExists
}

// initializeAggregateOutput creates the AggregateOutput and its internal maps.
func initializeAggregateOutput(endTime time.Time) *schema.AggregateOutput {
	return &schema.AggregateOutput{
		EndTime:               endTime,
		CommitMap:             make(map[string]schema.Metric),
		ChurnMap:              make(map[string]schema.Metric),
		ContribMap:            make(map[string]map[string]schema.Metric),
		FirstCommitMap:        make(map[string]time.Time),
		LinesAddedMap:         make(map[string]schema.Metric),
		LinesDeletedMap:       make(map[string]schema.Metric),
		RecentCommitMap:       make(map[string]schema.Metric),
		RecentChurnMap:        make(map[string]schema.Metric),
		RecentLinesAddedMap:   make(map[string]schema.Metric),
		RecentLinesDeletedMap: make(map[string]schema.Metric),
		RecentContribMap:      make(map[string]map[string]schema.Metric),
		DecayedCommitMap:      make(map[string]schema.Metric),
		DecayedChurnMap:       make(map[string]schema.Metric),
	}
}

// parseAndAggregateGitLog processes the git log output and aggregates data into the output maps.
func parseAndAggregateGitLog(out []byte, fileExists map[string]bool, output *schema.AggregateOutput, recentThreshold time.Time) {
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
		pathsToAggregate, add, del := parseFileStatsLine(l, fileExists)
		if len(pathsToAggregate) == 0 {
			continue
		}

		// Aggregate for each relevant path
		for _, p := range pathsToAggregate {
			aggregateForPath(p, add, del, currentAuthor, currentDate, output, recentThreshold)
		}
	}
}

// parseCommitHeader extracts author and date from a commit header line.
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

	// Intern the author name
	author := string(authorBytes)
	if cached, ok := authorCache[author]; ok {
		author = cached
	} else {
		authorCache[author] = author
	}

	dateStr := string(dateBytes)
	if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return author, date
	}

	return "", time.Time{}
}

// parseFileStatsLine parses a file stats line and returns paths to aggregate and churn values.
func parseFileStatsLine(line []byte, fileExists map[string]bool) ([]string, schema.Metric, schema.Metric) {
	firstTab := bytes.IndexByte(line, '\t')
	if firstTab == -1 {
		return nil, 0, 0
	}
	secondTab := bytes.IndexByte(line[firstTab+1:], '\t')
	if secondTab == -1 {
		return nil, 0, 0
	}
	secondTab += firstTab + 1

	add := parseChurnValue(line[:firstTab])
	del := parseChurnValue(line[firstTab+1 : secondTab])
	path := string(line[secondTab+1:])

	// Optimization: check if it's a simple path first to avoid slice allocation in determinePathsToAggregate
	if !strings.Contains(path, " => ") {
		if fileExists[path] {
			return []string{path}, add, del
		}
		return nil, add, del
	}

	pathsToAggregate := determinePathsToAggregate(path, fileExists)
	return pathsToAggregate, add, del
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
func determinePathsToAggregate(path string, fileExists map[string]bool) []string {
	if !strings.Contains(path, " => ") {
		if fileExists[path] {
			return []string{path}
		}
		return nil
	}

	// Handle renames
	oldPath, newPath := parseRenamePath(path)
	var paths []string
	if fileExists[oldPath] {
		paths = append(paths, oldPath)
	}
	if fileExists[newPath] {
		paths = append(paths, newPath)
	}
	return paths
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

	// Base metrics
	output.ChurnMap[path] += churn
	output.CommitMap[path]++
	output.DecayedCommitMap[path] += schema.Metric(decayFactor)
	output.DecayedChurnMap[path] += churn * schema.Metric(decayFactor)
	output.LinesAddedMap[path] += add
	output.LinesDeletedMap[path] += del

	if author != "" {
		if output.ContribMap[path] == nil {
			output.ContribMap[path] = make(map[string]schema.Metric)
		}
		output.ContribMap[path][author]++
	}

	if !date.IsZero() {
		if existing, ok := output.FirstCommitMap[path]; !ok || date.Before(existing) {
			output.FirstCommitMap[path] = date
		}
	}

	// Recent metrics filtering
	if !date.IsZero() && !date.Before(recentThreshold) {
		output.RecentChurnMap[path] += churn
		output.RecentCommitMap[path]++
		output.RecentLinesAddedMap[path] += add
		output.RecentLinesDeletedMap[path] += del
		if author != "" {
			if output.RecentContribMap[path] == nil {
				output.RecentContribMap[path] = make(map[string]schema.Metric)
			}
			output.RecentContribMap[path][author]++
		}
	}
}

// BuildFilteredFileList creates a unified list of files from activity maps
// and filters them based on the configuration.
func BuildFilteredFileList(gitSettings config.GitSettings, output *schema.AggregateOutput) []string {
	// 1. Estimate capacity for 'seen'. Use a good guess based on the largest map.
	capacity := max(
		len(output.ContribMap), max(
			len(output.ChurnMap), len(output.CommitMap)))

	// Use struct{} for zero-memory value.
	seen := make(map[string]struct{}, capacity)

	// 2. Populate 'seen' map from all three sources.

	// a) CommitMap and ChurnMap (Simple Iteration)
	// We combine this by iterating over the simple maps first.
	for k := range output.CommitMap {
		seen[k] = struct{}{}
	}
	for k := range output.ChurnMap {
		seen[k] = struct{}{}
	}

	// b) ContribMap (Nested Iteration)
	// We only care about the file path, which is the outer key (k).
	for k := range output.ContribMap {
		seen[k] = struct{}{}
	}

	// 3. Apply filters and build final list
	files := make([]string, 0, len(seen))

	// Pre-calculate filter state (only if PathFilter is set)
	pathFilter := gitSettings.GetPathFilter()
	pathFilterSet := pathFilter != ""

	for f := range seen {
		// Apply path filter check only if the filter is set
		if pathFilterSet && !strings.HasPrefix(f, pathFilter) {
			continue
		}

		// Apply excludes filter
		if schema.ShouldIgnore(f, gitSettings.GetExcludes()) {
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
