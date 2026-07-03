package telemetry

import (
	"context"
	"log"
)

// ReportError centralizes error reporting.
// It logs the error and could be extended to report to Sentry or other systems.
func ReportError(ctx context.Context, err error) {
	if err != nil {
		log.Printf("ReportError: %v\n", err)
	}
}
