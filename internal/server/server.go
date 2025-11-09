package server

import (
	"database/sql"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/compiler"
)

//go:embed all:../../web/dist
var webFiles embed.FS

type Server struct {
	db       *sql.DB
	stateDir string
	resolver *Resolver
}

func NewServer(db *sql.DB, stateDir string) *Server {
	return &Server{
		db:       db,
		stateDir: stateDir,
		resolver: NewResolver(db, stateDir),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// GraphQL endpoint
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: s.resolver}))
	mux.Handle("/api/graphql", auth.Middleware(srv))

	// GraphQL Playground (for development)
	mux.Handle("/playground", playground.Handler("GraphQL playground", "/api/graphql"))

	// Compile endpoint
	mux.HandleFunc("/api/compile", s.handleCompile)

	// Serve SPA
	webFS, err := fs.Sub(webFiles, "web/dist")
	if err != nil {
		log.Printf("Warning: Failed to load embedded web files: %v", err)
		webFS = nil
	}

	if webFS != nil {
		fileServer := http.FileServer(http.FS(webFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Serve index.html for non-API routes
			if r.URL.Path == "/" || !fileExists(webFS, r.URL.Path[1:]) {
				r.URL.Path = "/"
			}
			fileServer.ServeHTTP(w, r)
		})
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("LaTeX Editor API Server - Frontend not embedded"))
		})
	}

	return mux
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get tarball file
	file, _, err := r.FormFile("tarball")
	if err != nil {
		http.Error(w, "Missing tarball file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read tarball data
	tarballData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read tarball", http.StatusBadRequest)
		return
	}

	// Compile
	result, err := compiler.Compile(tarballData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func fileExists(fsys fs.FS, path string) bool {
	_, err := fs.Stat(fsys, path)
	return err == nil
}
