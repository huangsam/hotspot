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
