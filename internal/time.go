package internal

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Define the regular expression to capture "N [units] ago"
// e.g., "2 years ago", "3 months ago", "1 week ago".
var relativeTimeRe = regexp.MustCompile(`^(\d+)\s+(year|month|week|day|hour|minute)s?\s+ago$`)

// parseRelativeTime converts strings like "2 years ago" into a time.Time in the past.
func parseRelativeTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	matches := relativeTimeRe.FindStringSubmatch(s)

	if len(matches) == 0 {
		return time.Time{}, fmt.Errorf("invalid relative time format: %s", s)
	}

	// 1: Value (e.g., "2")
	// 2: Unit (e.g., "year" or "month")
	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "year":
		return now.AddDate(-value, 0, 0), nil
	case "month":
		return now.AddDate(0, -value, 0), nil
	case "week":
		// time.Duration uses nanoseconds, 7 * 24 * time.Hour is 1 week
		return now.Add(time.Duration(-value) * 7 * 24 * time.Hour), nil
	case "day":
		return now.Add(time.Duration(-value) * 24 * time.Hour), nil
	case "hour":
		return now.Add(time.Duration(-value) * time.Hour), nil
	case "minute":
		return now.Add(time.Duration(-value) * time.Minute), nil
	default:
		// Should be caught by the regex, but good for safety
		return time.Time{}, fmt.Errorf("unsupported time unit: %s", unit)
	}
}

// Define the regular expression to capture "N [units]".
var lookbackDurationRe = regexp.MustCompile(`^(\d+)\s+(year|month|week|day|hour|minute)s?$`)

// ParseLookbackDuration converts strings like "3 months" or "720h" into a single time.Duration.
// It first tries Go's built-in time.ParseDuration for standard formats, then falls back
// to custom parsing for human-readable formats.
func ParseLookbackDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Try Go's built-in duration parsing first (e.g., "720h", "168h", "30m")
	if duration, err := time.ParseDuration(s); err == nil {
		if duration == 0 {
			return 0, errors.New("zero duration is not useful")
		}
		return duration, nil
	}

	// Fall back to custom parsing for human-readable formats (e.g., "30 days", "2 weeks")
	s = strings.ToLower(s)
	matches := lookbackDurationRe.FindStringSubmatch(s)

	if len(matches) == 0 {
		return 0, fmt.Errorf("invalid lookback duration format: %s", s)
	}

	// 1: Value (e.g., "2")
	// 2: Unit (e.g., "year" or "month")
	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	var totalDuration time.Duration

	switch unit {
	case "year":
		// Approximation: 1 year ≈ 365 days
		totalDuration = time.Duration(value) * 365 * 24 * time.Hour
	case "month":
		// Approximation: 1 month ≈ 30 days
		totalDuration = time.Duration(value) * 30 * 24 * time.Hour
	case "week":
		// Approximation: 1 week = 7 days
		totalDuration = time.Duration(value) * 7 * 24 * time.Hour
	case "day":
		// Approximation: 1 day = 24 hours
		totalDuration = time.Duration(value) * 24 * time.Hour
	case "hour":
		totalDuration = time.Duration(value) * time.Hour
	case "minute":
		totalDuration = time.Duration(value) * time.Minute
	default:
		// Should be caught by the regex
		return 0, errors.New("unsupported time unit")
	}

	if totalDuration == 0 {
		return 0, errors.New("zero duration is not useful")
	}

	return totalDuration, nil
}

// CalculateAgeDays computes the duration in days from the given start time to now.
func CalculateAgeDays(start time.Time) int {
	d := time.Since(start)
	// The base calculation uses precise time.Duration integer arithmetic
	// We divide the total duration by the duration of a single day (24 hours)
	days := int(d / (24 * time.Hour))

	// Special case: If the file is only *just* over 24 hours old (24h <= age < 36h),
	// we treat it as 0 days old. The 36h margin (1.5 days) accounts for clock skew,
	// DST changes, and the time window truncation used for caching
	if d >= 24*time.Hour && d < 36*time.Hour {
		days = 0
	}
	return days
}
