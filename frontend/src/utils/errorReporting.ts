/**
 * Centralized error reporting utility.
 * All unexpected errors must funnel through this function.
 */
export function reportError(...args: unknown[]): void {
  // In a production environment, this would integrate with Sentry or similar service.

  console.error(...args);
}
