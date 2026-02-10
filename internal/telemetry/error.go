package telemetry

import (
	"context"
	"log"
)

// ReportError reports an error to the centralized error tracking system.
// Currently it logs to stderr, but should be updated to use Sentry or similar.
func ReportError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	// In the future, this would send to Sentry/DataDog/etc.
	// We use log.Printf for now as a simple fallback.
	log.Printf("ERROR: %v", err)
}
