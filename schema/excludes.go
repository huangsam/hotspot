package schema

import (
	_ "embed"
	"strings"
)

//go:embed data/default_excludes.txt
var defaultExcludesRaw string

// DefaultExclude represents the universal noise exclusion set applied across all analysis modes.
// It is formatted as a single comma-separated string for use in CLI flags and MCP defaults.
var DefaultExclude = formatExcludes(defaultExcludesRaw)

func formatExcludes(raw string) string {
	var parts []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		// Ignore empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, ", ")
}
