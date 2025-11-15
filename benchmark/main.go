// Package main provides a comprehensive performance benchmarking tool for the Hotspot CLI.
// It measures execution times across different repository sizes and command types,
// running each test multiple times, treating the first successful run as cold and averaging the rest as warm,
// generating CSV output for performance analysis and documentation.
//
// Prerequisites:
// - hotspot binary installed and available in PATH
// - Test repositories cloned to the specified base directory
// - Git repositories: csv-parser, fd, git, kubernetes
//
// Usage: go run benchmark/main.go [repo-base-dir]
//
//	repo-base-dir: Directory containing test repositories
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BenchmarkResult holds the result of a benchmark run (no-cache average, cold run and average of warm runs).
type BenchmarkResult struct {
	Repository  string
	Command     string
	NoCacheTime string
	ColdTime    string
	WarmTime    string
}

// BenchmarkConfig holds configuration for the benchmark run.
type BenchmarkConfig struct {
	RepoBase    string
	Timeout     time.Duration
	Workers     int
	NoCacheRuns int
	CacheRuns   int
	TestRepos   []string
	RepoPaths   map[string]string
	RepoRefs    map[string][2]string
}

func main() {
	// Parse command line arguments
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s [repo-base-dir]\n", os.Args[0])
		os.Exit(1)
	}
	repoBase := os.Args[1]

	config := BenchmarkConfig{
		RepoBase:    repoBase,
		Timeout:     5 * time.Minute,
		Workers:     14,
		NoCacheRuns: 3,
		CacheRuns:   4,
		TestRepos:   []string{"csv-parser", "fd", "git", "kubernetes"},
		RepoPaths: map[string]string{
			"csv-parser": "python/csvpy.cpp",
			"fd":         "src/main.rs",
			"git":        "builtin/add.c",
			"kubernetes": "cmd/cloud-controller-manager/main.go",
		},
		RepoRefs: map[string][2]string{
			"csv-parser": {"v1.0.0", "v1.1.0"},
			"fd":         {"v9.0.0", "v10.0.0"},
			"git":        {"v2.51.0", "v2.52.0-rc0"},
			"kubernetes": {"v1.34.0", "v1.35.0-alpha.0"},
		},
	}

	if err := checkPrerequisites(config); err != nil {
		fmt.Printf("Prerequisites check failed: %v\n", err)
		os.Exit(1)
	}

	// Clear the cache using hotspot cache clear
	fmt.Printf("Clearing cache...\n")
	clearCmd := exec.Command("hotspot", "cache", "clear")
	if output, err := clearCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to clear cache: %v\nOutput: %s\n", err, string(output))
	} else {
		fmt.Printf("Cache cleared successfully\n")
	}

	results := runBenchmarks(config)

	if err := saveResults(results); err != nil {
		fmt.Printf("Failed to save results: %v\n", err)
		os.Exit(1)
	}

	printSummary(results)
}

// checkPrerequisites verifies that hotspot binary and test repositories exist
func checkPrerequisites(config BenchmarkConfig) error {
	// Check if hotspot is available
	if _, err := exec.LookPath("hotspot"); err != nil {
		return fmt.Errorf("hotspot binary not found in PATH")
	}

	// Check if repositories exist
	for _, repo := range config.TestRepos {
		repoPath := filepath.Join(config.RepoBase, repo)
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			return fmt.Errorf("repository %s not found at %s", repo, repoPath)
		}
	}

	return nil
}

// runBenchmarks executes all benchmark tests across configured repositories
func runBenchmarks(config BenchmarkConfig) []BenchmarkResult {
	var results []BenchmarkResult

	fmt.Printf("Starting benchmark: %d repos, %v timeout, %d workers, no-cache: %d runs, cache: %d runs\n",
		len(config.TestRepos), config.Timeout, config.Workers, config.NoCacheRuns, config.CacheRuns)

	for _, repo := range config.TestRepos {
		fmt.Printf("Benchmarking %s\n", repo)

		repoPath := filepath.Join(config.RepoBase, repo)

		// Files analysis
		result := runBenchmarkSuite(config, repo, repoPath, "files", "files analysis", "")
		results = append(results, result)

		// Compare analysis
		refs, hasRefs := config.RepoRefs[repo]
		if hasRefs {
			args := fmt.Sprintf("--base-ref %s --target-ref %s", refs[0], refs[1])
			desc := fmt.Sprintf("compare analysis (%s -> %s)", refs[0], refs[1])
			result = runBenchmarkSuite(config, repo, repoPath, "compare", desc, "files "+args)
			results = append(results, result)
		}

		// Timeseries analysis
		path, hasPath := config.RepoPaths[repo]
		if hasPath {
			args := fmt.Sprintf("--path %s --interval \"6 months\" --points 4", path)
			desc := fmt.Sprintf("timeseries analysis (%s)", path)
			result = runBenchmarkSuite(config, repo, repoPath, "timeseries", desc, args)
			results = append(results, result)
		}
	}

	return results
}

