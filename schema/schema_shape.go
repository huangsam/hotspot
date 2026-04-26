package schema

import "time"

// PresetName identifies a named configuration preset.
type PresetName string

// Named configuration presets.
const (
	PresetSmall PresetName = "small"
	PresetLarge PresetName = "large"
	PresetInfra PresetName = "infra"
)

// Preset is a named collection of recommended configuration defaults derived from
// the example configuration files in examples/cli/.
type Preset struct {
	Name                 PresetName  `json:"name"`
	Description          string      `json:"description"`
	Mode                 ScoringMode `json:"mode"`
	Limit                int         `json:"limit"`
	Workers              int         `json:"workers"`
	Follow               bool        `json:"follow"`
	Detail               bool        `json:"detail"`
	Start                string      `json:"start,omitempty"` // relative time string, e.g. "2 years ago"
	Transitions          int         `json:"transitions"`     // Suggested value for get_release_journey
	RecencyThresholdLow  float64     `json:"recency_threshold_low"`
	RecencyThresholdHigh float64     `json:"recency_threshold_high"`
	Exclude              string      `json:"exclude,omitempty"`
	Output               OutputMode  `json:"output,omitempty"`
	Color                bool        `json:"color,omitempty"`
	Owner                bool        `json:"owner,omitempty"`
	Precision            int         `json:"precision,omitempty"`
	Explain              bool        `json:"explain,omitempty"`
}

// RepoShape captures key metrics from the first aggregation pass to characterize a repository.
type RepoShape struct {
	URN                string     `json:"urn,omitempty"`
	FileCount          int        `json:"file_count"`
	TotalCommits       float64    `json:"total_commits"`
	UniqueContributors int        `json:"unique_contributors"`
	AvgChurnPerFile    float64    `json:"avg_churn_per_file"`
	IaCFileRatio       float64    `json:"iac_file_ratio"`
	RecommendedPreset  PresetName `json:"recommended_preset"`
	Reasoning          []string   `json:"reasoning,omitempty"`
	Preset             Preset     `json:"preset"`
	AnalyzedAt         time.Time  `json:"analyzed_at"`
}
