package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/compiler"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Server struct {
	repo           db.Repository
	stateDir       string
	resolver       *Resolver
	projectService *project.Service // Added projectService
	authService    *auth.Service
}

func NewServer(repo db.Repository, stateDir string, projectService *project.Service, authService *auth.Service) *Server {
	return &Server{
		repo:           repo,
		stateDir:       stateDir,
		resolver:       NewResolver(repo, stateDir, projectService, authService),
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
	srv.SetErrorPresenter(codedErrorPresenter)
	mux.Handle("/api/graphql", ResponseWriterMiddleware(auth.Middleware(srv)))

	// GraphQL Playground (for development)
	mux.Handle("/api/graphiql", playground.Handler("GraphQL playground", "/api/graphql"))

	// Compile endpoint
	mux.Handle("/api/compile", auth.Middleware(http.HandlerFunc(s.handleCompile)))

	// Upload file endpoint
	mux.Handle("/api/projects/{projectId}/upload-file", auth.Middleware(http.HandlerFunc(s.handleUploadFile)))

	// Download file endpoint
	mux.Handle("/api/projects/{projectId}/download/{filePath...}", auth.Middleware(http.HandlerFunc(s.handleDownloadFile)))

	// Logout clears the HttpOnly session cookie (must be done server-side).
	mux.HandleFunc("POST /api/logout", s.handleLogout)

	// Serve Web App
	mux.Handle("/", GetWebApp())

	return mux
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	secure := os.Getenv("GO_ENV") != "development"
	http.SetCookie(w, &http.Cookie{
		Name:     "authToken",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCompile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		writeAPIError(w, apierrors.WithStatus(apierrors.CodeInvalidInput, "method not allowed", http.StatusMethodNotAllowed))
		return
	}

	projectId := r.URL.Query().Get("project")
	if projectId == "" {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing project query parameter"))
		return
	}

	_, _, user, err := s.resolver.getAndCheckProject(r.Context(), projectId)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing file query parameter"))
		return
	}

	result, err := compiler.Compile(r.Context(), s.projectService, projectId, filePath)
	if err != nil {
		log.Printf("Error compiling project %s file %s for user %s: %v", projectId, filePath, user.UserID, err)
		if strings.Contains(err.Error(), "latexmk command not found") {
			writeAPIError(w, apierrors.Wrap(apierrors.CodeCompileToolMissing, "latexmk not found", err))
			return
		}
		writeAPIError(w, apierrors.Wrap(apierrors.CodeCompileFailed, err.Error(), err))
		return
	}

	// Return JSON response
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		writeAPIError(w, apierrors.WithStatus(apierrors.CodeInvalidInput, "method not allowed", http.StatusMethodNotAllowed))
		return
	}

	projectIdStr := r.PathValue("projectId")
	if projectIdStr == "" {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing project ID"))
		return
	}

	_, _, _, err := s.resolver.getAndCheckProject(r.Context(), projectIdStr)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	err = r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "failed to parse form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing file"))
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		writeAPIError(w, apierrors.Internal(err))
		return
	}

	filePath := header.Filename

	err = s.projectService.SaveFile(projectIdStr, filePath, string(fileContent))
	if err != nil {
		log.Printf("Error saving file %s to project %s: %v", filePath, projectIdStr, err)
		writeAPIError(w, apierrors.Internal(err))
		return
	}

	_, err = s.projectService.CommitChanges(projectIdStr, "System", "Uploaded file: "+filePath)
	if err != nil {
		log.Printf("Error committing file %s to project %s: %v", filePath, projectIdStr, err)
		writeAPIError(w, apierrors.Internal(err))
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "path": filePath})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeAPIError(w, apierrors.WithStatus(apierrors.CodeInvalidInput, "method not allowed", http.StatusMethodNotAllowed))
		return
	}

	projectIdStr := r.PathValue("projectId")
	if projectIdStr == "" {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing project ID"))
		return
	}

	_, _, _, err := s.resolver.getAndCheckProject(r.Context(), projectIdStr)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	// filePath will be everything after /api/projects/{projectId}/download/
	// We need to decode the URL path because encodeURIComponent was used on the frontend
	encodedFilePath := r.URL.Path[len("/api/projects/"+projectIdStr+"/download/"):]
	filePath, err := project.DecodeFilePath(encodedFilePath)
	if err != nil {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "invalid file path"))
		return
	}

	if filePath == "" {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing file path"))
		return
	}

	fileReader, fileSize, err := s.projectService.ReadFile(projectIdStr, filePath)
	if err != nil {
		if err == project.ErrFileNotFound {
			writeAPIError(w, apierrors.New(apierrors.CodeFileNotFound, "file not found"))
			return
		}
		log.Printf("Error reading file %s from project %s: %v", filePath, projectIdStr, err)
		writeAPIError(w, apierrors.Internal(err))
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
		log.Printf("Error writing file content to response for %s: %v", filePath, err)
		// No need to set status code again, as headers might have been sent
	}
}

func codedErrorPresenter(ctx context.Context, e error) *gqlerror.Error {
	err := graphql.DefaultErrorPresenter(ctx, e)
	if coded, ok := apierrors.As(e); ok {
		if err.Extensions == nil {
			err.Extensions = map[string]interface{}{}
		}
		err.Extensions["code"] = string(coded.Code)
		err.Message = coded.Message
		return err
	}
	if err.Extensions == nil {
		err.Extensions = map[string]interface{}{}
	}
	if _, has := err.Extensions["code"]; !has {
		err.Extensions["code"] = string(apierrors.CodeUnknown)
	}
	return err
}

func writeAPIError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	if coded, ok := apierrors.As(err); ok {
		w.WriteHeader(coded.Status())
		if encErr := json.NewEncoder(w).Encode(coded.RESTBody()); encErr != nil {
			log.Printf("error encoding api error response: %v", encErr)
		}
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	if encErr := json.NewEncoder(w).Encode(apierrors.RESTBody{
		Code:    string(apierrors.CodeInternal),
		Message: err.Error(),
	}); encErr != nil {
		log.Printf("error encoding internal error response: %v", encErr)
	}
}
