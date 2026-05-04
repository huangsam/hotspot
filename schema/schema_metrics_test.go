package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildMetricsRenderModel_UsesBaseModesOnly(t *testing.T) {
	model := BuildMetricsRenderModel(nil)
	assert.Len(t, model.Modes, len(BaseScoringModes))

	var names []string
	for _, mode := range model.Modes {
		names = append(names, mode.Name)
	}

	assert.Contains(t, names, string(HotMode))
	assert.Contains(t, names, string(RiskMode))
	assert.Contains(t, names, string(ComplexityMode))
	assert.Contains(t, names, string(ROIMode))
	assert.NotContains(t, names, string(ActiveOwnersMode))
	assert.NotContains(t, names, string(RefactorNowMode))
	assert.NotContains(t, names, string(LegacyDebtMode))
}
