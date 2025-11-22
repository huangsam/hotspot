package schema

// Custom string types for type safety.
type (
	// BreakdownKey represents keys used in scoring breakdowns.
	BreakdownKey string

	// OutputMode represents the format of the output.
	OutputMode string

	// Status represents the status of a file.
	Status string

	// ScoringMode represents the scoring mode used.
	ScoringMode string

	// DatabaseBackend represents the database backend for caching.
	DatabaseBackend string
)

// Breakdown keys used in the scoring logic.
const (
	BreakdownContrib BreakdownKey = "contrib" // nContrib
	BreakdownCommits BreakdownKey = "commits" // nCommits
	BreakdownLOC     BreakdownKey = "loc"     // nLOC
	BreakdownSize    BreakdownKey = "size"    // nSize
	BreakdownAge     BreakdownKey = "age"     // nAge
	BreakdownChurn   BreakdownKey = "churn"   // nChurn

	BreakdownGini       BreakdownKey = "gini"        // nGiniRaw
	BreakdownInvContrib BreakdownKey = "inv_contrib" // nInvContrib
	BreakdownInvRecent  BreakdownKey = "inv_recent"  // nInvRecentCommits (used in stale)
	BreakdownLowRecent  BreakdownKey = "low_recent"  // nInvRecentCommits (used in complexity)
)

// All output modes supported.
const (
	CSVOut     OutputMode = "csv"
	TextOut    OutputMode = "text" // default
	JSONOut    OutputMode = "json"
	ParquetOut OutputMode = "parquet"
)

// All status supported.
const (
	NewStatus      Status = "new"
	ActiveStatus   Status = "active"
	InactiveStatus Status = "inactive"
	UnknownStatus  Status = "unknown"
)

// All scoring modes supported.
const (
	HotMode        ScoringMode = "hot" // default
	RiskMode       ScoringMode = "risk"
	ComplexityMode ScoringMode = "complexity"
	StaleMode      ScoringMode = "stale"
)

// All cache backends supported.
const (
	SQLiteBackend     DatabaseBackend = "sqlite" // default
	MySQLBackend      DatabaseBackend = "mysql"
	PostgreSQLBackend DatabaseBackend = "postgresql"
	NoneBackend       DatabaseBackend = "none"
)

// AllScoringModes returns a list of all supported scoring modes.
var AllScoringModes = []ScoringMode{HotMode, RiskMode, ComplexityMode, StaleMode}

// ValidOutputModes lists all valid output modes.
var ValidOutputModes = map[OutputMode]struct{}{
	CSVOut:     {},
	TextOut:    {},
	JSONOut:    {},
	ParquetOut: {},
}

// ValidScoringModes lists all valid scoring modes.
var ValidScoringModes = map[ScoringMode]struct{}{
	HotMode:        {},
	RiskMode:       {},
	ComplexityMode: {},
	StaleMode:      {},
}

// ValidCacheBackends lists all valid cache backends.
var ValidCacheBackends = map[DatabaseBackend]struct{}{
	SQLiteBackend:     {},
	MySQLBackend:      {},
	PostgreSQLBackend: {},
	NoneBackend:       {},
}

// GetDefaultWeights returns the default weight map for a given scoring mode.
func GetDefaultWeights(mode ScoringMode) map[BreakdownKey]float64 {
	switch mode {
	case RiskMode:
		return map[BreakdownKey]float64{
			BreakdownAge:        0.16,
			BreakdownChurn:      0.06,
			BreakdownCommits:    0.04,
			BreakdownGini:       0.26,
			BreakdownInvContrib: 0.30,
			BreakdownLOC:        0.06,
			BreakdownSize:       0.12,
		}
	case ComplexityMode:
		return map[BreakdownKey]float64{
			BreakdownAge:       0.30,
			BreakdownChurn:     0.30,
			BreakdownCommits:   0.10,
			BreakdownLOC:       0.20,
			BreakdownLowRecent: 0.05,
			BreakdownSize:      0.05,
		}
	case StaleMode:
		return map[BreakdownKey]float64{
			BreakdownAge:       0.20,
			BreakdownCommits:   0.15,
			BreakdownContrib:   0.05,
			BreakdownInvRecent: 0.35,
			BreakdownSize:      0.25,
		}
	default: // HotMode
		return map[BreakdownKey]float64{
			BreakdownAge:     0.10,
			BreakdownChurn:   0.40,
			BreakdownCommits: 0.40,
			BreakdownContrib: 0.05,
			BreakdownSize:    0.05,
		}
	}
}
