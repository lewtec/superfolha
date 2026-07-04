package telemetry

import (
	"context"
	"log"
)

// ReportError centralizes error reporting for the backend.
// In a production environment, this would integrate with Sentry, DataDog, etc.
// All unexpected error paths should funnel through this function.
func ReportError(ctx context.Context, err error) {
	if err != nil {
		log.Printf("Reported error: %v", err)
	}
}
