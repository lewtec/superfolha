# AI Agent Guidelines (AGENTS.md)

This file contains the rules and conventions that AI agents (such as Jules) must follow when contributing to this repository.

## Operational Memory

Where key concerns live in this repository:
- `cmd/superfolha/` -> Application entrypoints and bootstrapping.
- `internal/project/` -> Project business logic (lifecycle, git operations, file management). Moved away from GraphQL resolvers.
- `internal/server/schema.resolvers.go` -> GraphQL resolvers (must delegate to domain services; NO direct database or filesystem logic here).
- `internal/db/queries.sql` -> `sqlc` queries.
- `internal/server/cookies.go` -> Centralized HTTP cookie management, particularly for JWT.
- `internal/auth/` -> Authentication logic and low-level JWT handling.
- `internal/telemetry/error.go` -> Centralized error reporting (`telemetry.ReportError(ctx, err)`).
- `frontend/src/utils/errorReporting.ts` -> Centralized frontend error reporting (`reportError`).
- `frontend/src/i18n.ts` -> Internationalization configuration (detects browser language instead of manual switcher).
- `.github/workflows/autorelease.yml` -> CI/CD pipeline enforcing the Install -> Codegen -> PR (if changes) -> CI -> Release -> Artifacts sequence.

## Coding Conventions

### Backend (Go)
1. **Security - File Permissions:** Go code must comply with `gosec` security rules. Specifically, when using `os.WriteFile`, the file creation permissions **must** be exactly `0600`.
2. **Security - HTTP Server:** Always set explicit `ReadTimeout` and `WriteTimeout` on `http.Server` instances.
3. **Error Handling:** Empty catch blocks and silent failures (e.g., `_ = func()`) are strictly prohibited. All unexpected errors must be handled or reported using the centralized error-reporting function (`telemetry.ReportError`).
4. **Environment Variables in Tests:** Tests involving environment variables must use `t.Setenv(key, value)` instead of manual `os.Setenv` and `defer`.
5. **Import Aliasing:** When importing a third-party package that shares the same name as the local Go package (e.g., `github.com/go-git/go-git/v5` inside `internal/git`), it must be explicitly aliased in the import block to prevent typechecking and linting errors.

### Frontend
1. **Error Handling:** Centralized error reporting is implemented via `reportError` in `frontend/src/utils/errorReporting.ts`. Direct calls to `console.error` for unhandled exceptions are prohibited.
2. **Suspense & Relay:** The frontend uses Relay with React Suspense. Hooks like `useLazyLoadQuery` do not return a `loading` property; instead, components suspend during data fetching.

### Architecture & Workflows
1. **Unrequested Architectural Changes:** Extracting interfaces or restructuring code to decouple concrete implementations without explicit user requests is prohibited. Centralized error reporting is an explicit exception.
2. **Tooling & Versions:** All tools in `mise.toml` must be pinned to specific versions (usage of `latest` or `lts` is prohibited). This includes `go`, `bun`, `sqlc`, `tinytex` (e.g., `2026.02`), and `golangci-lint`.
3. **Mise Tasks:** Do not modify `mise` task names (e.g., renaming `gen` to `codegen`). Group tasks like `install`, `lint`, `fmt` must exclusively use wildcard dependencies (e.g., `["lint:*"]`).
4. **Generated Code:** Manually editing generated files (e.g., `*.sql.go`, `frontend/**/__generated__/**`) is strictly prohibited. Modifying the source files and running the respective code generator is required. Mass reformatting of generated files is prohibited.
5. **Artifacts:** Never commit tooling, bootstrap, or download artifacts (e.g., `install-mise.sh`, downloaded binaries, temporary installers) or LEWTEC agent skills to the repository.
