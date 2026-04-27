package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/huangsam/hotspot/core"
	"github.com/huangsam/hotspot/internal/config"
	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/internal/outwriter"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// batchCmd performs silent analysis on multiple repositories.
var batchCmd = &cobra.Command{
	Use:   "batch [paths...]",
	Short: "Perform silent analysis on a list of Git repositories.",
	Long: `Batch analyze multiple Git repositories to populate the analysis database.

This command is designed for "fleet-wide" cache warming. It:
1. Accepts a list of repository paths OR discovers them automatically with --auto.
2. Performs a silent analysis on each repository.
3. Records file scores for all 4 modes (Hot, Risk, Complexity, ROI).

Results are stored in your configured analysis database, making them
instantly available for the Intelligence Cockpit or MCP queries.

Examples:
  # Analyze specific repositories
  hotspot batch ./project-a ./project-b

  # Automatically find and analyze all repos in a directory tree
  hotspot batch --auto /path/to/many/repos

  # Find and analyze all repos in the current directory
  hotspot batch --auto .`,
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, args []string) {
		autoDiscovery := viper.GetBool("auto")
		var rawRepos []string

		if autoDiscovery {
			searchDirs := args
			if len(searchDirs) == 0 {
				searchDirs = []string{"."}
			}
			for _, dir := range searchDirs {
				absSearchDir, err := filepath.Abs(dir)
				if err != nil {
					logger.Warn(fmt.Sprintf("Invalid search directory: %s", dir), err)
					continue
				}
				logger.Info(fmt.Sprintf("Searching for Git repositories in %s...", absSearchDir))
				rawRepos = append(rawRepos, discoverRepos(absSearchDir)...)
			}
		} else {
			if len(args) == 0 {
				logger.Fatal("No repository paths provided. Use --auto for recursive discovery.", nil)
			}
			for _, arg := range args {
				absPath, err := filepath.Abs(arg)
				if err != nil {
					logger.Warn(fmt.Sprintf("Invalid path: %s", arg), err)
					continue
				}
				rawRepos = append(rawRepos, absPath)
			}
		}

		// Deduplicate repositories by absolute path
		seen := make(map[string]bool)
		var repos []string
		for _, repo := range rawRepos {
			if !seen[repo] {
				seen[repo] = true
				repos = append(repos, repo)
			}
		}

		if len(repos) == 0 {
			logger.Warn("No repositories found for analysis.", nil)
			return
		}

		if autoDiscovery {
			logger.Info(fmt.Sprintf("Found %d unique repositories.", len(repos)))
		}

		// Process repositories sequentially
		start := time.Now()
		shapes, errs := processRepos(repos)
		duration := time.Since(start)

		// Report human-facing summary to stderr
		if !cfg.Output.Quiet {
			if len(errs) > 0 {
				errorGroups := make(map[string][]string)
				for _, err := range errs {
					msg := err.Error()
					if strings.Contains(msg, "no files found for analysis") {
						errorGroups["no files found"] = append(errorGroups["no files found"], msg)
					} else {
						errorGroups["other errors"] = append(errorGroups["other errors"], msg)
					}
				}

				fmt.Fprintf(os.Stderr, "Batch analysis finished with %d failures:\n", len(errs))
				for group, msgs := range errorGroups {
					if group == "no files found" {
						fmt.Fprintf(os.Stderr, "- %d repositories had no files found (try adjusting excludes or search path)\n", len(msgs))
					} else {
						for i, msg := range msgs {
							if i >= 3 {
								fmt.Fprintf(os.Stderr, "- ... and %d more unique errors\n", len(msgs)-3)
								break
							}
							fmt.Fprintf(os.Stderr, "- %s\n", msg)
						}
					}
				}
			} else {
				fmt.Fprintln(os.Stderr, "Batch analysis complete.")
			}
		}

		// Final report via unified outwriter

		// Final report via unified outwriter
		if err := outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
			return resultWriter.WriteBatch(w, shapes, cfg.Output, cfg.Runtime, duration)
		}, "Wrote batch summary"); err != nil {
			logger.Fatal("Failed to write batch summary", err)
		}
	},
}

func discoverRepos(root string) []string {
	repos, err := git.DiscoverRepositories(root)
	if err != nil {
		logger.Warn("Error during repository discovery", err)
	}
	return repos
}

func processRepos(repos []string) ([]schema.RepoShape, []error) {
	var shapes []schema.RepoShape
	var errors []error
	total := len(repos)
	progress := logger.NewProgress()

	for i, repoPath := range repos {
		progress.Update(i+1, total, fmt.Sprintf("Analyzing %s...", filepath.Base(repoPath)))
		shape, err := analyzeOneRepo(repoPath)
		if err != nil {
			progress.IncrWarn()
			errors = append(errors, fmt.Errorf("%s: %w", repoPath, err))
			continue
		}
		shapes = append(shapes, shape)
	}
	progress.Complete("") // The summary is now handled by the caller

	return shapes, errors
}

func analyzeOneRepo(repoPath string) (schema.RepoShape, error) {
	repoCfg := cfg.Clone()
	repoCfg.Git.RepoPath = repoPath
	repoCfg.Output.Format = schema.NoneOut
	// Clear the URN so each repo resolves its own identity; a user-supplied
	// --urn would otherwise stamp the same URN on every repository in the
	// batch, causing cache-key collisions and cross-repo data contamination.
	repoCfg.Git.RepoURN = ""

	repoInput := *input
	repoInput.RepoPathStr = repoPath
	repoInput.Output = "none"
	repoInput.URN = ""

	if err := config.ResolveGitPathAndFilter(rootCtx, repoCfg, gitClient, &repoInput); err != nil {
		return schema.RepoShape{}, fmt.Errorf("failed to resolve git path: %w", err)
	}

	analysisCtx := core.WithSuppressHeader(rootCtx)
	shape, _, err := core.GetBatchAnalysisResults(analysisCtx, repoCfg, gitClient, cacheManager)
	return shape, err
}

func init() {
	batchCmd.Flags().Bool("auto", false, "Recursively discover repositories in the specified directory")
	_ = viper.BindPFlag("auto", batchCmd.Flags().Lookup("auto"))
}
