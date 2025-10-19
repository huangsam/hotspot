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

// FileMetrics represents the Git and file system metrics for a single file.
// It includes contribution statistics, commit history, size, age, and derived metrics
// used to determine the file's overall importance score.
type FileMetrics struct {
	Path               string    // Relative path to the file in the repository
	UniqueContributors int       // Number of different authors who modified the file
	Commits            int       // Total number of commits affecting this file
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
}

// main is the entry point for the critical-files analyzer.
// It parses command line flags, analyzes the repository, and outputs ranked results.
func main() {
	cfg, err := parseFlags()
	if err != nil {
		fmt.Println("❌", err)
		os.Exit(1)
	}

	files, err := listRepoFiles(cfg.RepoPath, cfg.PathFilter)
	if err != nil {
		fmt.Println("❌ Error listing files:", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("⚠️  No files found in repository.")
		return
	}

	fmt.Printf("🧠 critical-files: Analyzing %s\n", cfg.RepoPath)
	fmt.Printf("📅 Range: %s → %s\n\n", cfg.StartTime.Format(time.RFC3339), cfg.EndTime.Format(time.RFC3339))

	results := analyzeRepo(cfg, files)
	ranked := rankFiles(results, cfg.ResultLimit)
	printResults(ranked, cfg)
}

// parseFlags processes command line arguments and returns a Config struct.
// It uses the standard flag package to handle options for controlling the analysis.
// Returns an error if required arguments are missing or invalid.
func parseFlags() (*Config, error) {
	cfg := &Config{Workers: 8, EndTime: time.Now()}

	const maxLimit = 200 // Maximum number of files that can be analyzed

	// Define flags
	limit := flag.Int("limit", 10, fmt.Sprintf("Number of files to display (default: 10, max: %d)", maxLimit))
	filter := flag.String("filter", "", "Filter files by path prefix")
	startDate := flag.String("start", "", "Start date in ISO8601 format (e.g., 2023-01-01T00:00:00Z)")
	endDate := flag.String("end", "", "End date in ISO8601 format (defaults to current time)")
	workers := flag.Int("workers", 8, "Number of concurrent workers (default: 8)")
	mode := flag.String("mode", "hot", "Scoring mode: \"hot\" (hotspots) or \"risk\" (knowledge risk)")
	exclude := flag.String("exclude", "", "Comma-separated list of path prefixes or patterns to ignore (e.g. vendor,node_modules,*.min.js)")
	explain := flag.Bool("explain", false, "Print per-file component score breakdown (for debugging/tuning)")
	precision := flag.Int("precision", 1, "Decimal precision for numeric columns (1 or 2)")
	output := flag.String("output", "text", "Output format: text (default) or csv")
	csvFile := flag.String("csv-file", "", "Optional path to write CSV output directly (overrides stdout)")
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
	if *limit > maxLimit {
		return nil, fmt.Errorf("limit cannot exceed %d files", maxLimit)
	}
	cfg.ResultLimit = *limit
	cfg.PathFilter = *filter
	cfg.Workers = *workers
	cfg.Mode = *mode
	cfg.Explain = *explain
	// Explain flag
	if *explain {
		// store it in the Excludes slice as a sentinel? Better to add a field; we will add it below.
	}
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

	// Apply remaining flags (always, not only when excludes are present)
	if *precision < 1 {
		*precision = 1
	}
	if *precision > 2 {
		*precision = 2
	}
	cfg.Precision = *precision
	cfg.Output = strings.ToLower(*output)
	cfg.CSVFile = *csvFile

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

		// store explain in config by using a hidden field: append to Excludes as sentinel is hacky;
		// instead, add an Explain bool to Config (we'll add the field now)
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

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileCh {
				metrics := analyzeFile(cfg, f)
				resultCh <- metrics
			}
		}()
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

// analyzeFile computes all metrics for a single file in the repository.
// It gathers Git history data (commits, authors, dates), file size, and calculates
// derived metrics like churn and the Gini coefficient of author contributions.
// The analysis is constrained by the time range in cfg if specified.
func analyzeFile(cfg *Config, path string) FileMetrics {
	repo := cfg.RepoPath
	var metrics FileMetrics
	metrics.Path = path

	// Unique contributors + commits within range
	cmd := exec.Command("git", "-C", repo, "log", "--follow", "--pretty=format:%an,%ad", "--date=iso", "--", path)
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
	cmd = exec.Command("git", "-C", repo, "log", "--follow", "--numstat", "--", path)
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

	// (recent-window metrics removed)

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

// shouldIgnore returns true if the given path matches any of the exclude patterns/prefixes.
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
			pat := ex
			// filepath.Match doesn't support ** the same way shells do; as a
			// practical approximation, convert double-star to single-star.
			if strings.Contains(pat, "**") {
				pat = strings.ReplaceAll(pat, "**", "*")
			}
			if ok, err := filepath.Match(pat, path); err == nil && ok {
				return true
			}
			// Also try matching against the base filename (e.g. *.min.js)
			if ok, err := filepath.Match(pat, filepath.Base(path)); err == nil && ok {
				return true
			}
			continue
		}

		// Trailing slash: prefix match
		if strings.HasSuffix(ex, "/") {
			if strings.HasPrefix(path, ex) {
				return true
			}
			continue
		}

		// Leading dot: extension/suffix match
		if strings.HasPrefix(ex, ".") {
			if strings.HasSuffix(path, ex) {
				return true
			}
			continue
		}

		// Fallback: substring match
		if strings.Contains(path, ex) {
			return true
		}
	}
	return false
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
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			diffSum += math.Abs(values[i] - values[j])
		}
	}

	g := diffSum / (2 * float64(n*n) * mean)
	return math.Min(math.Max(g, 0), 1) // clamp to [0,1]
}

