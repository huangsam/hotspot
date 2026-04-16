package schema

// CoupledFile represents a file and its coupling strength to a target.
type CoupledFile struct {
	Path     string  `json:"path"`      // Path of the coupled file
	Score    float64 `json:"score"`     // Coupling score (0.0 to 1.0)
	How      string  `json:"how"`       // Human-readable reason (e.g., "Changed together 15 times")
	CoChange int     `json:"co_change"` // Number of times these files changed together
}

// BlastRadiusPair represents a pair of files that are logically coupled.
type BlastRadiusPair struct {
	Source   string  `json:"source"`    // The "source" file or the center of the radius
	Target   string  `json:"target"`    // The coupled file
	Score    float64 `json:"score"`     // Coupling score (Jaccard Index)
	CoChange int     `json:"co_change"` // Number of times they changed together
}

// BlastRadiusSummary provides metadata about the analysis.
type BlastRadiusSummary struct {
	TotalCommits int     `json:"total_commits"` // Total commits analyzed
	TotalPairs   int     `json:"total_pairs"`   // Number of highly coupled pairs found
	Threshold    float64 `json:"threshold"`     // Min coupling score used for filtering
}

// BlastRadiusResult is the top-level response for the get_blast_radius tool.
type BlastRadiusResult struct {
	Summary BlastRadiusSummary `json:"summary"`
	Pairs   []BlastRadiusPair  `json:"pairs"`
}
