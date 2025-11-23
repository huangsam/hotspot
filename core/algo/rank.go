package algo

import (
	"sort"

	"github.com/huangsam/hotspot/schema"
)

// RankFiles sorts files by their importance score in descending order
// and returns the top 'limit' files. If limit is greater than the number
// of files, all files are returned in sorted order.
func RankFiles(files []schema.FileResult, limit int) []schema.FileResult {
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModeScore > files[j].ModeScore
	})
	if len(files) > limit {
		return files[:limit]
	}
	return files
}

// RankFolders sorts folders by their importance score in descending order
// and returns the top 'limit' files. If limit is greater than the number
// of files, all files are returned in sorted order.
func RankFolders(folders []schema.FolderResult, limit int) []schema.FolderResult {
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].Score > folders[j].Score
	})
	if len(folders) > limit {
		return folders[:limit]
	}
	return folders
}
