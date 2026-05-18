package schema

import (
	_ "embed"
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

//go:embed data/scoring_config.yaml
var scoringConfigRaw []byte

//go:embed data/presets.yaml
var presetsRaw []byte

//go:embed data/composites.yaml
var compositesRaw []byte

type modeConfig struct {
	Name       string             `yaml:"name"`
	Purpose    string             `yaml:"purpose"`
	Factors    []string           `yaml:"factors"`
	FactorKeys []string           `yaml:"factor_keys"`
	Weights    map[string]float64 `yaml:"weights"`
}

type scoringConfig struct {
	Modes map[string]modeConfig `yaml:"modes"`
}

type compositesConfig struct {
	Composites map[ScoringMode]*CompositeConfig `yaml:"composites"`
}

var (
	sCfg scoringConfig
	pCfg map[string]Preset
	cCfg compositesConfig
)

func init() {
	if err := yaml.Unmarshal(scoringConfigRaw, &sCfg); err != nil {
		panic("failed to unmarshal scoring_config.yaml: " + err.Error())
	}
	if err := yaml.Unmarshal(presetsRaw, &pCfg); err != nil {
		panic("failed to unmarshal presets.yaml: " + err.Error())
	}
	if err := yaml.Unmarshal(compositesRaw, &cCfg); err != nil {
		panic("failed to unmarshal composites.yaml: " + err.Error())
	}

	validateCompositeConfigs()
}

func validateCompositeConfigs() {
	for mode, composite := range cCfg.Composites {
		if composite == nil {
			panic(fmt.Sprintf("composite mode %s has nil config", mode))
		}
		if len(composite.BaseModes) < 2 {
			panic(fmt.Sprintf("composite mode %s must define at least two base_modes", mode))
		}

		seen := map[ScoringMode]struct{}{}
		for _, baseMode := range composite.BaseModes {
			if !IsBaseMode(baseMode) {
				panic(fmt.Sprintf("composite mode %s has invalid base mode %s", mode, baseMode))
			}
			if _, ok := seen[baseMode]; ok {
				panic(fmt.Sprintf("composite mode %s contains duplicate base mode %s", mode, baseMode))
			}
			seen[baseMode] = struct{}{}
		}

		if len(composite.BlendWeights) == 0 {
			panic(fmt.Sprintf("composite mode %s must define blend_weights", mode))
		}

		for _, baseMode := range composite.BaseModes {
			weight, ok := composite.BlendWeights[baseMode]
			if !ok {
				panic(fmt.Sprintf("composite mode %s missing blend weight for base mode %s", mode, baseMode))
			}
			if weight <= 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
				panic(fmt.Sprintf("composite mode %s has invalid blend weight %v for base mode %s", mode, weight, baseMode))
			}
		}
	}
}

// GetDefaultWeights returns the default weights for a given scoring mode from the YAML config.
func GetDefaultWeights(mode ScoringMode) map[BreakdownKey]float64 {
	m, ok := sCfg.Modes[string(mode)]
	if !ok {
		// Default to HotMode for backward compatibility.
		m = sCfg.Modes[string(HotMode)]
	}

	weights := make(map[BreakdownKey]float64)
	for k, v := range m.Weights {
		weights[BreakdownKey(k)] = v
	}
	return weights
}

// GetScoringModeMetadata returns the purpose and factors for a mode.
func GetScoringModeMetadata(mode ScoringMode) (purpose string, factors []string, factorKeys []string) {
	m, ok := sCfg.Modes[string(mode)]
	if !ok {
		return "", nil, nil
	}
	return m.Purpose, m.Factors, m.FactorKeys
}

// GetPreset returns the Preset definition for a given name from the YAML config.
func GetPreset(name PresetName) Preset {
	p, ok := pCfg[string(name)]
	if !ok {
		// Fallback to small preset
		return pCfg[string(PresetSmall)]
	}
	return p
}

// AllPresets returns all defined presets in a stable order.
func AllPresets() []Preset {
	return []Preset{
		GetPreset(PresetSmall),
		GetPreset(PresetLarge),
		GetPreset(PresetInfra),
	}
}

// GetCompositeConfig returns the CompositeConfig for a given composite mode.
// Returns nil if the mode is not a composite.
func GetCompositeConfig(mode ScoringMode) *CompositeConfig {
	cfg, ok := cCfg.Composites[mode]
	if !ok {
		return nil
	}
	return cfg
}
