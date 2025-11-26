package schema

import "time"

// CheckResult holds the results of a policy check.
type CheckResult struct {
	Passed        bool
	FailedFiles   []CheckFailedFile
	TotalFiles    int
	CheckedModes  []ScoringMode
	TargetRef     string
	BaseRef       string
	Thresholds    map[ScoringMode]float64
	MaxScores     map[ScoringMode]float64
	MaxScoreFiles map[ScoringMode][]CheckMaxScoreFile
	Lookback      time.Duration
	AvgScores     map[ScoringMode]float64 // Average score per mode
}

// CheckMaxScoreFile represents a file that achieved the maximum score for a mode.
type CheckMaxScoreFile struct {
	Path   string
	Owners []string
}

// CheckFailedFile represents a file that failed the policy check.
type CheckFailedFile struct {
	Path      string
	Mode      ScoringMode
	Score     float64
	Threshold float64
}
