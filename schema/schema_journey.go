package schema

// JourneyStep represents the hotspot delta between two successive releases.
type JourneyStep struct {
	BaseRef   string           `json:"base_ref"`   // Starting tag/ref
	TargetRef string           `json:"target_ref"` // Ending tag/ref
	Result    ComparisonResult `json:"result"`     // Full comparison details and summary
}

// JourneySummary provides an aggregated overview across all steps.
type JourneySummary struct {
	TotalSteps         int     `json:"total_steps"`          // Number of transitions analyzed
	TotalNewFiles      int     `json:"total_new_files"`      // Cumulative new files across all steps
	TotalInactiveFiles int     `json:"total_inactive_files"` // Cumulative retired files
	NetScoreDelta      float64 `json:"net_score_delta"`      // Sum of all step score deltas
	PeakDeltaStep      string  `json:"peak_delta_step"`      // Transition with the largest net score delta
	Mode               string  `json:"mode"`                 // Scoring mode used
}

// JourneyResult is the top-level response for the get_release_journey MCP tool.
type JourneyResult struct {
	Summary JourneySummary `json:"summary"` // High-level overview
	Steps   []JourneyStep  `json:"steps"`   // Per-transition details, newest first
}
