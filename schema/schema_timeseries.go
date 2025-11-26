package schema

import "time"

// TimeseriesPoint represents a single data point in the timeseries.
type TimeseriesPoint struct {
	Period   string        `json:"period"`
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Score    float64       `json:"score"`
	Path     string        `json:"path"`
	Owners   []string      `json:"owners"`   // Top owners for this time period
	Mode     ScoringMode   `json:"mode"`     // Scoring mode used (hot, risk, complexity, stale)
	Lookback time.Duration `json:"lookback"` // Dynamic lookback duration for this point
}

// TimeseriesResult holds the timeseries data points.
type TimeseriesResult struct {
	Points []TimeseriesPoint `json:"points"`
}
