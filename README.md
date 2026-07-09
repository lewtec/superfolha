# Superfolha

A complete web-based LaTeX editor with Git version control, real-time compilation, and collaborative features.

## Features

- **Authentication**: JWT-based user authentication
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
- PostgreSQL with sqlc
- Git operations
- JWT authentication
- LaTeX compilation with pdflatex

### Frontend (React + TypeScript)
- Vite build tool
- Relay GraphQL client
- DaisyUI + Tailwind CSS
- CodeMirror 6 editor
- PDF.js viewer
- React Router

## Development

### Prerequisites
- [mise](https://mise.jdx.dev/) - Runtime and task manager
- PostgreSQL
- TexLive (for LaTeX compilation)

Go and Node.js will be automatically installed via mise.

### Quick Start

```bash
# mise will automatically install Go 1.21 and Node 20
mise install

# Install dependencies
mise run install

# Set environment variables
export DATABASE_URL="postgres://user:pass@localhost/latex_editor?sslmode=disable"
export JWT_SECRET="dev-secret-key-change-in-production"
export GO_ENV="development"

# Run development server
mise run dev
```

### Available Tasks

```bash
mise tasks                    # List all available tasks
mise run install             # Install all dependencies
mise run install-frontend    # Install frontend dependencies
mise run install-backend     # Install backend dependencies
mise run dev                 # Run development servers
mise run dev-frontend        # Run frontend dev server only
mise run dev-backend         # Run backend server only
mise run build               # Build complete application
mise run build-frontend      # Build frontend for production
mise run build-backend       # Build backend binary
mise run generate            # Generate GraphQL and sqlc code
mise run migrate             # Run database migrations
mise run clean               # Clean build artifacts
mise run docker              # Build Docker image
mise run test                # Run tests
mise run fmt                 # Format code
mise run lint                # Lint code
```

### Manual Setup

```bash
# Run server (migrations run automatically on startup; dev UI proxies to Vite)
go run ./cmd/superfolha --db="$DATABASE_URL" --state-dir=./data
```

## Production Build

```bash
# Using mise (builds SPA + embeds it with -tags release)
mise run build

# Manual
cd frontend && bun run build && cd ..
go build -tags release -o superfolha ./cmd/superfolha

# Run
./superfolha --db="postgres://..." --state-dir=/var/superfolha
```

## Docker

```bash
mise run docker

docker run -v /var/lib/superfolha:/data -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/superfolha?sslmode=require" \
  -e JWT_SECRET="your-secure-random-secret" \
  -e STATE_DIR=/data \
  superfolha
```

### Render

Deploy with the root `render.yaml` Blueprint (Docker web service + Postgres + 2 GB disk at `/data`). The image builds the SPA, embeds it with `-tags release`, and includes TeX Live for compilation.

### Railway

Use root `railway.toml` for build/deploy settings (Dockerfile, healthcheck, volume mount path `/data`). In the Railway project still add: Postgres (`DATABASE_URL`), a volume at `/data`, and `JWT_SECRET`.

## Configuration

### Environment Variables

- `DATABASE_URL`: PostgreSQL connection string (required)
- `JWT_SECRET`: Secret key for JWT token signing (required in production)
- `STATE_DIR`: Root for Git repositories (default: `./data`; Docker/Render: `/data`). Projects live at `{STATE_DIR}/repos/{uuid}`
- `PORT`: Server port (default: `8080`; Render injects this automatically)
- `GO_ENV`: Set to `development` for dev mode only (allows JWT_SECRET fallback; never set in production)

### CLI Flags

- `--db`: PostgreSQL connection string (overrides `DATABASE_URL`)
- `--state-dir`: Directory for Git repositories (overrides `STATE_DIR`)
- `--port`: Server port (overrides `PORT`)

### Example Configuration

```bash
export DATABASE_URL="postgres://user:pass@localhost/latex_editor?sslmode=disable"
export JWT_SECRET="your-secure-random-secret-key-min-32-chars"
export STATE_DIR="/var/superfolha/repos"
export PORT="8080"

./server
```

## Security Features

- **UUID v7**: Time-sortable UUIDs for better database performance and security
- **Bcrypt**: Password hashing with cost factor 12 (2024 recommended)
- **JWT Authentication**: Secure token-based authentication with 7-day expiration
- **Environment-based Secrets**: JWT secret must be set via environment variable in production
- **Password Validation**: Minimum 8 characters required
- **Repository Isolation**: Each project stored in isolated Git repository by UUID

## API Endpoints

- `GET /`: Serves SPA
- `POST /api/graphql`: GraphQL API
- `POST /api/compile`: LaTeX compilation endpoint
- `GET /playground`: GraphQL playground (development)

## License

MIT