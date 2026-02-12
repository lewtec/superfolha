package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/compiler"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
)

type Server struct {
	db             db.DBTX
	stateDir       string
	resolver       *Resolver
	projectService *project.Service
	authService    *auth.Service
}

func NewServer(db db.DBTX, stateDir string, projectService *project.Service, authService *auth.Service) *Server {
	return &Server{
		db:             db,
		stateDir:       stateDir,
		resolver:       NewResolver(db, stateDir, projectService, authService),
		projectService: projectService,
		authService:    authService,
	}
}

// contextKey is a type for context keys to avoid collisions.
type contextKey string

// ResponseWriterContextKey is the key to store http.ResponseWriter in context.
const ResponseWriterContextKey contextKey = "responseWriter"

// ResponseWriterMiddleware adds the http.ResponseWriter to the request context.
func ResponseWriterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ResponseWriterContextKey, w)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// GraphQL endpoint
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: s.resolver}))
	mux.Handle("/api/graphql", ResponseWriterMiddleware(auth.Middleware(srv)))

	// GraphQL Playground (for development)
	mux.Handle("/api/graphiql", playground.Handler("GraphQL playground", "/api/graphql"))

	// Compile endpoint
	mux.Handle("/api/compile", auth.Middleware(http.HandlerFunc(s.handleCompile)))

	// Upload file endpoint
	mux.Handle("/api/projects/{projectId}/upload-file", auth.Middleware(http.HandlerFunc(s.handleUploadFile)))

	// Download file endpoint
	mux.Handle("/api/projects/{projectId}/download/{filePath...}", auth.Middleware(http.HandlerFunc(s.handleDownloadFile)))

	// Serve Web App
	mux.Handle("/", GetWebApp())

	return mux
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"})
		return
	}

	projectId := r.URL.Query().Get("project")
	if projectId == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing project query parameter"})
		return
	}

	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "not authenticated"})
		return
	}
	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid user id"})
		return
	}

	_, err = s.projectService.GetProjectForUser(r.Context(), projectId, userID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing file query parameter"})
		return
	}

	// Compile
	result, err := compiler.Compile(s.projectService, projectId, filePath)
	if err != nil {
		log.Printf("Error compiling project %s file %s for user %s: %v", projectId, filePath, user.UserID, err)
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

	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "not authenticated"})
		return
	}
	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid user id"})
		return
	}

	_, err = s.projectService.GetProjectForUser(r.Context(), projectIdStr, userID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	err = r.ParseMultipartForm(32 << 20) // 32 MB max
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

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "not authenticated"})
		return
	}
	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid user id"})
		return
	}

	_, err = s.projectService.GetProjectForUser(r.Context(), projectIdStr, userID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	// filePath will be everything after /api/projects/{projectId}/download/
	encodedFilePath := r.URL.Path[len("/api/projects/"+projectIdStr+"/download/"):]
	filePath, err := project.DecodeFilePath(encodedFilePath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid file path"})
		return
	}

	if filePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing file path"})
		return
	}

	fileReader, fileSize, err := s.projectService.ReadFile(projectIdStr, filePath)
	if err != nil {
		if err == project.ErrFileNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "File not found"})
			return
		}
		log.Printf("Error reading file %s from project %s: %v", filePath, projectIdStr, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to read file"})
		return
	}
	defer fileReader.Close() // Ensure the file is closed

	// Read a small chunk to detect content type
	var buf [512]byte
	n, _ := io.ReadFull(fileReader, buf[:])
	contentType := http.DetectContentType(buf[:n])

	// Reset the reader to the beginning for full content streaming
	if seeker, ok := fileReader.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
	} else {
		log.Printf("Warning: fileReader is not seekable for %s", filePath)
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filePath+"\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))

	_, err = io.Copy(w, fileReader) // Stream the file content
	if err != nil {
		log.Printf("Error writing file content to response for %s: %v", filePath, err)
	}
}
