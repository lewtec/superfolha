# Consistently Ignored Patterns

This file lists patterns that are consistently rejected in Pull Requests. Agents should consult this file before creating PRs to avoid noise.

## IGNORE: Agent Journals

**- Pattern:** Creating or updating journal files like `.jules/sentinel.md` or `.jules/janitor.md` in the PR.
**- Justification:** The project maintainers do not want internal agent logs or journals committed to the repository.
**- Files Affected:** `.jules/*.md`

## IGNORE: Silent Error Suppression

**- Pattern:** Using `_ = func()` or `defer func() { _ = ... }()` to explicitly suppress errors.
**- Justification:** Suppressing errors hides potential failures and makes debugging difficult. Handle errors properly or allow linters to flag them.
**- Files Affected:** `*.go`

## IGNORE: Unsolicited Tooling Changes

**- Pattern:** Renaming `mise` tasks (e.g., `gen` -> `codegen`), adding unrequested linters (e.g., `dprint`), or changing tool definitions without a clear directive.
**- Justification:** Disrupts established workflows and project conventions.
**- Files Affected:** `mise.toml`, `.github/workflows/*.yml`

## IGNORE: Mass Reformatting

**- Pattern:** Large-scale whitespace or formatting changes, especially in generated files (e.g., `**/__generated__/**`) or frontend components.
**- Justification:** Creates noise in diffs, conflicts with existing tooling, and overrides manual formatting preferences.
**- Files Affected:** `frontend/src/**/*.tsx`, `**/__generated__/**`

## IGNORE: Downgrading Dependencies

**- Pattern:** Downgrading versions in `go.mod` or `mise.toml` (e.g., Go 1.25 -> 1.24) without explicit instruction.
**- Justification:** The project aims to stay on pinned, modern versions. Downgrades are generally regressions.
**- Files Affected:** `go.mod`, `mise.toml`
