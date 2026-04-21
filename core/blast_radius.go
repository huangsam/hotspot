package core

import (
	"context"
	"sort"
	"strings"

	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/schema"
)

// GetHotspotBlastRadiusResults identifies files that historically change together.
// It uses Jaccard Index to measure coupling strength.
func GetHotspotBlastRadiusResults(ctx context.Context, cfg *config.Config, client git.Client, limit int, threshold float64) (schema.BlastRadiusResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if threshold <= 0 {
		threshold = 0.3
	}

	// 1. Get the list of currently existing files to filter out deleted ones
	currentFiles, err := client.ListFilesAtRef(ctx, cfg.Git.RepoPath, "HEAD")
	if err != nil {
		return schema.BlastRadiusResult{}, err
	}
	fileExists := make(map[string]bool, len(currentFiles))
	for _, f := range currentFiles {
		fileExists[f] = true
	}

	// 2. Fetch activity log
	out, err := client.GetActivityLog(ctx, cfg.Git.RepoPath, cfg.Git.PathFilter, cfg.Git.StartTime, cfg.Git.EndTime)
	if err != nil {
		return schema.BlastRadiusResult{}, err
	}

	// 3. Parse log into commit batches
	lines := strings.Split(string(out), "\n")
	commits := make(map[string][]string)
	var currentCommit string
	totalCommits := 0

	matcher := schema.NewPathMatcher(cfg.Git.Excludes)

	for _, l := range lines {
		l = strings.Trim(l, " \t\r\n'")
		if strings.HasPrefix(l, "--") {
			parts := strings.SplitN(l[2:], "|", 2)
			if len(parts) > 0 {
				currentCommit = parts[0]
				totalCommits++
			}
			continue
		}
		if l == "" || currentCommit == "" {
			continue
		}

		// File stats line
		parts := strings.SplitN(l, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		path := parts[2]

		// Handle renames and filter
		cleanPaths := resolvePaths(path, fileExists, matcher)
		commits[currentCommit] = append(commits[currentCommit], cleanPaths...)
	}

	// 4. Calculate frequencies and co-occurrences
	fileCommits := make(map[string]int)              // How many commits file A appears in
	pairCoChanges := make(map[string]map[string]int) // How many commits A and B appear together in

	for _, files := range commits {
		// Dedup files in same commit (rare but possible with renames)
		uniqueFiles := make(map[string]bool)
		for _, f := range files {
			uniqueFiles[f] = true
		}

		fList := make([]string, 0, len(uniqueFiles))
		for f := range uniqueFiles {
			fileCommits[f]++
			fList = append(fList, f)
		}

		// Count pairs
		for i := 0; i < len(fList); i++ {
			for j := i + 1; j < len(fList); j++ {
				a, b := fList[i], fList[j]
				if a > b {
					a, b = b, a
				}
				if pairCoChanges[a] == nil {
					pairCoChanges[a] = make(map[string]int)
				}
				pairCoChanges[a][b]++
			}
		}
	}

	// 5. Calculate Jaccard scores
	type rawPair struct {
		a, b     string
		coChange int
		score    float64
	}
	var pairs []rawPair

	for a, targets := range pairCoChanges {
		for b, co := range targets {
			// J(A, B) = Co(A, B) / (C(A) + C(B) - Co(A, B))
			score := float64(co) / float64(fileCommits[a]+fileCommits[b]-co)
			if score >= threshold {
				pairs = append(pairs, rawPair{a, b, co, score})
			}
		}
	}

	// 6. Sort and limit
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].score != pairs[j].score {
			return pairs[i].score > pairs[j].score
		}
		return pairs[i].coChange > pairs[j].coChange
	})

	if len(pairs) > limit {
		pairs = pairs[:limit]
	}

	// 7. Format result
	result := schema.BlastRadiusResult{
		Summary: schema.BlastRadiusSummary{
			TotalCommits: totalCommits,
			TotalPairs:   len(pairs),
			Threshold:    threshold,
		},
		Pairs: make([]schema.BlastRadiusPair, 0, len(pairs)),
	}

	for _, p := range pairs {
		result.Pairs = append(result.Pairs, schema.BlastRadiusPair{
			Source:   p.a,
			Target:   p.b,
			Score:    p.score,
			CoChange: p.coChange,
		})
	}

	return result, nil
}

// resolvePaths replicates the logic in core/agg/agg.go for consistency.
func resolvePaths(path string, fileExists map[string]bool, matcher *schema.PathMatcher) []string {
	var candidates []string
	if !strings.Contains(path, " => ") {
		candidates = append(candidates, path)
	} else {
		// Handle rename format prefix{old => new}suffix or old => new
		if !strings.Contains(path, "{") {
			parts := strings.SplitN(path, " => ", 2)
			if len(parts) == 2 {
				candidates = append(candidates, parts[0], parts[1])
			}
		} else {
			braceStart := strings.Index(path, "{")
			braceEnd := strings.Index(path, "}")
			if braceStart != -1 && braceEnd != -1 && braceStart < braceEnd {
				prefix := path[:braceStart]
				suffix := path[braceEnd+1:]
				renamePart := path[braceStart+1 : braceEnd]
				if strings.Contains(renamePart, " => ") {
					renameParts := strings.SplitN(renamePart, " => ", 2)
					candidates = append(candidates, prefix+renameParts[0]+suffix, prefix+renameParts[1]+suffix)
				}
			}
		}
	}

	var result []string
	for _, c := range candidates {
		if fileExists[c] && !matcher.Match(c) {
			result = append(result, c)
		}
	}
	return result
}
