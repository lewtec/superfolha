package db

import "strings"

// InferDriver picks a backend from a DSN/path.
// postgres:// or postgresql:// → "postgres"; otherwise "sqlite".
func InferDriver(dsn string) string {
	lower := strings.ToLower(strings.TrimSpace(dsn))
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return "postgres"
	}
	return "sqlite"
}
