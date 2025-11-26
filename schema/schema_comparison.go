package schema

// ComparisonDetail holds the base info, target info, and their associated deltas.
type ComparisonDetail struct {
	Path         string      `json:"path"`          // Relative path to the target in the repository
	BeforeScore  float64     `json:"before_score"`  // Score from the original/base analysis
	AfterScore   float64     `json:"after_score"`   // Score from the comparison/new analysis
	Delta        float64     `json:"delta"`         // CompScore - BaseScore (Positive means worse/higher)
	DeltaCommits int         `json:"delta_commits"` // Change in total commits (Positive means more activity)
	DeltaChurn   int         `json:"delta_churn"`   // Change in total churn (Positive means more volatility)
	Status       Status      `json:"status"`        // Intrinsic status of the file as of now
	BeforeOwners []string    `json:"before_owners"` // Owners from the base analysis
	AfterOwners  []string    `json:"after_owners"`  // Owners from the target analysis
	Mode         ScoringMode `json:"mode"`          // Scoring mode used (hot, risk, complexity, stale)

	*FileComparison   `json:"file_compare,omitempty"`
	*FolderComparison `json:"folder_compare,omitempty"`
}

// ComparisonSummary has high-level deltas and counts.
type ComparisonSummary struct {
	// 1. Net Score Delta
	NetScoreDelta float64 `json:"net_score_delta"`

	// 2. Net Churn Delta
	NetChurnDelta int `json:"net_churn_delta"`

	// 3. File Status Counts
	TotalNewFiles      int `json:"total_new_files"`
	TotalInactiveFiles int `json:"total_inactive_files"`
	TotalModifiedFiles int `json:"total_modified_files"`

	// 4. Ownership Changes
	TotalOwnershipChanges int `json:"total_ownership_changes"`
}

// ComparisonResult holds the comparison details and summary.
type ComparisonResult struct {
	Details []ComparisonDetail `json:"details"`
	Summary ComparisonSummary  `json:"summary"`
}

// FileComparison has file deltas.
type FileComparison struct {
	DeltaLOC     int `json:"delta_loc"`     // Change in LOC (Positive means file growth)
	DeltaContrib int `json:"delta_contrib"` // Change in contributors (Positive means contrib growth)
}

// FolderComparison has folder deltas.
type FolderComparison struct{}
