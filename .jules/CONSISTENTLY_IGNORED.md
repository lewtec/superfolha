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

**- Pattern:** Renaming `mise` tasks (e.g., `gen` -> `codegen`), adding unrequested linters (e.g., `dprint`, `golangci-lint`), or changing tool definitions without a clear directive.
**- Justification:** Disrupts established workflows and project conventions.
**- Files Affected:** `mise.toml`, `.github/workflows/*.yml`, `.golangci.yml`

## IGNORE: Mass Reformatting

**- Pattern:** Large-scale whitespace or formatting changes, especially in generated files (e.g., `**/__generated__/**`) or frontend components.
**- Justification:** Creates noise in diffs, conflicts with existing tooling, and overrides manual formatting preferences.
**- Files Affected:** `frontend/src/**/*.tsx`, `**/__generated__/**`

## IGNORE: Downgrading Dependencies

**- Pattern:** Downgrading versions in `go.mod` or `mise.toml` (e.g., Go 1.25 -> 1.24) without explicit instruction.
**- Justification:** The project aims to stay on pinned, modern versions. Downgrades are generally regressions.
**- Files Affected:** `go.mod`, `mise.toml`

## IGNORE: Unrequested Architectural Changes

**- Pattern:** Extracting interfaces (e.g., `ProjectReader`), moving logic across files or services (e.g., from GraphQL resolvers to `ProjectService`), or introducing arbitrary new packages without explicit user request (NOTE: centralized error reporting is an exception and IS explicitly mandated by global instructions).
**- Justification:** Significant architectural changes are out-of-scope and clutter Pull Requests. Keep changes localized to the requested task.
**- Files Affected:** `internal/**/*.go`

## IGNORE: Manually Editing Generated Code

**- Pattern:** Modifying generated files manually (e.g., `internal/db/queries.sql.go` or frontend `__generated__` files) instead of updating the source file and running the code generator.
**- Justification:** Changes to generated code will be overwritten and lost the next time the generator runs. Always modify the source and regenerate.
**- Files Affected:** `*.sql.go`, `**/__generated__/**`

## IGNORE: Unpinned Tool Versions

**- Pattern:** Using `latest`, `lts`, or unpinned versions for tools in `mise.toml` (e.g., `bun = "latest"`).
**- Justification:** The project requires deterministic and reproducible builds. All tools must be pinned to explicit, specific versions.
**- Files Affected:** `mise.toml`

## IGNORE: Committing Bootstrap Artifacts

**- Pattern:** Committing local setup scripts, downloaded binaries, or temporary installers like `install-mise.sh` or `install.sh`.
**- Justification:** These are environment-specific transient files that pollute the repository and pose security risks.
**- Files Affected:** `install*.sh`, `*.tar.gz`, `*.zip`
