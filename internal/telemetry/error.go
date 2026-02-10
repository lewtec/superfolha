package telemetry

import (
	"context"
	"log"
)

// ReportError reports an error to the centralized error tracking system.
func ReportError(_ context.Context, err error) {
	log.Printf("ERROR: %v", err)
	// TODO: Add Sentry or other error reporting here
}
