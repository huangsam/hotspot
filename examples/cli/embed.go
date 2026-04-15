// Package clipresets contains embedded CLI preset configurations.
package clipresets

import (
	_ "embed"

	"github.com/huangsam/hotspot/schema"
)

//go:embed hotspot.small.yml
var smallPreset []byte

//go:embed hotspot.large.yml
var largePreset []byte

//go:embed hotspot.infra.yml
var infraPreset []byte

// PresetYAML returns the embedded YAML configuration for the given preset name.
// Unknown names fall back to the small preset.
func PresetYAML(name schema.PresetName) []byte {
	switch name {
	case schema.PresetLarge:
		return largePreset
	case schema.PresetInfra:
		return infraPreset
	default:
		return smallPreset
	}
}
