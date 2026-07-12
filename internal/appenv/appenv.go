// Package appenv reports process environment flags used across packages.
package appenv

import "os"

// IsDevelopment reports whether GO_ENV is exactly "development".
func IsDevelopment() bool {
	return os.Getenv("GO_ENV") == "development"
}
