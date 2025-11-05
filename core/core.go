// Package core has core logic for analysis, scoring and ranking.
package core

import "github.com/huangsam/hotspot/internal"

// ExecuteHotspotFiles runs the file-level analysis and prints results to stdout.
// It serves as the main entry point for the 'files' mode.
func ExecuteHotspotFiles(cfg *internal.Config) {
	ranked, err := AnalyzeFiles(cfg)
	if err != nil || len(ranked) == 0 {
		return
	}
	internal.PrintFileResults(ranked, cfg)
}

// ExecuteHotspotFolders runs the folder-level analysis and prints results to stdout.
// It serves as the main entry point for the 'folders' mode.
func ExecuteHotspotFolders(cfg *internal.Config) {
	ranked, err := AnalyzeFolders(cfg)
	if err != nil || len(ranked) == 0 {
		return
	}
	internal.PrintFolderResults(ranked, cfg)
}
