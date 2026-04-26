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

	// DatabaseBackend represents the database used for analysis, migration, etc.
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
	BreakdownLowRecent  BreakdownKey = "low_recent"  // nInvRecentCommits (Staleness / Decay)
)

// All output modes supported.
const (
	CSVOut      OutputMode = "csv"
	TextOut     OutputMode = "text" // default
	JSONOut     OutputMode = "json"
	ParquetOut  OutputMode = "parquet"
	Describe    OutputMode = "describe"
	MarkdownOut OutputMode = "markdown"
	HeatmapOut  OutputMode = "heatmap"
	NoneOut     OutputMode = "none"
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
	ROIMode        ScoringMode = "roi"
)

// Scoring label constants for criticality levels.
const (
	CriticalValue = "Critical" // Critical value
	HighValue     = "High"     // High value
	ModerateValue = "Moderate" // Moderate value
	LowValue      = "Low"      // Low value
)

// All cache backends supported.
const (
	SQLiteBackend     DatabaseBackend = "sqlite" // default
	MySQLBackend      DatabaseBackend = "mysql"
	PostgreSQLBackend DatabaseBackend = "postgresql"
	NoneBackend       DatabaseBackend = "none"
)

// AllScoringModes returns a list of all supported scoring modes.
var AllScoringModes = []ScoringMode{HotMode, RiskMode, ComplexityMode, ROIMode}

// ValidOutputModes lists all valid output modes.
var ValidOutputModes = map[OutputMode]struct{}{
	CSVOut:      {},
	TextOut:     {},
	JSONOut:     {},
	ParquetOut:  {},
	Describe:    {},
	MarkdownOut: {},
	HeatmapOut:  {},
	NoneOut:     {},
}

// ValidScoringModes lists all valid scoring modes.
var ValidScoringModes = map[ScoringMode]struct{}{
	HotMode:        {},
	RiskMode:       {},
	ComplexityMode: {},
	ROIMode:        {},
}

// ValidDatabaseBackends lists all valid database backends.
var ValidDatabaseBackends = map[DatabaseBackend]struct{}{
	SQLiteBackend:     {},
	MySQLBackend:      {},
	PostgreSQLBackend: {},
	NoneBackend:       {},
}