// runBenchmarkSuite runs both no-cache and cache benchmarks for a command
func runBenchmarkSuite(config BenchmarkConfig, repo, repoPath, command, description, extraArgs string) BenchmarkResult {
	fmt.Printf("Running %s on %s\n", description, repo)

	// Helper to run a benchmark phase
	runPhase := func(cacheBackend string, numRuns int, phaseName string) (coldTime float64, avgTime string) {
		fmt.Printf("  %s phase (%d runs)\n", phaseName, numRuns)
		cold, times := runBenchmark(config, repoPath, command, extraArgs, cacheBackend, numRuns)
		if len(times) == 0 {
			avgTime = "TIMEOUT"
		} else {
			var sum float64
			for _, t := range times {
				sum += t
			}
			avg := sum / float64(len(times))
			avgTime = fmt.Sprintf("%.3fs", avg)
		}
		return cold, avgTime
	}

	// Phase 1: No-cache runs
	_, noCacheAvg := runPhase("none", config.NoCacheRuns, "No-cache")

	// Phase 2: Cache runs
	coldTime, warmAvg := runPhase("sqlite", config.CacheRuns, "Cache")

	coldTimeStr := "TIMEOUT"
	if coldTime > 0 {
		coldTimeStr = fmt.Sprintf("%.3fs", coldTime)
	}

	fmt.Printf("  No-cache average: %s, Cold time: %s, Warm average: %s\n", noCacheAvg, coldTimeStr, warmAvg)

	return BenchmarkResult{
		Repository:  repo,
		Command:     command,
		NoCacheTime: noCacheAvg,
		ColdTime:    coldTimeStr,
		WarmTime:    warmAvg,
	}
}

// runBenchmark executes a hotspot command multiple times with specified cache backend and returns cold time and warm times
func runBenchmark(config BenchmarkConfig, repoPath, command, extraArgs, cacheBackend string, numRuns int) (coldTime float64, warmTimes []float64) {
	// Prepare command arguments
	args := []string{command, "--cache-backend", cacheBackend}
	if extraArgs != "" {
		args = append(args, parseArgs(extraArgs)...)
	}

	var times []float64
	for run := 1; run <= numRuns; run++ {
		start := time.Now()

		cmd := exec.Command("hotspot", args...)
		cmd.Dir = repoPath

		done := make(chan bool)
		var output []byte
		var cmdErr error

		go func() {
			output, cmdErr = cmd.CombinedOutput()
			done <- true
		}()

		select {
		case <-done:
			if cmdErr == nil && isSuccess(output, command) {
				times = append(times, time.Since(start).Seconds())
			}
		case <-time.After(config.Timeout):
			// Timeout - don't add to times
		}
	}

	if len(times) > 0 {
		coldTime = times[0]
		warmTimes = times[1:]
	}
	return
}

func parseArgs(argsStr string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for _, r := range argsStr {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes && current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			} else if inQuotes {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// isSuccess checks if command output indicates successful completion
func isSuccess(output []byte, command string) bool {
	outputStr := string(output)

	var completionPhrase string
	if command == "timeseries" {
		completionPhrase = "Timeseries analysis completed in"
	} else {
		completionPhrase = "Analysis completed in"
	}

	return strings.Contains(outputStr, completionPhrase) &&
		strings.Contains(outputStr, "using") &&
		strings.Contains(outputStr, "workers")
}

// saveResults writes benchmark results to a timestamped CSV file
func saveResults(results []BenchmarkResult) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("/tmp/hotspot_benchmark_%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", filename, closeErr)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"repo", "cmd", "no_cache_avg", "cold_time", "warm_avg"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write results
	for _, result := range results {
		if err := writer.Write([]string{result.Repository, result.Command, result.NoCacheTime, result.ColdTime, result.WarmTime}); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	fmt.Printf("Results saved to %s\n", filename)
	return nil
}

// printSummary displays the final benchmark results summary
func printSummary(results []BenchmarkResult) {
	fmt.Printf("Benchmark complete\n")

	printCommandSummary(results, "files", "Files Analysis:")
	printCommandSummary(results, "compare", "Compare Analysis:")
	printCommandSummary(results, "timeseries", "Timeseries Analysis:")

	fmt.Printf("Benchmark script completed successfully\n")
}

// printCommandSummary displays results for a specific command type
func printCommandSummary(results []BenchmarkResult, command, title string) {
	fmt.Printf("%s\n", title)
	for _, result := range results {
		if result.Command == command {
			fmt.Printf("  %-12s: No-cache: %s, Cold: %s, Warm: %s\n", result.Repository, result.NoCacheTime, result.ColdTime, result.WarmTime)
		}
	}
}
