// main holds all of the core and entry logic for hotspot CLI.
package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// cached recent maps populated when a repo-wide recent aggregation is run
var (
	recentCommitsMapGlobal map[string]int
	recentChurnMapGlobal   map[string]int
)

// map[file] -> map[author]commitCount
var recentContribMapGlobal map[string]map[string]int

// FileMetrics represents the Git and file system metrics for a single file.
// It includes contribution statistics, commit history, size, age, and derived metrics
// used to determine the file's overall importance score.
type FileMetrics struct {
	Path               string // Relative path to the file in the repository
	UniqueContributors int    // Number of different authors who modified the file
	Commits            int    // Total number of commits affecting this file
	// Recent* fields measure activity within a recent window when requested
	RecentContributors int
	RecentCommits      int
	RecentChurn        int
	SizeBytes          int64     // Current size of the file in bytes
	AgeDays            int       // Age of the file in days since first commit
	Churn              int       // Total number of lines added/deleted plus number of commits
	Gini               float64   // Gini coefficient of commit distribution (0-1, lower is more even)
	FirstCommit        time.Time // Timestamp of the file's first commit
	Score              float64   // Computed importance score (0-100)
	// Breakdown holds the normalized contribution of each metric for debugging/tuning.
	Breakdown map[string]float64
}

// Config holds the runtime configuration for the analysis.
// It includes repository location, time range filters, and execution parameters.
type Config struct {
	RepoPath    string    // Absolute path to the Git repository
	StartTime   time.Time // Start of time range for commit analysis (zero = no limit)
	EndTime     time.Time // End of time range for commit analysis (zero = no limit)
	PathFilter  string    // Optional path prefix filter for files
	ResultLimit int       // Maximum number of files to show in results
	Workers     int       // Number of concurrent workers for analysis
	Mode        string    // Scoring mode: "hot" or "risk"
	Excludes    []string  // Path prefixes/suffixes to ignore
	Explain     bool      // If true, print per-file breakdown
	Precision   int       // Decimal precision for numeric columns (1 or 2)
	Output      string    // Output format: "text" (default) or "csv"
	CSVFile     string    // Optional path to write CSV output directly
	Follow      bool      // If true, re-run per-file analysis with `--follow` for the top -limit files
}

const (
	maxPathWidth     = 40
	maxLimitDefault  = 200
	defaultWorkers   = 8
	defaultLimit     = 10
	defaultPrecision = 1
)

