## 2026-02-03 - Remove sensitive data logging

- **Issue:** `JWT_SECRET` was being logged to stdout.
- **Root Cause:** Debug logging left in production code.
- **Solution:** Removed the secret from the log message in `internal/auth/auth.go`.
- **Pattern:** Security/Privacy.
