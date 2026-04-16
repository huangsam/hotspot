package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/huangsam/hotspot/core"
	clipresets "github.com/huangsam/hotspot/examples/cli"
	"github.com/huangsam/hotspot/internal/logger"
	"github.com/huangsam/hotspot/schema"
	"github.com/spf13/cobra"
)

var (
	initPreset string
	initGlobal bool
	initStyle  string
	initForce  bool
)

const (
	styleMinimal = "minimal"
	styleFull    = "full"
)

// initCmd initializes the repository with a hotspot configuration file.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize hotspot configuration for the repository or globally.",
	Long: `Creates a .hotspot.yml configuration file to customize hotspot behavior.

If no preset is provided, shape analysis is run to recommend the best fit for
your repository. Use --global to save the configuration in your home directory,
which acts as a fallback for all repositories.

Styles:
  minimal: Only includes the 'preset' name (cleanest, uses built-in defaults).
  full:    Copies all recommended settings from the preset into the file.`,
	Args:    cobra.NoArgs,
	PreRunE: sharedSetupWrapper,
	Run: func(_ *cobra.Command, _ []string) {
		var presetName schema.PresetName
		if initPreset != "" {
			presetName = schema.PresetName(initPreset)
		} else {
			logger.Info("No preset provided, analyzing repository shape...")
			shape, _, err := core.GetHotspotShapeResults(rootCtx, cfg, gitClient, cacheManager)
			if err != nil {
				logger.Fatal("Failed to analyze repository shape", err)
			}
			presetName = shape.RecommendedPreset
			logger.Info(fmt.Sprintf("Recommended preset: %s", presetName))
		}

		var savePath string
		if initGlobal {
			home, err := os.UserHomeDir()
			if err != nil {
				logger.Fatal("Failed to get home directory", err)
			}
			savePath = filepath.Join(home, ".hotspot.yml")
		} else {
			savePath = filepath.Join(cfg.Git.RepoPath, ".hotspot.yml")
		}

		if _, err := os.Stat(savePath); err == nil && !initForce {
			logger.Fatal(fmt.Sprintf("Configuration file already exists at %s. Use --force to overwrite.", savePath), nil)
		}

		var data []byte
		if initStyle == styleFull {
			data = clipresets.PresetYAML(presetName)
		} else {
			data = fmt.Appendf(nil, "preset: %s\n", presetName)
		}

		if err := os.WriteFile(savePath, data, 0o644); err != nil {
			logger.Fatal(fmt.Sprintf("Failed to write configuration to %s", savePath), err)
		}

		logger.Info(fmt.Sprintf("Successfully initialized %s configuration at %s", presetName, savePath))
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initPreset, "preset", "", "Specify a preset (small, large, infra)")
	initCmd.Flags().BoolVar(&initGlobal, "global", false, "Save configuration in home directory")
	initCmd.Flags().StringVar(&initStyle, "style", styleMinimal, "Configuration style: minimal (default) or full")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing configuration file")
}