// main is the entry point for the hotspot analyzer.
// It parses command line flags, analyzes the repository, and outputs ranked results.
func main() {
	cfg, err := parseFlags()
	if err != nil {
		fmt.Println("❌", err)
		os.Exit(1)
	}

	var files []string

	if !cfg.StartTime.IsZero() {
		// Run repo-wide aggregation first and use the files seen in that pass.
		fmt.Printf("🔎 Aggregating recent activity since %s (single repo-wide pass)...\n", cfg.StartTime.Format(time.RFC3339))
		if err := aggregateRecent(cfg); err != nil {
			fmt.Println("⚠️  Warning: could not aggregate recent activity:", err)
		}

		// Build file list from union of recent maps so we only analyze files touched since StartTime
		seen := make(map[string]bool)
		for k := range recentCommitsMapGlobal {
			seen[k] = true
		}
		for k := range recentChurnMapGlobal {
			seen[k] = true
		}
		for k := range recentContribMapGlobal {
			seen[k] = true
		}
		for f := range seen {
			// apply path filter and excludes
			if cfg.PathFilter != "" && !strings.HasPrefix(f, cfg.PathFilter) {
				continue
			}
			if shouldIgnore(f, cfg.Excludes) {
				continue
			}
			files = append(files, f)
		}

		if len(files) == 0 {
			fmt.Println("⚠️  No files with activity found in the requested window.")
			return
		}
	} else {
		files, err = listRepoFiles(cfg.RepoPath, cfg.PathFilter)
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

	results := analyzeRepo(cfg, files)
	ranked := rankFiles(results, cfg.ResultLimit)

	// If the user requested a follow-pass, re-analyze the top N files using
	// git --follow to account for renames/history and then re-rank.
	if cfg.Follow && len(ranked) > 0 {
		n := min(cfg.ResultLimit, len(ranked))
		fmt.Printf("🔁 Running --follow re-analysis for top %d files...\n", n)
		for i := range n {
			f := ranked[i]
			// re-analyze with follow enabled
			rean := analyzeFileCommon(cfg, f.Path, true)
			// preserve path but update metrics and score
			rean.Path = f.Path
			ranked[i] = rean
		}
		// re-rank after follow pass
		ranked = rankFiles(ranked, cfg.ResultLimit)
	}
	printResults(ranked, cfg)
}

// parseFlags processes command line arguments and returns a Config struct.
// It uses the standard flag package to handle options for controlling the analysis.
// Returns an error if required arguments are missing or invalid.
func parseFlags() (*Config, error) {
	cfg := &Config{Workers: defaultWorkers, EndTime: time.Now()}

	// Define flags
	limit := flag.Int("limit", defaultLimit, fmt.Sprintf("Number of files to display (default: %d, max: %d)", defaultLimit, maxLimitDefault))
	filter := flag.String("filter", "", "Filter files by path prefix")
	startDate := flag.String("start", "", "Start date in ISO8601 format (e.g., 2023-01-01T00:00:00Z)")
	endDate := flag.String("end", "", "End date in ISO8601 format (defaults to current time)")
	workers := flag.Int("workers", defaultWorkers, fmt.Sprintf("Number of concurrent workers (default: %d)", defaultWorkers))
	mode := flag.String("mode", "hot", "Scoring mode: hot, risk, complexity, stale, onboarding, ownership, security")
	exclude := flag.String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore (e.g. vendor,node_modules,*.min.js)")
	explain := flag.Bool("explain", false, "Print per-file component score breakdown (for debugging/tuning)")
	precision := flag.Int("precision", defaultPrecision, "Decimal precision for numeric columns (1 or 2)")
	output := flag.String("output", "text", "Output format: text (default) or csv")
	csvFile := flag.String("csv-file", "", "Optional path to write CSV output directly (overrides stdout)")
	follow := flag.Bool("follow", false, "Re-run per-file analysis with --follow for the top -limit files (slower but handles renames)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <repo-path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return nil, fmt.Errorf("repository path is required")
	}

	cfg.RepoPath = flag.Arg(0)
	if *limit > maxLimitDefault {
		return nil, fmt.Errorf("limit cannot exceed %d files", maxLimitDefault)
	}
	cfg.ResultLimit = *limit
	cfg.PathFilter = *filter
	cfg.Workers = *workers
	cfg.Mode = *mode
	cfg.Explain = *explain

	// default excludes
	defaults := []string{"vendor/", "node_modules/", "third_party/", ".min.js", ".min.css"}
	cfg.Excludes = defaults
	if *exclude != "" {
		parts := strings.Split(*exclude, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.Excludes = append(cfg.Excludes, p)
			}
		}
	}

	if *precision < 1 {
		*precision = 1
	}
	if *precision > 2 {
		*precision = 2
	}
	cfg.Precision = *precision
	cfg.Output = strings.ToLower(*output)
	cfg.CSVFile = *csvFile
	cfg.Follow = *follow

	// Parse start date if provided
	if *startDate != "" {
		t, err := time.Parse(time.RFC3339, *startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start date: %v", err)
		}
		cfg.StartTime = t
	}

	// Parse end date if provided
	if *endDate != "" {
		t, err := time.Parse(time.RFC3339, *endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end date: %v", err)
		}
		cfg.EndTime = t
	}

	return cfg, nil
}

// listRepoFiles returns a list of all tracked files in the Git repository.
// If pathFilter is non-empty, only files whose paths start with the filter are included.
// Returns an error if the git command fails or the repository is invalid.
func listRepoFiles(repoPath, pathFilter string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if pathFilter != "" {
		filtered := []string{}
		for _, f := range lines {
			if strings.HasPrefix(f, pathFilter) {
				filtered = append(filtered, f)
			}
		}
		return filtered, nil
	}
	return lines, nil
}

