//go:build basic || database

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCompositeModesCLI runs the built hotspot binary against the repository
// and verifies running with a composite mode returns composite-mode outputs.
func TestCompositeModesCLI(t *testing.T) {
	modes := []string{"active_owners", "refactor_now", "legacy_debt"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			out, err := runHotspotCommand(t, "files", "--mode", mode, "--output", "json", "--limit", "1")
			assert.NoError(t, err)
			s := string(out)
			// Output is pretty-printed JSON; look for the key/value pair with spacing.
			assert.Contains(t, s, "\"mode_type\": \"composite\"")
		})
	}
}
