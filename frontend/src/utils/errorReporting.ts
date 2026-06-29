/**
 * Centralized error reporting utility.
 * All unexpected errors must be funneled through this function instead of direct console.error.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function reportError(error: unknown, context?: Record<string, any>) {
  // If Sentry or another service is added later, wire it up here.
  // For now, we log to console with context.

  if (context && Object.keys(context).length > 0) {
    console.error("[ErrorReport] Context:", context);
  }

  console.error("[ErrorReport]", error);
}