// analyzeRepo processes all files in parallel using a worker pool.
// It spawns cfg.Workers number of goroutines to analyze files concurrently
// and aggregates their results into a single slice of FileMetrics.
func analyzeRepo(cfg *Config, files []string) []FileMetrics {
	// Filter files according to excludes and path filter before analysis
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if shouldIgnore(f, cfg.Excludes) {
			continue
		}
		filtered = append(filtered, f)
	}

	results := make([]FileMetrics, 0, len(filtered))
	fileCh := make(chan string, len(filtered))
	resultCh := make(chan FileMetrics, len(files))
	var wg sync.WaitGroup

	for range cfg.Workers {
		wg.Go(func() {
			for f := range fileCh {
				metrics := analyzeFileCommon(cfg, f, false)
				resultCh <- metrics
			}
		})
	}

	for _, f := range filtered {
		fileCh <- f
	}
	close(fileCh)

	wg.Wait()
	close(resultCh)

	for r := range resultCh {
		results = append(results, r)
	}

	return results
}

// analyzeFileCommon computes all metrics for a single file in the repository.
// It gathers Git history data (commits, authors, dates), file size, and calculates
// derived metrics like churn and the Gini coefficient of author contributions.
// The analysis is constrained by the time range in cfg if specified.
// If useFollow is true, git --follow is used to track file renames.
func analyzeFileCommon(cfg *Config, path string, useFollow bool) FileMetrics {
	repo := cfg.RepoPath
	var metrics FileMetrics
	metrics.Path = path

	// Build git log command with optional --follow
	args := []string{"-C", repo, "log"}
	if useFollow {
		args = append(args, "--follow")
	}
	if !cfg.StartTime.IsZero() {
		args = append(args, "--since="+cfg.StartTime.Format(time.RFC3339))
	}
	args = append(args, "--pretty=format:%an,%ad", "--date=iso", "--", path)

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return metrics
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	contribCount := map[string]int{}
	totalCommits := 0
	var firstCommit time.Time

	for _, l := range lines {
		if l == "" {
			continue
		}
		parts := strings.SplitN(l, ",", 2)
		if len(parts) < 2 {
			continue
		}
		author := parts[0]
		dateStr := parts[1]
		date, _ := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if !cfg.StartTime.IsZero() && date.Before(cfg.StartTime) {
			continue
		}
		if !cfg.EndTime.IsZero() && date.After(cfg.EndTime) {
			continue
		}
		contribCount[author]++
		totalCommits++
		if firstCommit.IsZero() || date.Before(firstCommit) {
			firstCommit = date
		}
	}

	metrics.UniqueContributors = len(contribCount)
	metrics.Commits = totalCommits
	metrics.FirstCommit = firstCommit
	// If we couldn't determine a first commit (empty history or parsing issues),
	// set AgeDays to 0 instead of a huge value to avoid skewing the score.
	if firstCommit.IsZero() {
		metrics.AgeDays = 0
	} else {
		metrics.AgeDays = int(time.Since(firstCommit).Hours() / 24)
	}

	// File size
	info, err := os.Stat(filepath.Join(repo, path))
	if err == nil {
		metrics.SizeBytes = info.Size()
	}

	// Churn
	churnArgs := []string{"-C", repo, "log"}
	if useFollow {
		churnArgs = append(churnArgs, "--follow")
	}
	if !cfg.StartTime.IsZero() {
		churnArgs = append(churnArgs, "--since="+cfg.StartTime.Format(time.RFC3339))
	}
	churnArgs = append(churnArgs, "--numstat", "--", path)

	cmd = exec.Command("git", churnArgs...)
	out, _ = cmd.Output()
	reader := csv.NewReader(bytes.NewReader(out))
	reader.Comma = '\t'
	totalChanges := 0
	for {
		rec, err := reader.Read()
		if err != nil {
			break
		}
		if len(rec) >= 2 {
			add, _ := strconv.Atoi(rec[0])
			del, _ := strconv.Atoi(rec[1])
			totalChanges += add + del
		}
	}
	metrics.Churn = totalChanges + totalCommits

	// If we have global recent maps (populated by a single repo-wide pass),
	// use them to populate recent metrics for this file without doing per-file
	// git --follow invocations.
	if recentCommitsMapGlobal != nil {
		if v, ok := recentCommitsMapGlobal[path]; ok {
			metrics.RecentCommits = v
		}
	}
	if recentChurnMapGlobal != nil {
		if v, ok := recentChurnMapGlobal[path]; ok {
			metrics.RecentChurn = v
		}
	}
	if recentContribMapGlobal != nil {
		if m, ok := recentContribMapGlobal[path]; ok {
			metrics.RecentContributors = len(m)
		}
	}

	// Gini coefficient for author diversity
	values := make([]float64, 0, len(contribCount))
	for _, c := range contribCount {
		values = append(values, float64(c))
	}
	metrics.Gini = gini(values)

	// Composite score (0–100)
	metrics.Score = computeScore(&metrics, cfg.Mode)
	return metrics
}

