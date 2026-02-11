package telemetry

import (
	"context"
	"log"
)

// ReportError reports an error to the centralized error tracking system.
// Currently, it logs the error to the standard logger.
func ReportError(ctx context.Context, err error) {
	// In a real application, this would send the error to Sentry or another service.
	log.Printf("Error: %v", err)
}
