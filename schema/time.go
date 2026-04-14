package schema

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateTimeFormat is the default date time representation.
var DateTimeFormat = time.RFC3339

const (
	unitsPattern = `(year|month|week|day|hour|minute|y|mo|w|d|h|m)`
)

// Define the regular expressions to capture time values and units.
var (
	relativeTimeRe     = regexp.MustCompile(`^(\d+)\s*` + unitsPattern + `\s*s?\s+ago$`)
	lookbackDurationRe = regexp.MustCompile(`^(\d+)\s*` + unitsPattern + `\s*s?$`)
)

// parseRawUnit captures the value and normalized unit from a string match.
func parseRawUnit(matches []string) (int, string) {
	if len(matches) < 3 {
		return 0, ""
	}
	value, _ := strconv.Atoi(matches[1])
	return value, matches[2]
}

// ParseRelativeTime converts strings like "2 years ago" or "30d ago" into a time.Time in the past.
func ParseRelativeTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	matches := relativeTimeRe.FindStringSubmatch(s)

	if len(matches) == 0 {
		return time.Time{}, fmt.Errorf("invalid relative time format: %s", s)
	}

	value, unit := parseRawUnit(matches)

	switch unit {
	case "year", "y":
		return now.AddDate(-value, 0, 0), nil
	case "month", "mo":
		return now.AddDate(0, -value, 0), nil
	case "week", "w":
		// time.Duration uses nanoseconds, 7 * 24 * time.Hour is 1 week
		return now.Add(time.Duration(-value) * 7 * 24 * time.Hour), nil
	case "day", "d":
		return now.Add(time.Duration(-value) * 24 * time.Hour), nil
	case "hour", "h":
		return now.Add(time.Duration(-value) * time.Hour), nil
	case "minute", "m":
		return now.Add(time.Duration(-value) * time.Minute), nil
	default:
		// Should be caught by the regex, but good for safety
		return time.Time{}, fmt.Errorf("unsupported time unit: %s", unit)
	}
}

// ParseLookbackDuration converts strings like "3 months" or "720h" into a single time.Duration.
// It first tries custom parsing for human-readable formats (e.g., "30 days", "2 weeks", "30d"),
// then falls back to Go's built-in time.ParseDuration for standard formats.
func ParseLookbackDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	sLower := strings.ToLower(s)

	// Try custom parsing for human-readable formats first
	matches := lookbackDurationRe.FindStringSubmatch(sLower)
	if len(matches) > 0 {
		value, unit := parseRawUnit(matches)

		var totalDuration time.Duration

		switch unit {
		case "year", "y":
			// Approximation: 1 year ≈ 365 days
			totalDuration = time.Duration(value) * 365 * 24 * time.Hour
		case "month", "mo":
			// Approximation: 1 month ≈ 30 days
			totalDuration = time.Duration(value) * 30 * 24 * time.Hour
		case "week", "w":
			// Approximation: 1 week = 7 days
			totalDuration = time.Duration(value) * 7 * 24 * time.Hour
		case "day", "d":
			// Approximation: 1 day = 24 hours
			totalDuration = time.Duration(value) * 24 * time.Hour
		case "hour", "h":
			totalDuration = time.Duration(value) * time.Hour
		case "minute", "m":
			totalDuration = time.Duration(value) * time.Minute
		default:
			// Should be caught by the regex
			return 0, errors.New("unsupported time unit")
		}

		if totalDuration > 0 {
			return totalDuration, nil
		}
	}

	// Try Go's built-in duration parsing as a fallback (e.g., "168h30m")
	if duration, err := time.ParseDuration(s); err == nil {
		if duration == 0 {
			return 0, errors.New("zero duration is not useful")
		}
		return duration, nil
	}

	return 0, fmt.Errorf("invalid lookback duration format: %s", s)
}

// CalculateDaysBetween computes the number of days between two time points.
func CalculateDaysBetween(start, end time.Time) int {
	if end.Before(start) {
		return 0
	}
	return int(end.Sub(start) / (24 * time.Hour))
}

// CalculateDecayFactor computes a weighting factor [0,1] based on age in days.
// Formula: e^(-k * ageDays), where k = ln(2) / halfLifeDays.
func CalculateDecayFactor(ageDays float64, halfLifeDays float64) float64 {
	if halfLifeDays <= 0 {
		return 1.0 // No decay if half-life is non-positive
	}
	// k = ln(2) / halfLife
	k := math.Log(2) / halfLifeDays
	return math.Exp(-k * ageDays)
}