// shouldIgnore returns true if the given path matches any of the exclude patterns.
// It supports simple glob patterns (using filepath.Match) when the pattern
// contains wildcard characters (*, ?, [ ]). Patterns ending with '/' are treated
// as prefixes. Patterns starting with '.' are treated as suffix (extension) matches.
// A user can provide patterns like "vendor/", "node_modules/", "*.min.js".
func shouldIgnore(path string, excludes []string) bool {
	for _, ex := range excludes {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}

		// If the pattern contains glob characters, try filepath.Match.
		if strings.ContainsAny(ex, "*?[") || strings.Contains(ex, "**") {
			pat := strings.ReplaceAll(ex, "**", "*")
			if ok, err := filepath.Match(pat, path); err == nil && ok {
				return true
			}
			// Also try matching against the base filename (e.g. *.min.js)
			if ok, err := filepath.Match(pat, filepath.Base(path)); err == nil && ok {
				return true
			}
			continue
		}

		// Handle prefix, suffix, or substring matches
		switch {
		case strings.HasSuffix(ex, "/"):
			if strings.HasPrefix(path, ex) {
				return true
			}
		case strings.HasPrefix(ex, "."):
			if strings.HasSuffix(path, ex) {
				return true
			}
		case strings.Contains(path, ex):
			return true
		}
	}
	return false
}

// aggregateRecent performs a single repository-wide git log since cfg.StartTime
// and aggregates per-file recent commits, churn and contributors. It avoids
// expensive per-file --follow calls and is fast even on large repositories.
func aggregateRecent(cfg *Config) error {
	if cfg.StartTime.IsZero() {
		return nil
	}

	since := cfg.StartTime.Format(time.RFC3339)
	cmd := exec.Command("git", "-C", cfg.RepoPath, "log", "--since="+since, "--numstat", "--pretty=format:--%H|%an")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	recentCommitsMapGlobal = make(map[string]int)
	recentChurnMapGlobal = make(map[string]int)
	recentContribMapGlobal = make(map[string]map[string]int)

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

// gini calculates the Gini coefficient for a set of values.
// The Gini coefficient measures inequality in a distribution, ranging from 0 (perfect equality)
// to 1 (perfect inequality). It's used here to measure how evenly distributed commits are
// among contributors.
func gini(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)
	if mean == 0 {
		return 0
	}

	var diffSum float64
	for i := range n {
		for j := range n {
			diffSum += math.Abs(values[i] - values[j])
		}
	}

	g := diffSum / (2 * float64(n*n) * mean)
	return math.Min(math.Max(g, 0), 1) // clamp to [0,1]
}

