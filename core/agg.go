package core

import (
	"strconv"
	"strings"
	"time"

	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// AggregateRecent performs a single repository-wide git log since cfg.StartTime
// and aggregates per-file recent commits, churn and contributors. It avoids
// expensive per-file --follow calls and is fast even on large repositories.
func AggregateRecent(cfg *schema.Config) error {
	if cfg.StartTime.IsZero() {
		return nil
	}

	since := cfg.StartTime.Format(time.RFC3339)
	out, err := internal.RunGitCommand(cfg.RepoPath, "log", "--since="+since, "--numstat", "--pretty=format:--%H|%an")
	if err != nil {
		return err
	}

	recentCommitsMapGlobal := schema.GetRecentCommitsMapGlobal()
	recentChurnMapGlobal := schema.GetRecentChurnMapGlobal()
	recentContribMapGlobal := schema.GetRecentContribMapGlobal()

	lines := strings.Split(string(out), "\n")
	var currentAuthor string
	for _, l := range lines {
		if strings.HasPrefix(l, "--") {
			// commit header
			parts := strings.SplitN(l[2:], "|", 2)
			if len(parts) == 2 {
				currentAuthor = parts[1]
			} else {
				currentAuthor = ""
			}
			continue
		}
		if strings.TrimSpace(l) == "" {
			continue
		}
		parts := strings.SplitN(l, "\t", 3)
		if len(parts) < 3 {
			continue
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
		recentChurnMapGlobal[path] += add + del
		recentCommitsMapGlobal[path]++
		if currentAuthor != "" {
			if recentContribMapGlobal[path] == nil {
				recentContribMapGlobal[path] = make(map[string]int)
			}
			recentContribMapGlobal[path][currentAuthor]++
		}
	}
	return nil
}
