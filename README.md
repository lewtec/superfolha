# LaTeX Editor Web

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

# Set database URL
export DATABASE_URL="postgres://user:pass@localhost/latex_editor?sslmode=disable"

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
# Run migrations (optional, server does this on boot)
mise run migrate
# or: dbmate up

# Run server
go run cmd/server/main.go --db="$DATABASE_URL" --state-dir=./data
```

## Production Build

```bash
# Using mise
mise run build

# Manual
cd frontend && npm run build && cd ..
go build -o server cmd/server/main.go

# Run
./server --db="postgres://..." --state-dir=/var/latex-editor
```

## Docker

```bash
mise run docker

docker run -v /data:/data -p 8080:8080 \
  -e DATABASE_URL="postgres://..." \
  latex-editor
```

## Configuration

Via CLI flags or environment variables:

- `--state-dir` / `STATE_DIR`: Directory for Git repositories (default: `./data`)
- `--db` / `DATABASE_URL`: PostgreSQL connection string (required)
- `--port` / `PORT`: Server port (default: `8080`)

## API Endpoints

- `GET /`: Serves SPA
- `POST /api/graphql`: GraphQL API
- `POST /api/compile`: LaTeX compilation endpoint
- `GET /playground`: GraphQL playground (development)

## License

MIT