// computeScore calculates a file's importance score (0-100) based on its metrics.
// Supports multiple scoring modes:
// - hot: Activity hotspots (high commits, churn, contributors)
// - risk: Knowledge risk/bus factor (few contributors, high inequality)
// - complexity: Technical debt candidates (large, old, high churn)
// - stale: Maintenance debt (important but untouched)
// - onboarding: Files new developers should learn
// - ownership: Healthy ownership patterns
// - security: Security-critical file detection
func computeScore(m *FileMetrics, mode string) float64 {
	// Tunable maxima to normalize metrics. These are conservative defaults
	// chosen to avoid a few outliers dominating the score. Consider making
	// these flags or config values in the future.
	const (
		maxContrib = 20.0   // contributors beyond this saturate
		maxCommits = 500.0  // commits beyond this saturate
		maxSizeKB  = 500.0  // file size in KB beyond this saturate
		maxAgeDays = 3650.0 // ~10 years
		maxChurn   = 5000.0 // total added+deleted lines
	)

	clamp01 := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}

	// Normalize each metric into [0,1]
	nContrib := clamp01(float64(m.UniqueContributors) / maxContrib)
	nCommits := clamp01(float64(m.Commits) / maxCommits)
	nSize := clamp01((float64(m.SizeBytes) / 1024.0) / maxSizeKB)
	// Age is tricky: very old files shouldn't always be treated as critical.
	// We use a log-like scaling (but simple) to give diminishing returns.
	nAge := clamp01(math.Log1p(float64(m.AgeDays)) / math.Log1p(maxAgeDays))
	nChurn := clamp01(float64(m.Churn) / maxChurn)
	// Gini: lower is healthier; invert and clamp
	nGini := clamp01(1.0 - m.Gini)

	// For stale mode, we need inverse of recent activity
	nRecentCommits := clamp01(float64(m.RecentCommits) / 50.0) // assume 50 recent commits is high activity

	// Prepare breakdown map to return component contributions
	breakdown := make(map[string]float64)
	var raw float64

	switch strings.ToLower(mode) {
	case "risk":
		// Knowledge-risk focused scoring: prioritize concentration and bus-factor
		invContrib := clamp01(1.0 - (float64(m.UniqueContributors) / maxContrib))
		giniRaw := clamp01(m.Gini)

		const (
			wInvContrib = 0.32
			wGini       = 0.28
			wAgeRisk    = 0.18
			wSizeRisk   = 0.12
			wChurnRisk  = 0.06
			wCommRisk   = 0.04
		)
		breakdown["inv_contrib"] = wInvContrib * invContrib
		breakdown["gini"] = wGini * giniRaw
		breakdown["age"] = wAgeRisk * nAge
		breakdown["size"] = wSizeRisk * nSize
		breakdown["churn"] = wChurnRisk * nChurn
		breakdown["commits"] = wCommRisk * nCommits
		raw = breakdown["inv_contrib"] + breakdown["gini"] + breakdown["age"] + breakdown["size"] + breakdown["churn"] + breakdown["commits"]

	case "complexity":
		// Technical debt focus: large, old files with high total churn
		const (
			wSizeComplex  = 0.35
			wAgeComplex   = 0.25
			wChurnComplex = 0.25
			wCommComplex  = 0.10
			wContribLow   = 0.05 // prefer fewer contributors (concentrated complexity)
		)
		// Invert recent activity - we want files that aren't being actively worked on
		invRecentCommits := clamp01(1.0 - nRecentCommits)
		breakdown["size"] = wSizeComplex * nSize
		breakdown["age"] = wAgeComplex * nAge
		breakdown["churn"] = wChurnComplex * nChurn
		breakdown["commits"] = wCommComplex * nCommits
		breakdown["low_recent"] = wContribLow * invRecentCommits
		raw = breakdown["size"] + breakdown["age"] + breakdown["churn"] + breakdown["commits"] + breakdown["low_recent"]

	case "stale":
		// Maintenance debt: important but haven't been touched recently
		invRecentCommits := clamp01(1.0 - nRecentCommits)
		const (
			wAgeStale       = 0.30
			wSizeStale      = 0.25
			wInvRecentStale = 0.25 // penalize recent activity
			wCommitsStale   = 0.15 // historically important
			wContribStale   = 0.05
		)
		breakdown["age"] = wAgeStale * nAge
		breakdown["size"] = wSizeStale * nSize
		breakdown["inv_recent"] = wInvRecentStale * invRecentCommits
		breakdown["commits"] = wCommitsStale * nCommits
		breakdown["contrib"] = wContribStale * nContrib
		raw = breakdown["age"] + breakdown["size"] + breakdown["inv_recent"] + breakdown["commits"] + breakdown["contrib"]

	case "onboarding":
		// Files new developers should learn: active, well-maintained, moderate complexity
		const (
			wContribOnboard = 0.30
			wCommitsOnboard = 0.25
			wSizeOnboard    = 0.20 // prefer moderate size (not too large)
			wAgeOnboard     = 0.15 // some maturity is good
			wGiniOnboard    = 0.10 // even distribution is healthy
		)
		// For size, prefer moderate - penalize both very small and very large
		moderateSize := 1.0 - math.Abs(nSize-0.4) // peak at 40% of max
		breakdown["contrib"] = wContribOnboard * nContrib
		breakdown["commits"] = wCommitsOnboard * nCommits
		breakdown["size"] = wSizeOnboard * clamp01(moderateSize)
		breakdown["age"] = wAgeOnboard * nAge
		breakdown["gini"] = wGiniOnboard * nGini
		raw = breakdown["contrib"] + breakdown["commits"] + breakdown["size"] + breakdown["age"] + breakdown["gini"]

	case "ownership":
		// Healthy ownership patterns: even distribution, steady activity
		const (
			wGiniOwnership    = 0.35 // reward even distribution
			wContribOwnership = 0.25 // moderate number of contributors
			wCommitsOwnership = 0.20
			wChurnOwnership   = 0.10 // steady but not chaotic
			wAgeOwnership     = 0.10
		)
		// Prefer moderate contributors (not too few, not too many)
		moderateContrib := 1.0 - math.Abs(nContrib-0.5)
		breakdown["gini"] = wGiniOwnership * nGini
		breakdown["contrib"] = wContribOwnership * clamp01(moderateContrib)
		breakdown["commits"] = wCommitsOwnership * nCommits
		breakdown["churn"] = wChurnOwnership * nChurn
		breakdown["age"] = wAgeOwnership * nAge
		raw = breakdown["gini"] + breakdown["contrib"] + breakdown["commits"] + breakdown["churn"] + breakdown["age"]

	case "security":
		// Security-critical file detection
		invContrib := clamp01(1.0 - (float64(m.UniqueContributors) / maxContrib))
		giniRaw := clamp01(m.Gini)

		// Detect security-related keywords in path
		securityBoost := 0.0
		lowerPath := strings.ToLower(m.Path)
		securityKeywords := []string{"auth", "password", "token", "secret", "crypto", "security", "login", "session", "oauth", "jwt", "credential", "permission", "acl", "rbac"}
		for _, keyword := range securityKeywords {
			if strings.Contains(lowerPath, keyword) {
				securityBoost = 1.0
				break
			}
		}

		const (
			wSecurityBoost = 0.30 // path-based detection
			wAgeSecurity   = 0.25 // old security code = more exposure
			wInvContribSec = 0.20 // fewer eyes = risk
			wGiniSecurity  = 0.15 // concentration risk
			wSizeSecurity  = 0.10
		)
		breakdown["sec_boost"] = wSecurityBoost * securityBoost
		breakdown["age"] = wAgeSecurity * nAge
		breakdown["inv_contrib"] = wInvContribSec * invContrib
		breakdown["gini"] = wGiniSecurity * giniRaw
		breakdown["size"] = wSizeSecurity * nSize
		raw = breakdown["sec_boost"] + breakdown["age"] + breakdown["inv_contrib"] + breakdown["gini"] + breakdown["size"]

	default:
		// Hotspot scoring (default): where activity and volatility are concentrated
		const (
			wContrib = 0.18
			wCommits = 0.28 // many code changes
			wSize    = 0.16
			wAge     = 0.08
			wChurn   = 0.26 // plenty of churn in the code
			wGini    = 0.04
		)
		breakdown["contrib"] = wContrib * nContrib
		breakdown["commits"] = wCommits * nCommits
		breakdown["size"] = wSize * nSize
		breakdown["age"] = wAge * nAge
		breakdown["churn"] = wChurn * nChurn
		breakdown["gini"] = wGini * nGini
		raw = breakdown["contrib"] + breakdown["commits"] + breakdown["size"] + breakdown["age"] + breakdown["churn"] + breakdown["gini"]
	}

	score := raw * 100.0
	// If risk mode and this looks like a test file, slightly reduce score since
	// tests often have narrow contributors and shouldn't be first-class risks.
	if strings.ToLower(mode) == "risk" {
		if strings.Contains(m.Path, "_test") || strings.HasSuffix(m.Path, "_test.go") {
			score *= 0.75
		}
	}
	// Save breakdown (scaled to percent contributions) in the metrics for explain mode.
	if m.Breakdown == nil {
		m.Breakdown = make(map[string]float64)
	}
	for k, v := range breakdown {
		m.Breakdown[k] = v * 100.0
	}

	return score
}

