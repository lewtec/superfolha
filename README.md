<p align="center">
  <img src="internal/web/static/brand/logo.png" width="128" height="128" alt="Superfolha" />
</p>

# Superfolha

A complete web-based LaTeX editor with Git version control, real-time compilation, and collaborative features.

## Features

- **Authentication**: Ed25519 challenge-sign in the browser. Each session has its own SSH key for git.
- **Projects**: Create and manage multiple LaTeX projects
- **File Management**: Full file explorer with Git integration
- **Editor**: CodeMirror 6 with LaTeX syntax highlighting and autocomplete
- **Compilation**: Real-time LaTeX to PDF compilation with error reporting
- **PDF Viewer**: Built-in PDF.js viewer with navigation
- **Version Control**: Git-based version control for every project
- **Responsive UI**: DaisyUI components with light/dark themes

## Tech Stack

### Backend (Go)
- HTTP server with stdlib router
- GraphQL API via gqlgen
- SQLite (modernc.org/sqlite) via sqlc; PostgreSQL driver still present but **deprecated** (single-instance target)
- Git operations
- JWT authentication
- LaTeX compilation with latexmk / TeX Live

### Frontend (templ + JS island)
- Server-rendered [templ](https://templ.guide) pages (Backstage-style forms + 303)
- DaisyUI 5 + Tailwind 4 (bun → `/static/style.css`)
- CodeMirror 6 + Yjs editor island (`/static/editor.js`)
- Browser PDF viewer (blob iframe)
- go-i18n (en / es / pt)

## Development

### Prerequisites
- [mise](https://mise.jdx.dev/) - Runtime and task manager
- TexLive / latexmk (for LaTeX compilation)

Go and Node.js will be automatically installed via mise. Default is SQLite (no server). Optional **PostgreSQL 18+** for multi-instance (migrations use native `uuidv7()`).

### Quick Start

```bash
mise install
mise run install

export JWT_SECRET="dev-secret-key-change-in-production"
export GO_ENV="development"

# CSS + editor island + templ
mise run codegen

# SQLite (default)
go run ./cmd/superfolha --state-dir=./data

# or Postgres 18+ (provides uuidv7() for migrations)
# mise run devdb:up   # docker postgres:18 on :5423
# export DB_DRIVER=postgres
# export DATABASE_URL="postgres://admin:admin@localhost:5423/superfolha?sslmode=disable"
# go run ./cmd/superfolha --state-dir=./data
```

If `DATABASE_URL` / `--db` is omitted with SQLite, the server uses `{state-dir}/superfolha.db`.
A `postgres://` DSN auto-selects the postgres driver when `DB_DRIVER` is unset.
Postgres migrations require **18+** (`DEFAULT uuidv7()`).

### Available Tasks

```bash
mise tasks
mise run install
mise run build
mise run gen
mise run docker
mise run test
mise run fmt
mise run lint
```

## Production Build

```bash
mise run build

# or
go generate ./...
go build -o superfolha ./cmd/superfolha

./superfolha --state-dir=/var/superfolha --db=/var/superfolha/superfolha.db
```

## Docker

**Local multi-stage build** (SPA + Go + TeX Live), same as Railway/Render:

```bash
mise run docker   # docker build -f Dockerfile.build -t superfolha .

docker run -v /var/lib/superfolha:/data -p 8080:8080 \
  -e JWT_SECRET="your-secure-random-secret" \
  -e STATE_DIR=/data \
  -e DATABASE_URL=/data/superfolha.db \
  superfolha
```

**Published image** (GoReleaser → `ghcr.io/lewtec/superfolha`, TeX Live base + prebuilt binary):

```bash
docker pull ghcr.io/lewtec/superfolha:latest
docker run --rm -p 8080:8080 \
  -e JWT_SECRET="your-secure-random-secret" \
  -v superfolha-data:/data \
  ghcr.io/lewtec/superfolha:latest
```

Root `Dockerfile` is the **GoReleaser runtime** image (copies `$TARGETPLATFORM/superfolha` into `texlive/texlive`). Full source builds use `Dockerfile.build`.

### Releases

Versioning is **git tags via [svu](https://github.com/caarlos0/svu)** — no `version.txt` / `make_release`.

```bash
# Local (needs GHCR login + docker buildx):
mise run release -- next    # svu next  (from conventional commits)
mise run release -- patch   # force patch bump
mise run release -- minor
mise run release -- major

# CI: Autorelease workflow (push to main, Saturday cron, or workflow_dispatch)
# → mise run release -- next|patch|…
```

That tags with `svu`, pushes the tag, then **GoReleaser** publishes binaries, checksums, and `ghcr.io/lewtec/superfolha` (TeX Live base).

### Render

Deploy with root `render.yaml` (Docker web service + 2 GB disk at `/data`). SQLite and Git repos share the volume. No managed Postgres.

### Railway

Use root `railway.toml`. Attach a volume at `/data`, set `JWT_SECRET`. SQLite defaults to `/data/superfolha.db` when `STATE_DIR=/data`.

## Configuration

### Environment Variables

- `DB_DRIVER` / `DATABASE_DRIVER`: `sqlite` (default) or `postgres`
- `DATABASE_URL` / `DATABASE_DSN`: SQLite path/`file:` DSN, or `postgres://...` URL
- `JWT_SECRET`: Secret key for JWT token signing (required in production)
- `GITHUB_APP_ID`, `GITHUB_APP_CLIENT_ID`, `GITHUB_APP_CLIENT_SECRET`, `GITHUB_APP_PRIVATE_KEY`, `GITHUB_APP_SLUG`: GitHub App for login and installation tokens
- `STATE_DIR`: Root for Git repositories (default: `./data`; Docker: `/data`). Projects live at `{STATE_DIR}/repos/{uuid}`
- `PORT`: Used as listen port when `--addr` is not set (platforms like Railway inject this)
- `GO_ENV`: Set to `development` for dev mode only (allows JWT_SECRET fallback)

### CLI Flags

- `--db-driver`: `sqlite` or `postgres` (overrides `DB_DRIVER`)
- `--db`: DSN/path (overrides `DATABASE_URL`)
- `--state-dir`: State directory (overrides `STATE_DIR`)
- `--addr`: Listen address (default: `:$PORT` if `PORT` is set, else `127.0.0.1:8080`)

### Example Configuration

```bash
export JWT_SECRET="your-secure-random-secret-key-min-32-chars"
export STATE_DIR="/var/superfolha"
export DATABASE_URL="/var/superfolha/superfolha.db"
# optional; default is 127.0.0.1:8080 without PORT
# export PORT=8080

./superfolha --addr=0.0.0.0:8080
```

## Security Features

- **UUID v7**: Time-sortable UUIDs generated in application code
- **Bcrypt**: Password hashing with cost factor 12
- **JWT Authentication**: Secure token-based authentication with 7-day expiration
- **Environment-based Secrets**: JWT secret must be set via environment variable in production
- **Password Validation**: Minimum 8 characters required
- **Repository Isolation**: Each project stored in isolated Git repository by UUID

## API Endpoints

- `GET /`: Landing
- `GET/POST /login`, `/register`; `POST /logout`; `POST /lang`
- `GET/POST /projects`; `POST /projects/{id}/delete`
- `GET /editor/{id}`: templ chrome + editor island
- `GET /api/compile`: LaTeX compilation (`?project=&file=`); flushes live CRDT hub first when present
- `GET /ws/projects/{id}`: Live collaboration WebSocket (JWT session; hello/hello.ack fence + Yjs)
- `POST /api/logout`: Clears session cookie

Live collab is documented in `SPEC.md` (ygo server CRDT, Git working tree projection, single-instance).

## License

MIT
