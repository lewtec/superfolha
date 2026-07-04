# Project Conventions

## Error Handling

*   **Centralized Error Reporting:** All unexpected errors MUST be funneled through the centralized error reporting mechanisms. Silent error suppression (empty catch blocks) or raw logging of errors to console/log is strictly prohibited.
    *   **Frontend:** Use `reportError` from `frontend/src/utils/errorReporting.ts` instead of `console.error`.
    *   **Backend:** Use `telemetry.ReportError(ctx, err)` from `internal/telemetry/error.go` instead of standard `log.Printf` or `log.Println` for errors.

## Code Map

*   `frontend/src/utils/errorReporting.ts` -> Centralized error reporting for the React frontend.
*   `internal/telemetry/error.go` -> Centralized error reporting for the Go backend.
