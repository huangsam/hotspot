package iocache

import (
	"encoding/json"
	"fmt"

	"github.com/huangsam/hotspot/internal/logger"
)

// BackfillAnalysisURNs scans existing analysis runs and populates the 'urn' column.
// It uses the 'repo_path' from the config_params and establishes a 'local:' URN for legacy runs.
func BackfillAnalysisURNs(store *AnalysisStoreImpl) error {
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

		// For backfill of legacy runs, we use the local path URN.
		urn := "local:" + repoPath

		if err = store.UpdateAnalysisRunURN(run.AnalysisID, urn); err != nil {
			logger.Warn(fmt.Sprintf("Failed to backfill URN for run %d", run.AnalysisID), err)
		}
	}

	return nil
}
