package schema

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateCompositeConfigs ensures the runtime composite config validation
// rejects malformed composite definitions. Tests temporarily override the
// package-level cCfg and restore it after each subtest.
func TestValidateCompositeConfigs_PanicsOnInvalidConfigs(t *testing.T) {
	orig := cCfg
	defer func() { cCfg = orig }()

	t.Run("missing base_modes", func(t *testing.T) {
		cCfg.Composites = map[ScoringMode]*CompositeConfig{
			"bad_missing": {BaseModes: []ScoringMode{}, BlendWeights: map[ScoringMode]float64{}},
		}
		assert.Panics(t, func() { validateCompositeConfigs() })
	})

	t.Run("duplicate basemodes", func(t *testing.T) {
		cCfg.Composites = map[ScoringMode]*CompositeConfig{
			"bad_dup": {BaseModes: []ScoringMode{HotMode, HotMode}, BlendWeights: map[ScoringMode]float64{HotMode: 1}},
		}
		assert.Panics(t, func() { validateCompositeConfigs() })
	})

	t.Run("non-base mode", func(t *testing.T) {
		cCfg.Composites = map[ScoringMode]*CompositeConfig{
			"bad_nonbase": {BaseModes: []ScoringMode{ScoringMode("unknown"), HotMode}, BlendWeights: map[ScoringMode]float64{ScoringMode("unknown"): 1, HotMode: 1}},
		}
		assert.Panics(t, func() { validateCompositeConfigs() })
	})

	t.Run("missing blend_weights", func(t *testing.T) {
		cCfg.Composites = map[ScoringMode]*CompositeConfig{
			"bad_weights": {BaseModes: []ScoringMode{HotMode, RiskMode}, BlendWeights: map[ScoringMode]float64{}},
		}
		assert.Panics(t, func() { validateCompositeConfigs() })
	})

	t.Run("invalid weight", func(t *testing.T) {
		cCfg.Composites = map[ScoringMode]*CompositeConfig{
			"bad_weight_zero": {BaseModes: []ScoringMode{HotMode, RiskMode}, BlendWeights: map[ScoringMode]float64{HotMode: 0, RiskMode: 1}},
		}
		assert.Panics(t, func() { validateCompositeConfigs() })

		cCfg.Composites = map[ScoringMode]*CompositeConfig{
			"bad_weight_nan": {BaseModes: []ScoringMode{HotMode, RiskMode}, BlendWeights: map[ScoringMode]float64{HotMode: math.NaN(), RiskMode: 1}},
		}
		assert.Panics(t, func() { validateCompositeConfigs() })
	})
}
