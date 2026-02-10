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
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/compiler"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/telemetry"
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
	mux.Handle("/api/projects/{projectID}/upload-file", auth.Middleware(http.HandlerFunc(s.handleUploadFile)))

	// Download file endpoint
	mux.Handle("/api/projects/{projectID}/download/{filePath...}", auth.Middleware(http.HandlerFunc(s.handleDownloadFile)))

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

	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing project query parameter"})
		return
	}

	_, _, user, err := s.resolver.getAndCheckProject(r.Context(), projectID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized) // getAndCheckProject returns "not authenticated" or "not authorized"
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
	// The compiler.Compile function needs to be updated to accept projectID, filePath, and projectService
	result, err := compiler.Compile(s.projectService, projectID, filePath)
	if err != nil {
		telemetry.ReportError(r.Context(), fmt.Errorf("compiling project %s file %s for user %s: %w", projectID, filePath, user.UserID, err))
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

	projectIDStr := r.PathValue("projectID")
	if projectIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing project ID"})
		return
	}

	_, _, _, err := s.resolver.getAndCheckProject(r.Context(), projectIDStr)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized) // getAndCheckProject returns "not authenticated" or "not authorized"
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
	err = s.projectService.SaveFile(projectIDStr, filePath, string(fileContent))
	if err != nil {
		telemetry.ReportError(r.Context(), fmt.Errorf("saving file %s to project %s: %w", filePath, projectIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to save uploaded file"})
		return
	}

	// Use ProjectService to commit the change
	_, err = s.projectService.CommitChanges(projectIDStr, "System", "Uploaded file: "+filePath) // Use a placeholder author
	if err != nil {
		telemetry.ReportError(r.Context(), fmt.Errorf("committing file %s to project %s: %w", filePath, projectIDStr, err))
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

	projectIDStr := r.PathValue("projectID")
	if projectIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Missing project ID"})
		return
	}

	_, _, _, err := s.resolver.getAndCheckProject(r.Context(), projectIDStr)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized) // getAndCheckProject returns "not authenticated" or "not authorized"
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}
	// filePath will be everything after /api/projects/{projectID}/download/
	// We need to decode the URL path because encodeURIComponent was used on the frontend
	encodedFilePath := r.URL.Path[len("/api/projects/"+projectIDStr+"/download/"):]
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

	fileReader, fileSize, err := s.projectService.ReadFile(projectIDStr, filePath)
	if err != nil {
		if err == project.ErrFileNotFound {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "File not found"})
			return
		}
		telemetry.ReportError(r.Context(), fmt.Errorf("reading file %s from project %s: %w", filePath, projectIDStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to read file"})
		return
	}
	defer fileReader.Close() // Ensure the file is closed

	// Read a small chunk to detect content type
	// http.DetectContentType needs at most 512 bytes
	var buf [512]byte
	n, _ := io.ReadFull(fileReader, buf[:])
	contentType := http.DetectContentType(buf[:n])

	// Reset the reader to the beginning for full content streaming
	// This requires the underlying reader to be seekable.
	// Since fileReader is an os.File, it is seekable.
	if seeker, ok := fileReader.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
	} else {
		// If not seekable, we would need to handle this differently,
		// e.g., by reading into a buffer and then prepending the buffer
		// to the stream, or by not detecting content type this way.
		// For os.File, this path should not be taken.
		log.Printf("Warning: fileReader is not seekable for %s", filePath)
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filePath+"\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize)) // Set Content-Length header

	_, err = io.Copy(w, fileReader) // Stream the file content
	if err != nil {
		telemetry.ReportError(r.Context(), fmt.Errorf("writing file content to response for %s: %w", filePath, err))
		// No need to set status code again, as headers might have been sent
	}
}
