// Package agg has aggregation logic for Git activity data.
package agg

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal/contract"
	"github.com/huangsam/hotspot/schema"
)

// aggregateActivity performs a single repository-wide git log and aggregates per-file
// commits, churn and contributors. It runs over the entire history if
// cfg.StartTime is zero, or runs since cfg.StartTime otherwise.
// It filters out files that no longer exist in a single pass.
func aggregateActivity(ctx context.Context, cfg *contract.Config, client contract.GitClient) (*schema.AggregateOutput, error) {
	// 1. Get the list of currently existing files
	currentFiles, err := client.ListFilesAtRef(ctx, cfg.RepoPath, "HEAD")
	if err != nil {
		return nil, err
	}
	fileExists := buildFileExistenceMap(currentFiles)

	// 2. Initialize aggregation maps
	commitsMap, churnMap, contribMap, firstCommitMap := initializeAggregationMaps()

	// 3. Run the git log command
	out, err := client.GetActivityLog(ctx, cfg.RepoPath, cfg.StartTime, cfg.EndTime)
	if err != nil {
		return nil, err
	}

	// 4. Parse and aggregate the git log output
	parseAndAggregateGitLog(out, fileExists, commitsMap, churnMap, contribMap, firstCommitMap)

	output := &schema.AggregateOutput{
		ChurnMap:       churnMap,
		CommitMap:      commitsMap,
		ContribMap:     contribMap,
		FirstCommitMap: firstCommitMap,
	}

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

// initializeAggregationMaps creates the maps used for aggregating git data.
func initializeAggregationMaps() (map[string]int, map[string]int, map[string]map[string]int, map[string]time.Time) {
	commitsMap := make(map[string]int)
	churnMap := make(map[string]int)
	contribMap := make(map[string]map[string]int)
	firstCommitMap := make(map[string]time.Time)
	return commitsMap, churnMap, contribMap, firstCommitMap
}

// parseAndAggregateGitLog processes the git log output and aggregates data into the maps.
func parseAndAggregateGitLog(out []byte, fileExists map[string]bool, commitsMap, churnMap map[string]int, contribMap map[string]map[string]int, firstCommitMap map[string]time.Time) {
	lines := strings.Split(string(out), "\n")
	var currentAuthor string
	var currentDate time.Time

	for _, l := range lines {
		l = strings.Trim(l, " \t\r\n'")

		if strings.HasPrefix(l, "--") {
			// Commit header line
			currentAuthor, currentDate = parseCommitHeader(l)
			continue
		}
		if l == "" {
			continue // Skip blank lines
		}

		// File stats line
		pathsToAggregate, add, del := parseFileStatsLine(l, fileExists)
		if len(pathsToAggregate) == 0 {
			continue
		}

		// Aggregate for each relevant path
		for _, p := range pathsToAggregate {
			aggregateForPath(p, add+del, currentAuthor, currentDate, commitsMap, churnMap, contribMap, firstCommitMap)
		}
	}
}

// parseCommitHeader extracts author and date from a commit header line.
func parseCommitHeader(line string) (string, time.Time) {
	if !strings.HasPrefix(line, "--") || len(line) < 5 { // --x|y|z minimum
		return "", time.Time{}
	}
	parts := strings.SplitN(line[2:], "|", 3) // commit|author|date
	if len(parts) == 3 {
		author := parts[1]
		dateStr := parts[2]
		if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
			return author, date
		}
	}
	return "", time.Time{}
}

// parseFileStatsLine parses a file stats line and returns paths to aggregate and churn values.
func parseFileStatsLine(line string, fileExists map[string]bool) ([]string, int, int) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) < 3 {
		return nil, 0, 0
	}

	addStr, delStr, path := parts[0], parts[1], parts[2]

	add := parseChurnValue(addStr)
	del := parseChurnValue(delStr)

	pathsToAggregate := determinePathsToAggregate(path, fileExists)
	return pathsToAggregate, add, del
}

// parseChurnValue converts a churn string to int, handling "-" as 0.
func parseChurnValue(s string) int {
	if s == "-" {
		return 0
	}
	if val, err := strconv.Atoi(s); err == nil && val >= 0 {
		return val
	}
	return 0
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
func aggregateForPath(path string, churn int, author string, date time.Time, commitsMap, churnMap map[string]int, contribMap map[string]map[string]int, firstCommitMap map[string]time.Time) {
	churnMap[path] += churn
	commitsMap[path]++

	if author != "" {
		if contribMap[path] == nil {
			contribMap[path] = make(map[string]int)
		}
		contribMap[path][author]++
	}

	if !date.IsZero() {
		if existing, ok := firstCommitMap[path]; !ok || date.Before(existing) {
			firstCommitMap[path] = date
		}
	}
}

// BuildFilteredFileList creates a unified list of files from activity maps
// and filters them based on the configuration.
func BuildFilteredFileList(cfg *contract.Config, output *schema.AggregateOutput) []string {
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
	pathFilterSet := cfg.PathFilter != ""

	for f := range seen {
		// Apply path filter check only if the filter is set
		if pathFilterSet && !strings.HasPrefix(f, cfg.PathFilter) {
			continue
		}

		// Apply excludes filter
		if contract.ShouldIgnore(f, cfg.Excludes) {
			continue
		}

		files = append(files, f)
	}

	return files
}

// AggregateAndScoreFolders correctly aggregates file results into folders.
func AggregateAndScoreFolders(cfg *contract.Config, fileResults []schema.FileResult) []schema.FolderResult {
	folderResults := make(map[string]*schema.FolderResult)

	// Map to track the aggregate commit count per author per folder:
	// folderPath -> authorName -> totalCommitsByAuthorInFolder
	folderAuthorContributions := make(map[string]map[string]int)

	for _, fr := range fileResults {
		// 1. Determine the folder path
		folderPath := filepath.Dir(fr.Path)
		if cfg.PathFilter == "" && folderPath == "." {
			continue // Skip the root if not filtered
		}

		if _, ok := folderResults[folderPath]; !ok {
			folderResults[folderPath] = &schema.FolderResult{
				Path: folderPath,
				Mode: cfg.Mode,
			}
		}

		// 2. Aggregate simple metrics and score components
		folderResults[folderPath].Commits += fr.Commits
		folderResults[folderPath].Churn += fr.Churn
		folderResults[folderPath].TotalLOC += fr.LinesOfCode
		folderResults[folderPath].WeightedScoreSum += fr.ModeScore * float64(fr.LinesOfCode)

		// 3. Aggregate author contributions for owner calculation
		if len(fr.Owners) > 0 {
			if folderAuthorContributions[folderPath] == nil {
				folderAuthorContributions[folderPath] = make(map[string]int)
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
	score := folderResult.WeightedScoreSum / float64(folderResult.TotalLOC)

	// Apply optional debuffs if needed, similar to CalculateFileScore
	// For simplicity, we just return the raw weighted average here.
	return score
}
