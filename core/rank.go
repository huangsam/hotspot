package core

import (
	"sort"

	"github.com/huangsam/hotspot/schema"
)

// RankFiles sorts files by their importance score in descending order
// and returns the top 'limit' files. If limit is greater than the number
// of files, all files are returned in sorted order.
func RankFiles(files []schema.FileMetrics, limit int) []schema.FileMetrics {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Score > files[j].Score
	})
	if len(files) > limit {
		return files[:limit]
	}
	return files
}
