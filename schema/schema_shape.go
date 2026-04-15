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
	Name        PresetName  `json:"name"`
	Description string      `json:"description"`
	Mode        ScoringMode `json:"mode"`
	Limit       int         `json:"limit"`
	Workers     int         `json:"workers"`
	Follow      bool        `json:"follow"`
	Detail      bool        `json:"detail"`
	Start       string      `json:"start,omitempty"` // relative time string, e.g. "2 years ago"
}

// RepoShape captures key metrics from the first aggregation pass to characterize a repository.
type RepoShape struct {
	FileCount          int        `json:"file_count"`
	TotalCommits       float64    `json:"total_commits"`
	UniqueContributors int        `json:"unique_contributors"`
	AvgChurnPerFile    float64    `json:"avg_churn_per_file"`
	IaCFileRatio       float64    `json:"iac_file_ratio"`
	RecommendedPreset  PresetName `json:"recommended_preset"`
	Preset             Preset     `json:"preset"`
	AnalyzedAt         time.Time  `json:"analyzed_at"`
}

// GetPreset returns the Preset definition for a given name.
// Unknown names fall back to PresetSmall.
func GetPreset(name PresetName) Preset {
	switch name {
	case PresetLarge:
		return Preset{
			Name:        PresetLarge,
			Description: "Optimized for large monorepos with many services and deep Git histories.",
			Mode:        ROIMode,
			Limit:       30,
			Workers:     16,
			Follow:      true,
			Detail:      true,
			Start:       "1 year ago",
		}
	case PresetInfra:
		return Preset{
			Name:        PresetInfra,
			Description: "Optimized for infrastructure-as-code repositories (Terraform, Ansible, Helm, etc.).",
			Mode:        RiskMode,
			Limit:       20,
			Workers:     8,
			Follow:      true,
			Detail:      true,
			Start:       "2 years ago",
		}
	default: // PresetSmall
		return Preset{
			Name:        PresetSmall,
			Description: "Optimized for small, focused repositories (CLI tools, microservices, libraries).",
			Mode:        HotMode,
			Limit:       10,
			Workers:     4,
			Follow:      false,
			Detail:      false,
		}
	}
}

// AllPresets returns all defined presets in a stable order.
func AllPresets() []Preset {
	return []Preset{
		GetPreset(PresetSmall),
		GetPreset(PresetLarge),
		GetPreset(PresetInfra),
	}
}
