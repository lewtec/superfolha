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
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	projectId := r.URL.Query().Get("project")
	if projectId == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Missing project query parameter"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	_, _, user, err := s.resolver.getAndCheckProject(r.Context(), projectId)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": err.Error()}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Missing file query parameter"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	// Compile
	result, err := compiler.Compile(s.projectService, projectId, filePath)
	if err != nil {
		telemetry.ReportError(r.Context(), fmt.Errorf("Error compiling project %s file %s for user %s: %w", projectId, filePath, user.UserID, err))
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": err.Error()}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	// Return JSON response
	if err := json.NewEncoder(w).Encode(result); err != nil {
		telemetry.ReportError(r.Context(), err)
	}
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	projectIdStr := r.PathValue("projectId")
	if projectIdStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Missing project ID"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	_, _, _, err := s.resolver.getAndCheckProject(r.Context(), projectIdStr)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": err.Error()}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	err = r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Failed to parse form"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Missing file"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Failed to read file content"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	filePath := header.Filename

	err = s.projectService.SaveFile(projectIdStr, filePath, string(fileContent))
	if err != nil {
		telemetry.ReportError(r.Context(), fmt.Errorf("Error saving file %s to project %s: %w", filePath, projectIdStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Failed to save uploaded file"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	_, err = s.projectService.CommitChanges(projectIdStr, "System", "Uploaded file: "+filePath)
	if err != nil {
		telemetry.ReportError(r.Context(), fmt.Errorf("Error committing file %s to project %s: %w", filePath, projectIdStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Failed to commit uploaded file"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "path": filePath}); err != nil {
		telemetry.ReportError(r.Context(), err)
	}
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Method not allowed"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	projectIdStr := r.PathValue("projectId")
	if projectIdStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Missing project ID"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	_, _, _, err := s.resolver.getAndCheckProject(r.Context(), projectIdStr)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": err.Error()}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	encodedFilePath := r.URL.Path[len("/api/projects/"+projectIdStr+"/download/"):]
	filePath, err := project.DecodeFilePath(encodedFilePath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Invalid file path"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	if filePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Missing file path"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}

	fileReader, fileSize, err := s.projectService.ReadFile(projectIdStr, filePath)
	if err != nil {
		if err == project.ErrFileNotFound {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"message": "File not found"}); err != nil {
				telemetry.ReportError(r.Context(), err)
			}
			return
		}
		telemetry.ReportError(r.Context(), fmt.Errorf("Error reading file %s from project %s: %w", filePath, projectIdStr, err))
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Failed to read file"}); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
		return
	}
	defer fileReader.Close()

	var buf [512]byte
	n, _ := io.ReadFull(fileReader, buf[:])
	contentType := http.DetectContentType(buf[:n])

	if seeker, ok := fileReader.(io.ReadSeeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			telemetry.ReportError(r.Context(), err)
		}
	} else {
		log.Printf("Warning: fileReader is not seekable for %s", filePath)
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filePath+"\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))

	if _, err = io.Copy(w, fileReader); err != nil {
		telemetry.ReportError(r.Context(), err)
	}
}
