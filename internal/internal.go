// Package internal has helpers that are only useful within the hotspot runtime.
package internal

// getTextLabel returns a text label indicating the criticality level
// based on the file's importance score:
// - Critical (≥80)
// - High (≥60)
// - Moderate (≥40)
// - Low (<40)
func getTextLabel(score float64) string {
	switch {
	case score >= 80:
		return "Critical"
	case score >= 60:
		return "High"
	case score >= 40:
		return "Moderate"
	default:
		return "Low"
	}
}
