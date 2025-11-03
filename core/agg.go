package core

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// aggregateActivity performs a single repository-wide git log and aggregates per-file
// commits, churn and contributors. It runs over the entire history if
// cfg.StartTime is zero, or runs since cfg.StartTime otherwise.
// It filters out files that no longer exist in a single pass.
func aggregateActivity(cfg *internal.Config) (*schema.AggregateOutput, error) {
	// 1. Get the list of currently existing files FIRST.
	// This git call is very fast.
	currentFiles, err := listRepoFiles(cfg.RepoPath)
	if err != nil {
		return nil, err
	}
	// Build a lookup map for O(1) existence checks.
	fileExists := make(map[string]bool)
	for _, file := range currentFiles {
		fileExists[file] = true
	}

	// 2. Build the git command and select the correct maps
	args := []string{"log", "--numstat", "--pretty=format:'--%H|%an'"}

	commitsMap := make(map[string]int)
	churnMap := make(map[string]int)
	contribMap := make(map[string]map[string]int)

	if !cfg.StartTime.IsZero() {
		// If we get to the place where we need global maps for "all" time, then
		// we should refactor this function to return an "immutable" output
		// instead of relying on globals
		since := cfg.StartTime.Format(internal.DateTimeFormat)
		args = append(args, "--since="+since)
	}

	// 3. Run the expensive git log command ONCE
	out, err := internal.RunGitCommand(cfg.RepoPath, args...)
	if err != nil {
		return nil, err
	}

	// 4. Perform aggregation AND filtering in a single pass
	lines := strings.Split(string(out), "\n")
	var currentAuthor string
	for _, l := range lines {
		// Strip the surrounding single quotes, whitespace, and carriage returns
		l = strings.Trim(l, " \t\r\n'")

		if strings.HasPrefix(l, "--") {
			// Commit header line
			parts := strings.SplitN(l[2:], "|", 2) // Slice off the leading "--"
			if len(parts) == 2 {
				currentAuthor = parts[1]
			} else {
				currentAuthor = "" // Should not happen with this format
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

		if !fileExists[path] {
			continue // Skip this file; it no longer exists.
		}

		add := 0
		del := 0
		if addStr != "-" {
			add, _ = strconv.Atoi(addStr)
		}
		if delStr != "-" {
			del, _ = strconv.Atoi(delStr)
		}

		// Aggregate using the dynamically selected maps
		churnMap[path] += add + del
		commitsMap[path]++
		if currentAuthor != "" {
			if contribMap[path] == nil {
				contribMap[path] = make(map[string]int)
			}
			contribMap[path][currentAuthor]++
		}
	}

	output := &schema.AggregateOutput{
		ChurnMap:   churnMap,
		CommitMap:  commitsMap,
		ContribMap: contribMap,
	}

	// 5. No filtering loops are needed. The maps are already clean.
	return output, nil
}

// listRepoFiles returns a list of all tracked files in the Git repository.
func listRepoFiles(repoPath string) ([]string, error) {
	out, err := internal.RunGitCommand(repoPath, "ls-files")
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

// aggregateAndScoreFolders correctly aggregates file results into folders.
func aggregateAndScoreFolders(cfg *internal.Config, fileMetrics []schema.FileMetrics) []schema.FolderResults {
	folderResults := make(map[string]*schema.FolderResults)

	// Map to track the aggregate commit count per author per folder:
	// folderPath -> authorName -> totalCommitsByAuthorInFolder
	folderAuthorContributions := make(map[string]map[string]int)

	for _, fm := range fileMetrics {
		// 1. Determine the folder path
		folderPath := filepath.Dir(fm.Path)
		if cfg.PathFilter == "" && folderPath == "." {
			continue // Skip the root if not filtered
		}

		if _, ok := folderResults[folderPath]; !ok {
			folderResults[folderPath] = &schema.FolderResults{
				Path: folderPath,
			}
		}

		// 2. Aggregate simple metrics and score components
		folderResults[folderPath].Commits += fm.Commits
		folderResults[folderPath].Churn += fm.Churn
		folderResults[folderPath].TotalLOC += fm.LinesOfCode
		folderResults[folderPath].WeightedScoreSum += fm.Score * float64(fm.LinesOfCode)

		// 3. Aggregate author contributions for owner calculation
		if fm.Owner != "" {
			if folderAuthorContributions[folderPath] == nil {
				folderAuthorContributions[folderPath] = make(map[string]int)
			}

			// Use the file's total commits as the weight for its primary author's contribution
			// to the folder. This finds the author who has done the most work (measured by commits)
			// across all files in the folder.
			folderAuthorContributions[folderPath][fm.Owner] += fm.Commits
		}
	}

	// Finalize: Calculate unique contributor count and the final score
	results := make([]schema.FolderResults, 0, len(folderResults))
	for _, res := range folderResults {
		// Calculate the score (Average File Score, weighted by LOC)
		res.Score = calculateFolderScore(res)

		// Determine the Most Frequent Author (Owner)
		if authorMap := folderAuthorContributions[res.Path]; len(authorMap) > 0 {
			var owner string
			var maxCommits int
			for author, commits := range authorMap {
				if maxCommits < commits {
					maxCommits = commits
					owner = author
				}
			}
			res.Owner = owner
		}

		results = append(results, *res)
	}

	return results
}
