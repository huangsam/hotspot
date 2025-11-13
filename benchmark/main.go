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

	"github.com/huangsam/hotspot/internal"
)

// BenchmarkResult holds the result of a benchmark run (cold run and average of warm runs).
type BenchmarkResult struct {
	Repository string
	Command    string
	ColdTime   string
	WarmTime   string
}

// BenchmarkConfig holds configuration for the benchmark run.
type BenchmarkConfig struct {
	RepoBase  string
	Timeout   time.Duration
	Workers   int
	NumRuns   int
	TestRepos []string
	RepoPaths map[string]string
	RepoRefs  map[string][2]string
}

func main() {
	// Parse command line arguments
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s [repo-base-dir]\n", os.Args[0])
		os.Exit(1)
	}
	repoBase := os.Args[1]

	config := BenchmarkConfig{
		RepoBase:  repoBase,
		Timeout:   5 * time.Minute,
		Workers:   14,
		NumRuns:   5,
		TestRepos: []string{"csv-parser", "fd", "git", "kubernetes"},
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

	// Remove the old database file if it exists
	dbPath := internal.GetDBFilePath()
	_ = os.Remove(dbPath)

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

	fmt.Printf("Starting benchmark: %d repos, %v timeout, %d workers, %d runs per test\n",
		len(config.TestRepos), config.Timeout, config.Workers, config.NumRuns)

	for _, repo := range config.TestRepos {
		fmt.Printf("Benchmarking %s\n", repo)

		repoPath := filepath.Join(config.RepoBase, repo)

		// Files analysis
		result := runBenchmark(config, repo, repoPath, "files", "files analysis", "")
		results = append(results, result)

		// Compare analysis
		refs, hasRefs := config.RepoRefs[repo]
		if hasRefs {
			args := fmt.Sprintf("--base-ref %s --target-ref %s", refs[0], refs[1])
			desc := fmt.Sprintf("compare analysis (%s -> %s)", refs[0], refs[1])
			result = runBenchmark(config, repo, repoPath, "compare", desc, "files "+args)
			results = append(results, result)
		}

		// Timeseries analysis
		path, hasPath := config.RepoPaths[repo]
		if hasPath {
			args := fmt.Sprintf("--path %s --interval \"6 months\" --points 4", path)
			desc := fmt.Sprintf("timeseries analysis (%s)", path)
			result = runBenchmark(config, repo, repoPath, "timeseries", desc, args)
			results = append(results, result)
		}
	}

	return results
}

// runBenchmark executes a hotspot command multiple times and returns the cold run time and average warm run time
func runBenchmark(config BenchmarkConfig, repo, repoPath, command, description, extraArgs string) BenchmarkResult {
	fmt.Printf("Running %s on %s (%d runs)\n", description, repo, config.NumRuns)

	var coldTime float64
	var warmTimes []float64
	coldSet := false

	for run := 1; run <= config.NumRuns; run++ {
		fmt.Printf("  Run %d/%d...\n", run, config.NumRuns)

		start := time.Now()

		// Prepare command - properly parse arguments
		args := []string{command}
		if extraArgs != "" {
			// Simple argument parsing - split on spaces but preserve quoted strings
			args = append(args, parseArgs(extraArgs)...)
		}

		cmd := exec.Command("hotspot", args...)
		cmd.Dir = repoPath

		// Run with timeout
		done := make(chan bool, 1)
		var output []byte
		var cmdErr error

		go func() {
			output, cmdErr = cmd.CombinedOutput()
			done <- true
		}()

		select {
		case <-done:
			elapsed := time.Since(start)

			// Check if command succeeded by examining output
			if cmdErr == nil && isSuccess(output, command) {
				if !coldSet {
					coldTime = elapsed.Seconds()
					coldSet = true
					fmt.Printf("    Cold run completed in %.3fs\n", elapsed.Seconds())
				} else {
					warmTimes = append(warmTimes, elapsed.Seconds())
					fmt.Printf("    Warm run completed in %.3fs\n", elapsed.Seconds())
				}
			} else {
				fmt.Printf("    Failed\n")
				if cmdErr != nil {
					fmt.Printf("    Error: %v\n", cmdErr)
				}
			}

		case <-time.After(config.Timeout):
			fmt.Printf("    Timed out after %v\n", config.Timeout)
		}
	}

	// Compute cold and warm times
	coldTimeStr := "TIMEOUT"
	if coldSet {
		coldTimeStr = fmt.Sprintf("%.3fs", coldTime)
	}
	warmAvgStr := computeAverageTime(warmTimes)
	fmt.Printf("  Cold time: %s, Warm average: %s\n", coldTimeStr, warmAvgStr)

	return BenchmarkResult{
		Repository: repo,
		Command:    command,
		ColdTime:   coldTimeStr,
		WarmTime:   warmAvgStr,
	}
}

// parseArgs splits command arguments while preserving quoted strings
func parseArgs(argsStr string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for i, r := range argsStr {
		switch r {
		case '"':
			if inQuotes {
				// End of quoted string
				inQuotes = false
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				// Start of quoted string
				inQuotes = true
			}
		case ' ':
			if inQuotes {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				// End of unquoted argument
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}

		// Handle end of string
		if i == len(argsStr)-1 && current.Len() > 0 {
			args = append(args, current.String())
		}
	}

	return args
}

// isSuccess checks if command output indicates successful completion
func isSuccess(output []byte, command string) bool {
	outputStr := string(output)

	switch command {
	case "files", "compare":
		return strings.Contains(outputStr, "Analysis completed in") &&
			strings.Contains(outputStr, "using") &&
			strings.Contains(outputStr, "workers")
	case "timeseries":
		return strings.Contains(outputStr, "Timeseries analysis completed in") &&
			strings.Contains(outputStr, "using") &&
			strings.Contains(outputStr, "workers")
	default:
		return len(outputStr) > 0
	}
}

// computeAverageTime calculates the average time from multiple runs
func computeAverageTime(times []float64) string {
	if len(times) == 0 {
		return "TIMEOUT"
	}

	var sum float64
	for _, t := range times {
		sum += t
	}
	avg := sum / float64(len(times))
	return fmt.Sprintf("%.3fs", avg)
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
	if err := writer.Write([]string{"repo", "cmd", "cold_time", "warm_avg"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write results
	for _, result := range results {
		if err := writer.Write([]string{result.Repository, result.Command, result.ColdTime, result.WarmTime}); err != nil {
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
			fmt.Printf("  %-12s: Cold: %s, Warm: %s\n", result.Repository, result.ColdTime, result.WarmTime)
		}
	}
}
