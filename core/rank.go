package core

import (
	"sort"

	"github.com/huangsam/hotspot/schema"
)

// rankFiles sorts files by their importance score in descending order
// and returns the top 'limit' files. If limit is greater than the number
// of files, all files are returned in sorted order.
func rankFiles(files []schema.FileResult, limit int) []schema.FileResult {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Score > files[j].Score
	})
	if len(files) > limit {
		return files[:limit]
	}
	return files
}

// rankFolders sorts folders by their importance score in descending order
// and returns the top 'limit' files. If limit is greater than the number
// of files, all files are returned in sorted order.
func rankFolders(folders []schema.FolderResult, limit int) []schema.FolderResult {
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].Score > folders[j].Score
	})
	if len(folders) > limit {
		return folders[:limit]
	}
	return folders
}
