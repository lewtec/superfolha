package server

import (
	"encoding/json"

	"io"

	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/lewtec/superfolha/internal/auth"

	"github.com/lewtec/superfolha/internal/compiler"

	"github.com/lewtec/superfolha/internal/db"

	"github.com/lewtec/superfolha/internal/git" // Import git package

	"os"

	"path"
)

type Server struct {
	db db.DBTX

	stateDir string

	resolver *Resolver
}

func NewServer(db db.DBTX, stateDir string) *Server {

	return &Server{

		db: db,

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

	// Upload file endpoint

	mux.HandleFunc("/api/projects/{projectId}/upload-file", s.handleUploadFile)

	// Serve Web App

	mux.Handle("/", GetWebApp())

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

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return

	}

	projectIdStr := r.PathValue("projectId")

	if projectIdStr == "" {

		http.Error(w, "Missing project ID", http.StatusBadRequest)

		return

	}

	err := r.ParseMultipartForm(32 << 20) // 32 MB max

	if err != nil {

		http.Error(w, "Failed to parse form", http.StatusBadRequest)

		return

	}

	file, header, err := r.FormFile("file")

	if err != nil {

		http.Error(w, "Missing file", http.StatusBadRequest)

		return

	}

	defer file.Close()

	fileContent, err := io.ReadAll(file)

	if err != nil {

		http.Error(w, "Failed to read file content", http.StatusInternalServerError)

		return

	}

	filePath := header.Filename // Use original filename as path for now

	// Save the file using git.SaveFile

	// This requires a context and user ID, which are not directly available here.

	// For now, let's directly write to the stateDir and then commit.

	// A more robust solution would involve refactoring the git.SaveFile to accept project ID and file content.

	projectPath := path.Join(s.stateDir, projectIdStr)

	fullFilePath := path.Join(projectPath, filePath)

	// Ensure the directory exists

	err = os.MkdirAll(path.Dir(fullFilePath), 0755)

	if err != nil {

		http.Error(w, "Failed to create directory", http.StatusInternalServerError)

		return

	}

	err = os.WriteFile(fullFilePath, fileContent, 0644)

	if err != nil {

		http.Error(w, "Failed to write file", http.StatusInternalServerError)

		return

	}

	// Commit the change

	// This also requires a context and user ID.

	// For simplicity, let's assume a generic commit message and user for now.

	// A proper solution would integrate with the existing git.Commit logic.

	// Add and commit the change

	err = git.AddAll(projectPath)

	if err != nil {

		http.Error(w, "Failed to stage uploaded file", http.StatusInternalServerError)

		return

	}

	_, err = git.CommitChanges(projectPath, "System", "Uploaded file: "+filePath) // Use a placeholder author

	if err != nil {

		http.Error(w, "Failed to commit uploaded file", http.StatusInternalServerError)

		return

	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "path": filePath})

}
