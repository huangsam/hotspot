package internal

import (
	"fmt"
)

// PrintMetricsDefinitions displays the formal definitions of all scoring modes.
// This is a static display that does not require Git analysis.
func PrintMetricsDefinitions() error {
	fmt.Println("🔥 Hotspot Scoring Modes")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("All scores = weighted sum of normalized factors")
	fmt.Println("Weights loaded from hotspot.yaml config file")
	fmt.Println()

	modes := []struct {
		name    string
		purpose string
		factors string
		formula string
	}{
		{
			name:    "🔥 HOT",
			purpose: "Activity hotspots - high recent activity & volatility",
			factors: "Commits, Churn, Contributors, Age, Size",
			formula: "0.40*Commits + 0.40*Churn + 0.05*Contributors + 0.05*Size + 0.10*Age",
		},
		{
			name:    "⚠️  RISK",
			purpose: "Knowledge risk/bus factor - concentrated ownership",
			factors: "InvContributors, Gini, Age, Churn, Commits, LOC, Size",
			formula: "0.30*InvContributors + 0.26*Gini + 0.16*Age + 0.06*Churn + 0.04*Commits + 0.06*LOC + 0.12*Size",
		},
		{
			name:    "🧩 COMPLEXITY",
			purpose: "Technical debt - large, old files with high maintenance burden",
			factors: "Age, Churn, Commits, LOC, LowRecent, Size",
			formula: "0.30*Age + 0.30*Churn + 0.10*Commits + 0.20*LOC + 0.05*LowRecent + 0.05*Size",
		},
		{
			name:    "🕰️  STALE",
			purpose: "Maintenance debt - important files untouched recently",
			factors: "InvRecent, Age, Size, Commits, Contributors",
			formula: "0.35*InvRecent + 0.20*Age + 0.25*Size + 0.15*Commits + 0.05*Contributors",
		},
	}

	for _, mode := range modes {
		fmt.Printf("%s: %s\n", mode.name, mode.purpose)
		fmt.Printf("   Factors: %s\n", mode.factors)
		fmt.Printf("   Formula: Score = %s\n", mode.formula)
		fmt.Println()
	}

	fmt.Println("🔗 Special Relationship")
	fmt.Println("RISK Score = HOT Score / Ownership Diversity Factor")
	fmt.Println("(Factor ↓ when few contributors → RISK Score ↑)")
	fmt.Println()
	fmt.Println("⚙️  Configuration")
	fmt.Println("Custom weights in .hotspot.yaml:")
	fmt.Println("  weights:")
	fmt.Println("    hot: {commits: 0.40, churn: 0.40, contrib: 0.05, age: 0.10, size: 0.05}")
	fmt.Println("    risk: {inv_contrib: 0.30, gini: 0.26, age: 0.16, churn: 0.06, commits: 0.04, loc: 0.06, size: 0.12}")
	fmt.Println("    # ... etc for complexity & stale")

	return nil
}
