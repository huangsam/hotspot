package schema

// EnrichedFileResult adds presentation data to a FileResult.
type EnrichedFileResult struct {
	Rank  int    `json:"rank"`
	Label string `json:"label"`
	FileResult
}

// EnrichedFolderResult adds presentation data to a FolderResult.
type EnrichedFolderResult struct {
	Rank  int    `json:"rank"`
	Label string `json:"label"`
	FolderResult
}

// GetPlainLabel returns a plain text label indicating the criticality level
// based on the importance score.
func GetPlainLabel(score float64) string {
	switch {
	case score >= 80:
		return "Critical"
	case score >= 60:
		return "High"
	case score >= 40:
		return "Moderate"
	default:
		return "Low"
	}
}

// EnrichFiles adds rank and label to a list of file results.
func EnrichFiles(files []FileResult) []EnrichedFileResult {
	output := make([]EnrichedFileResult, len(files))
	for i, f := range files {
		output[i] = EnrichedFileResult{
			Rank:       i + 1,
			Label:      GetPlainLabel(f.ModeScore),
			FileResult: f,
		}
	}
	return output
}

// EnrichFolders adds rank and label to a list of folder results.
func EnrichFolders(folders []FolderResult) []EnrichedFolderResult {
	output := make([]EnrichedFolderResult, len(folders))
	for i, f := range folders {
		output[i] = EnrichedFolderResult{
			Rank:         i + 1,
			Label:        GetPlainLabel(f.Score),
			FolderResult: f,
		}
	}
	return output
}