// rankFiles sorts files by their importance score in descending order
// and returns the top 'limit' files. If limit is greater than the number
// of files, all files are returned in sorted order.
func rankFiles(files []FileMetrics, limit int) []FileMetrics {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Score > files[j].Score
	})
	if len(files) > limit {
		return files[:limit]
	}
	return files
}

// truncatePath truncates a file path to a maximum width with ellipsis prefix.
func truncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}

// selectOutputFile returns the appropriate file handle for CSV output.
func selectOutputFile(cfg *Config) *os.File {
	if cfg.CSVFile != "" {
		if file, err := os.Create(cfg.CSVFile); err == nil {
			return file
		}
		fmt.Fprintf(os.Stderr, "warning: cannot open csv file %s: falling back to stdout\n", cfg.CSVFile)
	}
	return os.Stdout
}

// writeCSVResults writes the analysis results in CSV format.
func writeCSVResults(w *csv.Writer, files []FileMetrics, fmtFloat func(float64) string, intFmt string) {
	// CSV header
	_ = w.Write([]string{"rank", "file", "score", "label", "contributors", "commits", "size_kb", "age_days", "churn", "gini", "first_commit"})
	for i, f := range files {
		rec := []string{
			strconv.Itoa(i + 1),
			f.Path,
			fmtFloat(f.Score),
			labelColor(f.Score),
			fmt.Sprintf(intFmt, f.UniqueContributors),
			fmt.Sprintf(intFmt, f.Commits),
			fmtFloat(float64(f.SizeBytes) / 1024.0),
			fmt.Sprintf(intFmt, f.AgeDays),
			fmt.Sprintf(intFmt, f.Churn),
			fmtFloat(f.Gini),
			f.FirstCommit.Format("2006-01-02"),
		}
		_ = w.Write(rec)
	}
}

