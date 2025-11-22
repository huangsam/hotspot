package iocache

import (
	"fmt"

	"github.com/huangsam/hotspot/schema"
)

// PrintCacheStatus prints cache status information.
func PrintCacheStatus(status schema.CacheStatus) {
	fmt.Printf("Cache Backend: %s\n", status.Backend)
	fmt.Printf("Connected: %t\n", status.Connected)
	if !status.Connected {
		return
	}
	fmt.Printf("Total Entries: %d\n", status.TotalEntries)
	if status.TotalEntries > 0 {
		fmt.Printf("Last Entry: %s\n", status.LastEntryTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Oldest Entry: %s\n", status.OldestEntryTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("Table Size: %d bytes\n", status.TableSizeBytes)
}

// PrintAnalysisStatus prints analysis status information.
func PrintAnalysisStatus(status schema.AnalysisStatus) {
	fmt.Printf("Analysis Backend: %s\n", status.Backend)
	fmt.Printf("Connected: %t\n", status.Connected)
	if !status.Connected {
		return
	}
	fmt.Printf("Total Runs: %d\n", status.TotalRuns)
	if status.TotalRuns > 0 {
		fmt.Printf("Last Run ID: %d\n", status.LastRunID)
		fmt.Printf("Last Run: %s\n", status.LastRunTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Oldest Run: %s\n", status.OldestRunTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Total Files Analyzed: %d\n", status.TotalFilesAnalyzed)
	}
	fmt.Println("Table Sizes:")
	for table, size := range status.TableSizes {
		fmt.Printf("  %s: %d rows\n", table, size)
	}
}
