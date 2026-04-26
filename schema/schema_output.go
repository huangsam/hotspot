package schema

import "time"

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
		return CriticalValue
	case score >= 60:
		return HighValue
	case score >= 40:
		return ModerateValue
	default:
		return LowValue
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

// Metadata contains runtime information about the analysis.
type Metadata struct {
	AnalysisDuration time.Duration `json:"analysis_duration_ns"`
	AnalysisTime     string        `json:"analysis_duration"`
	Workers          int           `json:"workers"`
	CacheBackend     string        `json:"cache_backend"`
	Timestamp        time.Time     `json:"timestamp"`
}

// FileResultsOutput is the standard container for file analysis results.
type FileResultsOutput struct {
	Results  []EnrichedFileResult `json:"results"`
	Metadata Metadata             `json:"metadata"`
}

// FolderResultsOutput is the standard container for folder analysis results.
type FolderResultsOutput struct {
	Results  []EnrichedFolderResult `json:"results"`
	Metadata Metadata               `json:"metadata"`
}

// ComparisonResultsOutput is the standard container for comparison analysis results.
type ComparisonResultsOutput struct {
	Results  ComparisonResult `json:"results"`
	Metadata Metadata         `json:"metadata"`
}

// RepoShapeOutput is the standard container for repository shape analysis results.
type RepoShapeOutput struct {
	Results  RepoShape `json:"results"`
	Metadata Metadata  `json:"metadata"`
}

// TimeseriesResultsOutput is the standard container for timeseries analysis results.
type TimeseriesResultsOutput struct {
	Results  TimeseriesResult `json:"results"`
	Metadata Metadata         `json:"metadata"`
}

// BlastRadiusResultsOutput is the standard container for blast radius analysis results.
type BlastRadiusResultsOutput struct {
	Results  BlastRadiusResult `json:"results"`
	Metadata Metadata          `json:"metadata"`
}

// JourneyResultsOutput is the standard container for release journey analysis results.
type JourneyResultsOutput struct {
	Results  JourneyResult `json:"results"`
	Metadata Metadata      `json:"metadata"`
}

// CheckResultsOutput is the standard container for policy check analysis results.
type CheckResultsOutput struct {
	Results  *CheckResult `json:"results"`
	Metadata Metadata     `json:"metadata"`
}

// BatchAnalysisResultsOutput is the standard container for fleet-wide batch analysis results.
type BatchAnalysisResultsOutput struct {
	Results  []RepoShape `json:"results"`
	Metadata Metadata    `json:"metadata"`
}

// RuntimeSettings matches the minimal interface needed for metadata building.
type RuntimeSettings interface {
	GetWorkers() int
	GetCacheBackend() DatabaseBackend
}

// BuildMetadata constructs a standard metadata object from runtime and duration.
func BuildMetadata(runtime RuntimeSettings, duration time.Duration) Metadata {
	return Metadata{
		AnalysisDuration: duration,
		AnalysisTime:     duration.String(),
		Workers:          runtime.GetWorkers(),
		CacheBackend:     string(runtime.GetCacheBackend()),
		Timestamp:        time.Now().UTC(),
	}
}
