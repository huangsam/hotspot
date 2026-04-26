package iocache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/huangsam/hotspot/internal/git"
	"github.com/huangsam/hotspot/internal/logger"
)

// BackfillAnalysisURNs scans existing analysis runs and populates the 'urn' column.
// It uses the 'repo_path' from the config_params to resolve a stable URN if the path exists.
func BackfillAnalysisURNs(store AnalysisStore, client git.Client) error {
	runs, err := store.GetAllAnalysisRuns()
	if err != nil {
		return fmt.Errorf("failed to fetch analysis runs for backfill: %w", err)
	}

	for _, run := range runs {
		if run.URN != "" {
			continue // Already has a URN
		}

		if run.ConfigParams == nil || *run.ConfigParams == "" {
			continue
		}

		var params map[string]any
		if err := json.Unmarshal([]byte(*run.ConfigParams), &params); err != nil {
			continue
		}

		repoPath, ok := params["repo_path"].(string)
		if !ok || repoPath == "" {
			continue
		}

		// If the repo directory still exists on this machine, try to resolve its actual URN.
		// This upgrades legacy path-based URNs to modern git: or local:hash URNs.
		urn := "local:" + repoPath
		if info, err := os.Stat(repoPath); err == nil && info.IsDir() {
			if resolved := git.ResolveURN(context.Background(), client, repoPath); resolved != "" {
				urn = resolved
			}
		}

		if err = store.UpdateAnalysisRunURN(run.AnalysisID, urn); err != nil {
			logger.Warn(fmt.Sprintf("Failed to backfill URN for run %d", run.AnalysisID), err)
		}
	}

	return nil
}
