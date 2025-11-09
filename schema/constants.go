package schema

// Breakdown keys used in the scoring logic.
const (
	BreakdownContrib = "contrib" // nContrib
	BreakdownCommits = "commits" // nCommits
	BreakdownLOC     = "loc"     // nLOC
	BreakdownSize    = "size"    // nSize
	BreakdownAge     = "age"     // nAge
	BreakdownChurn   = "churn"   // nChurn

	BreakdownGini       = "gini"        // nGiniRaw
	BreakdownInvContrib = "inv_contrib" // nInvContrib
	BreakdownInvRecent  = "inv_recent"  // nInvRecentCommits (used in stale)
	BreakdownLowRecent  = "low_recent"  // nInvRecentCommits (used in complexity)
)

// All output modes supported.
const (
	CSVOut  = "csv"
	TextOut = "text"
	JSONOut = "json"
)

// All status supported.
const (
	NewStatus      = "new"
	ActiveStatus   = "active"
	InactiveStatus = "inactive"
	UnknownStatus  = "unknown"
)

// All scoring modes supported.
const (
	HotMode        = "hot"
	RiskMode       = "risk"
	ComplexityMode = "complexity"
	StaleMode      = "stale"
)

// GetDefaultWeights returns the default weight map for a given scoring mode.
func GetDefaultWeights(mode string) map[string]float64 {
	switch mode {
	case RiskMode:
		return map[string]float64{
			BreakdownAge:        0.16,
			BreakdownChurn:      0.06,
			BreakdownCommits:    0.04,
			BreakdownGini:       0.26,
			BreakdownInvContrib: 0.30,
			BreakdownLOC:        0.06,
			BreakdownSize:       0.12,
		}
	case ComplexityMode:
		return map[string]float64{
			BreakdownAge:       0.30,
			BreakdownChurn:     0.30,
			BreakdownCommits:   0.10,
			BreakdownLOC:       0.20,
			BreakdownLowRecent: 0.05,
			BreakdownSize:      0.05,
		}
	case StaleMode:
		return map[string]float64{
			BreakdownAge:       0.20,
			BreakdownCommits:   0.15,
			BreakdownContrib:   0.05,
			BreakdownInvRecent: 0.35,
			BreakdownSize:      0.25,
		}
	default: // HotMode
		return map[string]float64{
			BreakdownAge:     0.10,
			BreakdownChurn:   0.40,
			BreakdownCommits: 0.40,
			BreakdownContrib: 0.05,
			BreakdownSize:    0.05,
		}
	}
}
