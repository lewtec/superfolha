package telemetry

import (
	"context"
	"log"
)

// ReportError reports an error to the centralized error tracking system.
func ReportError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	// Fallback to log for now, but this is the centralized place.
	// In the future, this can be hooked up to Sentry or similar.
	log.Printf("Reported Error: %v", err)
}
