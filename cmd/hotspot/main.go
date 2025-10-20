// main holds all of the core and entry logic for hotspot CLI.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal"
	"github.com/huangsam/hotspot/schema"
)

// main is the entry point for the hotspot analyzer.
// It parses command line flags, analyzes the repository, and outputs ranked results.
func main() {
	cfg, err := schema.ParseFlags()
	if err != nil {
		fmt.Println("❌", err)
		os.Exit(1)
	}

	var files []string

	if !cfg.StartTime.IsZero() {
		// Run repo-wide aggregation first and use the files seen in that pass.
		fmt.Printf("🔎 Aggregating recent activity since %s (single repo-wide pass)...\n", cfg.StartTime.Format(time.RFC3339))
		if err := core.AggregateRecent(cfg); err != nil {
			fmt.Println("⚠️  Warning: could not aggregate recent activity:", err)
		}

		// Build file list from union of recent maps so we only analyze files touched since StartTime
		seen := make(map[string]bool)
		for k := range core.RecentCommitsMapGlobal {
			seen[k] = true
		}
		for k := range core.RecentChurnMapGlobal {
			seen[k] = true
		}
		for k := range core.RecentContribMapGlobal {
			seen[k] = true
		}
		for f := range seen {
			// apply path filter and excludes
			if cfg.PathFilter != "" && !strings.HasPrefix(f, cfg.PathFilter) {
				continue
			}
			if internal.ShouldIgnore(f, cfg.Excludes) {
				continue
			}
			files = append(files, f)
		}

		if len(files) == 0 {
			fmt.Println("⚠️  No files with activity found in the requested window.")
			return
		}
	} else {
		files, err = core.ListRepoFiles(cfg.RepoPath, cfg.PathFilter)
		if err != nil {
			fmt.Println("❌ Error listing files:", err)
			os.Exit(1)
		}
		if len(files) == 0 {
			fmt.Println("⚠️  No files found in repository.")
			return
		}
	}

	fmt.Printf("🧠 hotspot: Analyzing %s\n", cfg.RepoPath)
	fmt.Printf("📅 Range: %s → %s\n\n", cfg.StartTime.Format(time.RFC3339), cfg.EndTime.Format(time.RFC3339))

	results := core.AnalyzeRepo(cfg, files)
	ranked := core.RankFiles(results, cfg.ResultLimit)

	// If the user requested a follow-pass, re-analyze the top N files using
	// git --follow to account for renames/history and then re-rank.
	if cfg.Follow && len(ranked) > 0 {
		n := min(cfg.ResultLimit, len(ranked))
		fmt.Printf("🔁 Running --follow re-analysis for top %d files...\n", n)
		for i := range n {
			f := ranked[i]
			// re-analyze with follow enabled
			rean := core.AnalyzeFileCommon(cfg, f.Path, true)
			// preserve path but update metrics and score
			rean.Path = f.Path
			ranked[i] = rean
		}
		// re-rank after follow pass
		ranked = core.RankFiles(ranked, cfg.ResultLimit)
	}
	internal.PrintResults(ranked, cfg)
}
