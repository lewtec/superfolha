package telemetry

import (
	"context"
	"log"
)

// ReportError logs the error.
// Ideally, this should report to Sentry or similar service.
func ReportError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	log.Printf("Error: %v", err)
}
