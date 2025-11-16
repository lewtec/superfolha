package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/compiler"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project" // Import project package
)

type Server struct {
	db             db.DBTX
	stateDir       string
	resolver       *Resolver
	projectService *project.Service // Added projectService
}

func NewServer(db db.DBTX, stateDir string, projectService *project.Service) *Server {
	return &Server{
		db:             db,
		stateDir:       stateDir,
		resolver:       NewResolver(db, stateDir, projectService), // Pass projectService here
		projectService: projectService,
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

	// Upload file endpoint
	mux.HandleFunc("/api/projects/{projectId}/upload-file", s.handleUploadFile)

	// Serve Web App
	mux.Handle("/", GetWebApp())

	return mux
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"})
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to parse form"})
		return
	}

	// Get tarball file
	file, _, err := r.FormFile("tarball")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing tarball file"})
		return
	}
	defer file.Close()

	// Read tarball data
	tarballData, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to read tarball"})
		return
	}

	// Compile
	result, err := compiler.Compile(tarballData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	// Return JSON response
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"})
		return
	}

	projectIdStr := r.PathValue("projectId")
	if projectIdStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing project ID"})
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to parse form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing file"})
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to read file content"})
		return
	}

	filePath := header.Filename // Use original filename as path for now

	// Use ProjectService to save the file
	err = s.projectService.SaveFile(projectIdStr, filePath, string(fileContent))
	if err != nil {
		log.Printf("Error saving file %s to project %s: %v", filePath, projectIdStr, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to save uploaded file"})
		return
	}

	// Use ProjectService to commit the change
	_, err = s.projectService.CommitChanges(projectIdStr, "System", "Uploaded file: "+filePath) // Use a placeholder author
	if err != nil {
		log.Printf("Error committing file %s to project %s: %v", filePath, projectIdStr, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to commit uploaded file"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "path": filePath})
}
