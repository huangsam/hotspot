package core

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// aggregateActivity performs a single repository-wide git log and aggregates per-file
// commits, churn and contributors. It runs over the entire history if
// cfg.StartTime is zero, or runs since cfg.StartTime otherwise.
// It filters out files that no longer exist in a single pass.
func aggregateActivity(ctx context.Context, cfg *internal.Config, client internal.GitClient) (*schema.AggregateOutput, error) {
	// 1. Get the list of currently existing files FIRST using the new explicit method.
	// This git call is very fast and uses the abstract client method.
	currentFiles, err := client.ListFilesAtRef(ctx, cfg.RepoPath, "HEAD")
	if err != nil {
		return nil, err
	}
	// Build a lookup map for O(1) existence checks.
	fileExists := make(map[string]bool)
	for _, file := range currentFiles {
		fileExists[file] = true
	}

	// 2. Initialize aggregation maps. (No change here)
	commitsMap := make(map[string]int)
	churnMap := make(map[string]int)
	contribMap := make(map[string]map[string]int)
	firstCommitMap := make(map[string]time.Time)

	// 3. Run the expensive git log command ONCE using the new explicit method.
	// The client now handles argument construction for zero-valued times.
	out, err := client.GetActivityLog(ctx, cfg.RepoPath, cfg.StartTime, cfg.EndTime)
	if err != nil {
		return nil, err
	}

	// 4. Perform aggregation AND filtering in a single pass (Logic unchanged)
	lines := strings.Split(string(out), "\n")
	var currentAuthor string
	var currentDate time.Time
	for _, l := range lines {
		// Strip the surrounding single quotes, whitespace, and carriage returns
		l = strings.Trim(l, " \t\r\n'")

		if strings.HasPrefix(l, "--") {
			// Commit header line
			parts := strings.SplitN(l[2:], "|", 3) // commit|author|date
			if len(parts) == 3 {
				currentAuthor = parts[1]
				dateStr := parts[2]
				if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
					currentDate = date
				} else {
					currentDate = time.Time{}
				}
			} else {
				currentAuthor = ""
				currentDate = time.Time{}
			}
			continue
		}
		if l == "" {
			continue // Skip blank lines after trimming
		}

		// This is a file stats line
		parts := strings.SplitN(l, "\t", 3)
		if len(parts) < 3 {
			continue // Skip unexpected lines (like merge info without stats)
		}
		addStr := parts[0]
		delStr := parts[1]
		path := parts[2]

		add := 0
		del := 0
		if addStr != "-" {
			add, _ = strconv.Atoi(addStr)
		}
		if delStr != "-" {
			del, _ = strconv.Atoi(delStr)
		}

		// Handle renames: git --numstat can output renames in various formats.
		// Examples:
		// - Simple: "old/path => new/path"
		// - With braces and suffix: "{old => new}/file" (e.g., "{schema => internal}/configs.go")
		// - With braces and prefix: "dir/{old => new}" (e.g., "internal/{output.go => output_files.go}")
		// We parse these to extract oldPath and newPath, then aggregate for existing files.
		var pathsToAggregate []string
		if strings.Contains(path, " => ") {
			var oldPath, newPath string
			if strings.Contains(path, "{") && strings.Contains(path, "}") {
				// Handle braced formats: split into prefix{old => new}suffix
				braceStart := strings.Index(path, "{")
				braceEnd := strings.Index(path, "}")
				if braceStart != -1 && braceEnd != -1 && braceStart < braceEnd {
					prefix := path[:braceStart]
					renamePart := path[braceStart+1 : braceEnd] // "old => new"
					suffix := path[braceEnd+1:]
					if strings.Contains(renamePart, " => ") {
						renameParts := strings.SplitN(renamePart, " => ", 2)
						oldPath = prefix + renameParts[0] + suffix
						newPath = prefix + renameParts[1] + suffix
					}
				}
			} else {
				// Handle simple format: "old => new"
				renameParts := strings.SplitN(path, " => ", 2)
				oldPath = renameParts[0]
				newPath = renameParts[1]
			}
			if fileExists[oldPath] {
				pathsToAggregate = append(pathsToAggregate, oldPath)
			}
			if fileExists[newPath] {
				pathsToAggregate = append(pathsToAggregate, newPath)
			}
		} else if fileExists[path] {
			pathsToAggregate = []string{path}
		}

		// Aggregate for each relevant path
		for _, p := range pathsToAggregate {
			// Aggregate using the dynamically selected maps
			churnMap[p] += add + del
			commitsMap[p]++
			if currentAuthor != "" {
				if contribMap[p] == nil {
					contribMap[p] = make(map[string]int)
				}
				contribMap[p][currentAuthor]++
			}

			// Track first commit time
			if !currentDate.IsZero() {
				if existing, ok := firstCommitMap[p]; !ok || currentDate.Before(existing) {
					firstCommitMap[p] = currentDate
				}
			}
		}
	}

	output := &schema.AggregateOutput{
		ChurnMap:       churnMap,
		CommitMap:      commitsMap,
		ContribMap:     contribMap,
		FirstCommitMap: firstCommitMap,
	}

	// 5. No filtering loops are needed. The maps are already clean.
	return output, nil
}

// buildFilteredFileList creates a unified list of files from activity maps
// and filters them based on the configuration.
func buildFilteredFileList(cfg *internal.Config, output *schema.AggregateOutput) []string {
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
		if internal.ShouldIgnore(f, cfg.Excludes) {
			continue
		}

		files = append(files, f)
	}

	return files
}

// aggregateAndScoreFolders correctly aggregates file results into folders.
func aggregateAndScoreFolders(cfg *internal.Config, fileResults []schema.FileResult) []schema.FolderResult {
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
			}
		}

		// 2. Aggregate simple metrics and score components
		folderResults[folderPath].Commits += fr.Commits
		folderResults[folderPath].Churn += fr.Churn
		folderResults[folderPath].TotalLOC += fr.LinesOfCode
		folderResults[folderPath].WeightedScoreSum += fr.Score * float64(fr.LinesOfCode)

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
