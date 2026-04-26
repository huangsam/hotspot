package iocache

import (
	"fmt"
	"regexp"
	"strings"
)

// validateTableName validates that the table name is a safe SQL identifier.
// It ensures the name consists only of alphanumeric characters and underscores,
// starting with a letter or underscore, to prevent SQL injection.
func validateTableName(name string) error {
	if name == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	matched, err := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	if err != nil {
		return fmt.Errorf("error validating table name: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid table name: %s (must match pattern ^[a-zA-Z_][a-zA-Z0-9_]*$)", name)
	}
	return nil
}

// ensureSQLitePragmas appends recommended SQLite pragmas to the connection string.
// Currently it adds busy_timeout(5000) to ensure concurrent processes wait
// rather than failing immediately with SQLITE_BUSY.
func ensureSQLitePragmas(connStr string) string {
	if strings.Contains(connStr, "?") {
		return connStr + "&_pragma=busy_timeout(5000)"
	}
	return connStr + "?_pragma=busy_timeout(5000)"
}