// printResults outputs the analysis results in a formatted table.
// For each file it shows rank, path (truncated if needed), importance score,
// criticality label, and all individual metrics that contribute to the score.
func printResults(files []FileMetrics, cfg *Config) {
	explain := cfg.Explain
	precision := cfg.Precision
	outFmt := cfg.Output

	// helper format strings for numbers
	numFmt := "%.*f"
	intFmt := "%d"
	// closure to format floats with the configured precision
	fmtFloat := func(v float64) string {
		return fmt.Sprintf(numFmt, precision, v)
	}

	// If CSV output requested, skip printing the human-readable table
	if outFmt == "csv" {
		file := selectOutputFile(cfg)
		w := csv.NewWriter(file)
		writeCSVResults(w, files, fmtFloat, intFmt)
		w.Flush()
		if file != os.Stdout {
			_ = file.Close()
			fmt.Fprintf(os.Stderr, "wrote CSV to %s\n", cfg.CSVFile)
		}
		return
	}

	// Define columns and initial header names
	headers := []string{"Rank", "File", "Score", "Label", "Contrib", "Commits", "Size(KB)", "Age(d)", "Churn", "Gini", "First Commit"}

	// Compute column widths from headers and data
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	for idx, f := range files {
		// Rank width
		rankStr := strconv.Itoa(idx + 1)
		if len(rankStr) > widths[0] {
			widths[0] = len(rankStr)
		}
		// File path width
		p := truncatePath(f.Path, maxPathWidth)
		if len(p) > widths[1] {
			widths[1] = len(p)
		}
		// Score
		s := fmt.Sprintf("%.1f", f.Score)
		if len(s) > widths[2] {
			widths[2] = len(s)
		}
		// Label
		lbl := labelColor(f.Score)
		if len(lbl) > widths[3] {
			widths[3] = len(lbl)
		}
		// Other numeric columns
		nums := []string{
			fmt.Sprintf("%d", f.UniqueContributors),
			fmt.Sprintf("%d", f.Commits),
			fmt.Sprintf("%.1f", float64(f.SizeBytes)/1024.0),
			fmt.Sprintf("%d", f.AgeDays),
			fmt.Sprintf("%d", f.Churn),
			fmt.Sprintf("%.2f", f.Gini),
			f.FirstCommit.Format("2006-01-02"),
		}
		for i, n := range nums {
			col := i + 4 // starts at Contrib column
			if len(n) > widths[col] {
				widths[col] = len(n)
			}
		}
	}

	// Build format string dynamically
	fmts := []string{
		fmt.Sprintf("%%%ds", widths[0]),
		fmt.Sprintf("%%-%ds", widths[1]),
		fmt.Sprintf("%%%ds", widths[2]),
		fmt.Sprintf("%%-%ds", widths[3]),
	}
	// Numeric right-aligned columns
	for i := 4; i < len(headers)-1; i++ {
		fmts = append(fmts, fmt.Sprintf("%%%ds", widths[i]))
	}
	// Last column (date) left-aligned
	fmts = append(fmts, fmt.Sprintf("%%-%ds", widths[len(headers)-1]))

	// Compose header line
	var headerParts []string
	for i, h := range headers {
		headerParts = append(headerParts, fmt.Sprintf(fmts[i], h))
	}

	// Compose separator line
	sepParts := make([]string, len(headers))
	for i := range headers {
		sepParts[i] = strings.Repeat("-", widths[i])
	}

	// Print human-readable header and separator
	fmt.Println(strings.Join(headerParts, "  "))
	fmt.Println(strings.Join(sepParts, "  "))

	// Print rows
	for i, f := range files {
		p := truncatePath(f.Path, maxPathWidth)
		rowVals := []any{
			strconv.Itoa(i + 1),
			p,
			fmtFloat(f.Score),
			labelColor(f.Score),
			fmt.Sprintf(intFmt, f.UniqueContributors),
			fmt.Sprintf(intFmt, f.Commits),
			fmtFloat(float64(f.SizeBytes) / 1024.0),
			fmt.Sprintf(intFmt, f.AgeDays),
			fmt.Sprintf(intFmt, f.Churn),
			fmtFloat(f.Gini),
			f.FirstCommit.Format("2006-01-02"),
		}

		// Build formatted row using fmts
		var parts []string
		for j, rv := range rowVals {
			parts = append(parts, fmt.Sprintf(fmts[j], fmt.Sprint(rv)))
		}
		fmt.Println(strings.Join(parts, "  "))

		// Explain breakdown if requested
		if explain && len(f.Breakdown) > 0 {
			fmt.Println()
			fmt.Print("      Breakdown:")
			// print key/value pairs sorted by keys for deterministic output
			keys := []string{"contrib", "commits", "size", "age", "churn", "gini", "inv_contrib"}
			for _, k := range keys {
				if v, ok := f.Breakdown[k]; ok {
					fmt.Printf(" %s=%.1f%%", k, v)
				}
			}
			fmt.Println()
			fmt.Println()
		}
	}
}

// labelColor returns a text label indicating the criticality level
// based on the file's importance score:
// - Critical (≥80)
// - High (≥60)
// - Moderate (≥40)
// - Low (<40)
func labelColor(score float64) string {
	switch {
	case score >= 80:
		return "Critical"
	case score >= 60:
		return "High"
	case score >= 40:
		return "Moderate"
	default:
		return "Low"
	}
}
