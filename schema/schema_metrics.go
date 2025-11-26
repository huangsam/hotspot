package schema

// MetricsMode represents a scoring mode for display purposes.
type MetricsMode struct {
	Name       string             `json:"name"`
	Purpose    string             `json:"purpose"`
	Factors    []string           `json:"factors"`
	FactorKeys []string           `json:"factor_keys,omitempty"` // Only used for JSON output
	Weights    map[string]float64 `json:"weights,omitempty"`     // Only used for JSON output
	Formula    string             `json:"formula,omitempty"`     // Only used for JSON output
}

// MetricsRenderModel contains all processed data needed for displaying metrics definitions.
type MetricsRenderModel struct {
	Title               string                `json:"title"`
	Description         string                `json:"description"`
	Modes               []MetricsModeWithData `json:"modes"`
	SpecialRelationship map[string]string     `json:"special_relationship"`
}

// MetricsModeWithData extends MetricsMode with computed weights and formula.
type MetricsModeWithData struct {
	MetricsMode
	Weights map[string]float64 `json:"weights"`
	Formula string             `json:"formula"`
}
