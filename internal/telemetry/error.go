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
	// For now, just log it. In the future, this can be connected to Sentry.
	log.Printf("ERROR: %v", err)
}
