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
- Go 1.21+
- Node.js 18+
- PostgreSQL
- TexLive (for LaTeX compilation)
- Git

### Quick Start

```bash
# Install dependencies
make install-frontend
make install-backend

# Set database URL
export DATABASE_URL="postgres://user:pass@localhost/latex_editor?sslmode=disable"

# Run development server
make dev
```

### Backend Setup

```bash
# Run migrations (optional, server does this on boot)
dbmate up

# Run server
go run cmd/server/main.go --db="$DATABASE_URL" --state-dir=./data
```

### Frontend Setup

```bash
cd frontend
npm install
npm run dev  # Proxies API to :8080
```

## Production Build

```bash
# Using Makefile
make build

# Manual
cd frontend && npm run build && cd ..
go build -o server cmd/server/main.go

# Run
./server --db="postgres://..." --state-dir=/var/latex-editor
```

## Docker

```bash
make docker

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