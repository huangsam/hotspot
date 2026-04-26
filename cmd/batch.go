package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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
  hotspot batch --auto /path/to/many/repos`,
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, args []string) {
		autoDiscovery := viper.GetBool("auto")
		var repos []string

		if autoDiscovery {
			searchDir := "."
			if len(args) > 0 {
				searchDir = args[0]
			}
			absSearchDir, err := filepath.Abs(searchDir)
			if err != nil {
				logger.Fatal("Invalid search directory", err)
			}
			fmt.Fprintf(os.Stderr, "Searching for Git repositories in %s...\n", absSearchDir)
			repos = discoverRepos(absSearchDir)
			fmt.Fprintf(os.Stderr, "Found %d repositories.\n", len(repos))
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
				repos = append(repos, absPath)
			}
		}

		if len(repos) == 0 {
			return
		}

		// Process repositories sequentially
		shapes := processRepos(repos)

		// Final report via unified outwriter
		_ = outwriter.WriteWithOutputFile(cfg.Output, func(w io.Writer) error {
			return resultWriter.WriteBatch(w, shapes, cfg.Output)
		}, "Wrote batch summary")

		fmt.Fprintln(os.Stderr, "\nBatch analysis complete.")
	},
}

func discoverRepos(root string) []string {
	repos, err := git.DiscoverRepositories(root)
	if err != nil {
		logger.Warn("Error during repository discovery", err)
	}
	return repos
}

func processRepos(repos []string) []schema.RepoShape {
	var shapes []schema.RepoShape
	total := len(repos)
	for i, repoPath := range repos {
		fmt.Fprintf(os.Stderr, "\rProgress: [%d/%d] Analyzing %s...                    ", i+1, total, filepath.Base(repoPath))
		shape, err := analyzeOneRepo(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError analyzing %s: %v\n", repoPath, err)
			continue
		}
		shapes = append(shapes, shape)
	}
	fmt.Fprintln(os.Stderr) // Clear the progress line
	return shapes
}

func analyzeOneRepo(repoPath string) (schema.RepoShape, error) {
	repoCfg := cfg.Clone()
	repoCfg.Git.RepoPath = repoPath
	repoCfg.Output.Format = schema.NoneOut

	repoInput := *input
	repoInput.RepoPathStr = repoPath
	repoInput.Output = "none"

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
