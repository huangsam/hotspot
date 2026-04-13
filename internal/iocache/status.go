package iocache

import (
	"fmt"
	"os"

	"github.com/huangsam/hotspot/schema"
)

// PrintCacheStatus prints cache status information.
func PrintCacheStatus(status schema.CacheStatus) {
	fmt.Fprintf(os.Stderr, "Cache Backend: %s\n", status.Backend)
	fmt.Fprintf(os.Stderr, "Connected: %t\n", status.Connected)
	if !status.Connected {
		return
	}
	fmt.Fprintf(os.Stderr, "Total Entries: %d\n", status.TotalEntries)
	if status.TotalEntries > 0 {
		fmt.Fprintf(os.Stderr, "Last Entry: %s\n", status.LastEntryTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(os.Stderr, "Oldest Entry: %s\n", status.OldestEntryTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(os.Stderr, "Table Size: %d bytes\n", status.TableSizeBytes)
}

// PrintAnalysisStatus prints analysis status information.
func PrintAnalysisStatus(status schema.AnalysisStatus) {
	fmt.Fprintf(os.Stderr, "Analysis Backend: %s\n", status.Backend)
	fmt.Fprintf(os.Stderr, "Connected: %t\n", status.Connected)
	if !status.Connected {
		return
	}
	fmt.Fprintf(os.Stderr, "Total Runs: %d\n", status.TotalRuns)
	if status.TotalRuns > 0 {
		fmt.Fprintf(os.Stderr, "Last Run ID: %d\n", status.LastRunID)
		fmt.Fprintf(os.Stderr, "Last Run: %s\n", status.LastRunTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(os.Stderr, "Oldest Run: %s\n", status.OldestRunTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(os.Stderr, "Total Files Analyzed: %d\n", status.TotalFilesAnalyzed)
	}
	fmt.Fprintln(os.Stderr, "Table Sizes:")
	for table, size := range status.TableSizes {
		fmt.Fprintf(os.Stderr, "  %s: %d rows\n", table, size)
	}
}
