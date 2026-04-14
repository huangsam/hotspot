package schema

import (
	"database/sql/driver"
	"fmt"
	"math"
	"strconv"
)

// Metric is a smart numeric type for the Hotspot engine.
// It serves as the core magnitude unit across the entire pipeline.
//
// Key Responsibilities:
//  1. Captures continuous magnitudes (e.g., fractional churn/commits).
//  2. Bridges V6 DB migration via sql.Scanner and driver.Valuer.
//  3. JSON marshals whole numbers as integers for backward compatibility.
//  4. Supports .Display() with intelligent formatting for fractions.
//
// This architecture makes Hotspot "Sponge-Ready"—capable of ingesting and blending
// fuzzy or weighted signals from non-git sources like JIRA, Slack, or CI logs.
type Metric float64

// Float64 returns the underlying float64 value.
func (n Metric) Float64() float64 {
	return float64(n)
}

// Int returns the value as a rounded integer.
func (n Metric) Int() int {
	return int(math.Round(float64(n)))
}

// Value implements the driver.Valuer interface for database persistence.
func (n Metric) Value() (driver.Value, error) {
	return float64(n), nil
}

// Scan implements the sql.Scanner interface for database retrieval.
func (n *Metric) Scan(src any) error {
	if src == nil {
		*n = 0
		return nil
	}

	switch v := src.(type) {
	case float64:
		*n = Metric(v)
	case int64:
		*n = Metric(float64(v))
	case []byte:
		// Handle cases where some drivers return numeric values as strings
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return fmt.Errorf("failed to scan Metric from string: %w", err)
		}
		*n = Metric(f)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("failed to scan Metric from string: %w", err)
		}
		*n = Metric(f)
	default:
		return fmt.Errorf("unsupported type for Metric scan: %T", src)
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to avoid decimal noise for whole numbers.
func (n Metric) MarshalJSON() ([]byte, error) {
	f := float64(n)
	// If it's effectively an integer, marshal it as one to preserve backward compatibility
	if f == math.Trunc(f) {
		return []byte(strconv.FormatInt(int64(f), 10)), nil
	}
	return []byte(strconv.FormatFloat(f, 'f', -1, 64)), nil
}

// UnmarshalJSON implements custom JSON unmarshaling.
func (n *Metric) UnmarshalJSON(data []byte) error {
	f, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return err
	}
	*n = Metric(f)
	return nil
}

// Display returns a human-friendly string representation.
// It hides decimals for whole numbers and uses sensible precision for fractions.
func (n Metric) Display() string {
	f := float64(n)
	if f == math.Trunc(f) {
		return fmt.Sprintf("%.0f", f)
	}
	// For fractional values, use up to 2 decimal places but trim trailing zeros
	return strconv.FormatFloat(f, 'f', 2, 64)
}

// String implements the fmt.Stringer interface.
func (n Metric) String() string {
	return n.Display()
}
