package schema

import (
	"testing"
)

func TestConfigLoaderWeights(t *testing.T) {
	modes := []ScoringMode{HotMode, RiskMode, ComplexityMode, ROIMode}
	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			weights := GetDefaultWeights(mode)
			if len(weights) == 0 {
				t.Errorf("expected weights for mode %s, got none", mode)
			}

			// Verify total weight is non-zero (some might be exactly 1.0, some not, but none should be empty)
			total := 0.0
			for _, w := range weights {
				total += w
			}
			if total == 0 {
				t.Errorf("expected non-zero total weight for mode %s", mode)
			}
		})
	}
}

func TestConfigLoaderMetadata(t *testing.T) {
	modes := []ScoringMode{HotMode, RiskMode, ComplexityMode, ROIMode}
	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			purpose, factors, factorKeys := GetScoringModeMetadata(mode)
			if purpose == "" {
				t.Errorf("expected purpose for mode %s", mode)
			}
			if len(factors) == 0 {
				t.Errorf("expected factors for mode %s", mode)
			}
			if len(factorKeys) == 0 {
				t.Errorf("expected factorKeys for mode %s", mode)
			}
			if len(factors) != len(factorKeys) {
				t.Errorf("expected factors and factorKeys to have same length for mode %s", mode)
			}
		})
	}
}

func TestGetCompositeConfig(t *testing.T) {
	compositeModes := []ScoringMode{ActiveOwnersMode, RefactorNowMode, LegacyDebtMode}
	for _, mode := range compositeModes {
		t.Run(string(mode), func(t *testing.T) {
			cfg := GetCompositeConfig(mode)
			if cfg != nil {
				if len(cfg.BaseModes) < 2 {
					t.Errorf("expected at least two base modes for %s, got %d", mode, len(cfg.BaseModes))
				}
				if len(cfg.BlendWeights) < len(cfg.BaseModes) {
					t.Errorf("expected blend weights for all base modes for %s", mode)
				}
				for _, baseMode := range cfg.BaseModes {
					if !IsBaseMode(baseMode) {
						t.Errorf("composite mode %s has non-base base mode %s", mode, baseMode)
					}
				}
			} else {
				t.Errorf("expected composite config for mode %s", mode)
			}
		})
	}

	if GetCompositeConfig(ScoringMode("invalid_composite")) != nil {
		t.Errorf("expected nil for unknown composite mode")
	}
}
