package internal

import (
	"fmt"
	"regexp"

	"github.com/huangsam/hotspot/schema"
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

// quoteTableName returns the properly quoted table name for the given backend.
func quoteTableName(name string, backend schema.CacheBackend) string {
	switch backend {
	case schema.PostgreSQLBackend:
		return fmt.Sprintf("\"%s\"", name)
	case schema.MySQLBackend:
		return fmt.Sprintf("`%s`", name)
	default: // SQLite
		return fmt.Sprintf("\"%s\"", name)
	}
}
