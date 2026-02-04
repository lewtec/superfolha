# Sentinel Journal 🛡️

## 2026-02-04 - Fix Path Traversal in Repository Manager

**Vulnerability:** Path Traversal (Zip Slip / Directory Traversal) in `ProjectRepository.SaveFile`, `DeleteFile`, and `ReadFile`.
**Severity:** CRITICAL
**Impact:** Allows attackers to overwrite or read arbitrary files on the server filesystem by supplying crafted file paths (e.g., `../../etc/passwd`).
**Fix:** Implemented `validatePath` helper to ensure all file operations are confined within the repository directory.
**Verification:** Added unit tests to confirm that paths containing `..` or escaping the root are rejected.
