package contract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// Scoring label constants.
const (
	CriticalValue = "Critical" // Critical value
	HighValue     = "High"     // High value
	ModerateValue = "Moderate" // Moderate value
	LowValue      = "Low"      // Low value
)

// Color variables for console output.
var (
	CriticalColor = color.New(color.FgRed, color.Bold)     // criticalColor represents standard danger.
	HighColor     = color.New(color.FgMagenta, color.Bold) // highColor represents strong, distinct warning.
	ModerateColor = color.New(color.FgYellow)              // moderateColor represents standard caution, not bold.
	LowColor      = color.New(color.FgCyan)                // lowColor represents informational / low-priority signal.
)

// GetPlainLabel returns a plain text label indicating the criticality level
// based on the file's importance score. This is the core logic used for
// CSV, JSON, and table printing.
func GetPlainLabel(score float64) string {
	switch {
	case score >= 80:
		return CriticalValue
	case score >= 60:
		return HighValue
	case score >= 40:
		return ModerateValue
	default:
		return LowValue
	}
}

// GetColorLabel returns a colored text label for console output (table).
// It uses GetPlainLabel to determine the string, and then applies the appropriate color.
func GetColorLabel(score float64) string {
	text := GetPlainLabel(score)

	switch text {
	case CriticalValue:
		return CriticalColor.Sprint(text)
	case HighValue:
		return HighColor.Sprint(text)
	case ModerateValue:
		return ModerateColor.Sprint(text)
	default: // "Low"
		return LowColor.Sprint(text)
	}
}

// SelectOutputFile returns the appropriate file handle for output, based on the provided
// file path and format type. It falls back to os.Stdout on error.
// This function replaces both selectCSVOutputFile and selectJSONOutputFile.
func SelectOutputFile(filePath string) (*os.File, error) {
	if filePath == "" {
		return os.Stdout, nil
	}
	return os.Create(filePath)
}

// ShouldIgnore returns true if the given path matches any of the exclude patterns.
// It supports simple glob patterns (using filepath.Match) when the pattern
// contains wildcard characters (*, ?, [ ]). Patterns ending with '/' are treated
// as prefixes. Patterns starting with '.' are treated as suffix (extension) matches.
// A user can provide patterns like "vendor/", "node_modules/", "*.min.js".
func ShouldIgnore(path string, excludes []string) bool {
	for _, ex := range excludes {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}

		// If the pattern contains glob characters, try filepath.Match.
		if strings.ContainsAny(ex, "*?[") || strings.Contains(ex, "**") {
			pat := strings.ReplaceAll(ex, "**", "*")
			if ok, err := filepath.Match(pat, path); err == nil && ok {
				return true
			}
			// Also try matching against the base filename (e.g. *.min.js)
			if ok, err := filepath.Match(pat, filepath.Base(path)); err == nil && ok {
				return true
			}
			continue
		}

		// Handle prefix, suffix, or substring matches
		switch {
		case strings.HasSuffix(ex, "/"):
			if strings.HasPrefix(path, ex) {
				return true
			}
		case strings.HasPrefix(ex, "."):
			if strings.HasSuffix(path, ex) {
				return true
			}
		case strings.Contains(path, ex):
			return true
		}
	}
	return false
}

// LogFatal logs an error and exits the program.
func LogFatal(msg string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "Fatal %s: %v\n", msg, err)
	os.Exit(1)
}

// LogWarn logs a warning message to stderr.
func LogWarn(msg string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "Warn %s: %v\n", msg, err)
}

// GetDBFilePath returns the path to the SQLite DB file for cache storage.
func GetDBFilePath() string {
	// Implementation from internal/persist_global.go
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".hotspot_cache.db"
	}
	return filepath.Join(homeDir, ".hotspot_cache.db")
}

// GetAnalysisDBFilePath returns the path to the SQLite DB file for analysis storage.
func GetAnalysisDBFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".hotspot_analysis.db"
	}
	return filepath.Join(homeDir, ".hotspot_analysis.db")
}

// NormalizeTimeseriesPath normalizes a user-provided path relative to the repo root
// and ensures it's within the repository boundaries.
func NormalizeTimeseriesPath(repoPath, userPath string) (string, error) {
	// Handle absolute paths by making them relative to repo
	if filepath.IsAbs(userPath) {
		relPath, err := filepath.Rel(repoPath, userPath)
		if err != nil {
			return "", fmt.Errorf("path is outside repository: %s", userPath)
		}
		userPath = relPath
	}

	// Clean the path to resolve any .. or . components
	cleanPath := filepath.Clean(userPath)

	// Ensure the path doesn't go outside the repo (no leading .. after cleaning)
	if strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("path is outside repository: %s", userPath)
	}

	// Convert to forward slashes for consistency with Git paths
	normalized := strings.ReplaceAll(cleanPath, string(filepath.Separator), "/")

	// Remove leading ./ if present
	normalized = strings.TrimPrefix(normalized, "./")

	return normalized, nil
}

// TruncatePath truncates a file path to a maximum width with ellipsis prefix.
// Requires maxWidth > 3 to ensure there's space for both the "..." prefix and at least one character of content.
// Without this check, small maxWidth values could cause slice bounds errors in the truncation calculation.
func TruncatePath(path string, maxWidth int) string {
	runes := []rune(path)
	if len(runes) > maxWidth && maxWidth > 3 {
		return "..." + string(runes[len(runes)-maxWidth+3:])
	}
	return path
}

// ParseBoolString parses a string value into a boolean.
// Accepts "yes", "no", "true", "false", "1", "0" (case-insensitive).
// Returns an error for invalid values.
func ParseBoolString(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "yes", "true", "1":
		return true, nil
	case "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean string: %s (expected yes/no/true/false/1/0)", s)
	}
}
