package schema

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed data/scoring_config.yaml
var scoringConfigRaw []byte

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

var sCfg scoringConfig

func init() {
	if err := yaml.Unmarshal(scoringConfigRaw, &sCfg); err != nil {
		panic("failed to unmarshal scoring_config.yaml: " + err.Error())
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