// computeScore calculates a file's importance score (0-100) based on its metrics.
// The score is a weighted sum of normalized metrics:
// - Number of contributors (15%)
// - Number of commits (25%)
// - File size (15%)
// - File age (15%)
// - Code churn (25%)
// - Author distribution evenness (5%)
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

	// Prepare breakdown map to return component contributions
	breakdown := make(map[string]float64)
	var raw float64
	switch strings.ToLower(mode) {
	case "risk":
		// Knowledge-risk focused scoring: prioritize concentration and bus-factor
		// For risk mode we treat contributor concentration (low number of distinct
		// contributors and high Gini) as higher risk.
		// We'll invert some signals: fewer contributors => higher risk, higher
		// Gini => higher risk.
		invContrib := clamp01(1.0 - (float64(m.UniqueContributors) / maxContrib))
		// Use Gini directly as inequality metric (higher => worse)
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
	default:
		// Hotspot scoring (default): where activity and volatility are concentrated
		const (
			wContrib = 0.18
			wCommits = 0.28
			wSize    = 0.16
			wAge     = 0.08
			wChurn   = 0.26
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

	// Additional modes
	switch strings.ToLower(mode) {
	case "bus-factor":
		// Emphasize single-ownership and inequality
		const (
			wInvContrib = 0.50
			wGini       = 0.35
			wCommits    = 0.08
			wAge        = 0.07
		)
		invContrib := clamp01(1.0 - (float64(m.UniqueContributors) / maxContrib))
		breakdown = map[string]float64{
			"inv_contrib": wInvContrib * invContrib,
			"gini":        wGini * m.Gini,
			"commits":     wCommits * nCommits,
			"age":         wAge * nAge,
		}
		raw = breakdown["inv_contrib"] + breakdown["gini"] + breakdown["commits"] + breakdown["age"]

	case "maintainability":
		// Surface large, frequently-changing files as refactor candidates
		const (
			wSize  = 0.30
			wChurn = 0.28
			wComm  = 0.18
			wGini  = 0.12
			wAgeM  = 0.12
		)
		breakdown = map[string]float64{
			"size":    wSize * nSize,
			"churn":   wChurn * nChurn,
			"commits": wComm * nCommits,
			"gini":    wGini * nGini,
			"age":     wAgeM * nAge,
		}
		raw = breakdown["size"] + breakdown["churn"] + breakdown["commits"] + breakdown["gini"] + breakdown["age"]

		// recent-activity mode removed
	}

	score := raw * 100.0
	// If risk mode and this looks like a test file, slightly reduce score since
	// tests often have narrow contributors and shouldn't be first-class risks.
	if strings.ToLower(mode) == "risk" {
		if strings.Contains(m.Path, "_test") || strings.HasSuffix(m.Path, "_test.go") {
			score = score * 0.75
		}
	}
	// Save breakdown (scaled to percent contributions) in the metrics for explain mode.
	if m.Breakdown == nil {
		m.Breakdown = make(map[string]float64)
	}
	for k, v := range breakdown {
		m.Breakdown[k] = v * 100.0
	}
	// Store the adjusted score back to the metric (note: caller still assigns the return value)
	// Final clamp to [0,100]
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
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
	// Define columns and initial header names
	headers := []string{"Rank", "File", "Score", "Label", "Contrib", "Commits", "Size(KB)", "Age(d)", "Churn", "Gini", "First Commit"}

	// Compute column widths from headers and data
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	// Helper to consider a printable width for file path (truncate long paths)
	const maxPathWidth = 40

	for idx, f := range files {
		// Rank width
		rankStr := getRank(idx + 1)
		if len(rankStr) > widths[0] {
			widths[0] = len(rankStr)
		}
		// File path width
		p := f.Path
		if len([]rune(p)) > maxPathWidth {
			p = "..." + string([]rune(p)[len([]rune(p))-maxPathWidth+3:])
		}
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
		// Also check First Commit column width (last header)
		if len(nums[len(nums)-1]) > widths[len(widths)-1] {
			widths[len(widths)-1] = len(nums[len(nums)-1])
		}
	}

	// Build format string dynamically
	// Example: %-4s  %-40s  %6s  %-10s  %8s %8s %9s %7s %7s %6s  %s
	fmts := []string{
		// Right-align the Rank column for prettier numeric alignment in the table
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

	// If CSV output requested, skip printing the human-readable table header
	// and separator; write CSV and return.
	if outFmt == "csv" {
		var file *os.File
		var err error
		if cfg.CSVFile != "" {
			file, err = os.Create(cfg.CSVFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: cannot open csv file %s for writing: %v; falling back to stdout\n", cfg.CSVFile, err)
				file = os.Stdout
			}
		} else {
			file = os.Stdout
		}

		w := csv.NewWriter(file)
		// CSV header
		w.Write([]string{"rank", "file", "score", "label", "contributors", "commits", "size_kb", "age_days", "churn", "gini", "first_commit"})
		for i, f := range files {
			p := f.Path
			if len([]rune(p)) > maxPathWidth {
				p = "..." + string([]rune(p)[len([]rune(p))-maxPathWidth+3:])
			}
			rec := []string{
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
			w.Write(rec)
		}
		w.Flush()
		if file != os.Stdout {
			file.Close()
			fmt.Fprintf(os.Stderr, "wrote CSV to %s\n", cfg.CSVFile)
		}
		return
	}

	// Print human-readable header and separator
	fmt.Println(strings.Join(headerParts, "  "))
	fmt.Println(strings.Join(sepParts, "  "))

	// Print rows
	for i, f := range files {
		p := f.Path
		if len([]rune(p)) > maxPathWidth {
			p = "..." + string([]rune(p)[len([]rune(p))-maxPathWidth+3:])
		}
		rowVals := []interface{}{
			getRank(i + 1),
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
			// Ensure we always pass a string to the string-based format specifiers
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

// getRank returns a formatted rank indicator for a given position.
// getRank returns a simple integer string for the rank (e.g., "1", "2").
// This is used in both human-readable and CSV outputs to keep the rank
// column compact and machine-friendly.
func getRank(rank int) string {
	return strconv.Itoa(rank)
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
