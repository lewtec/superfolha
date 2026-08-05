package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/appenv"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/compiler"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/session"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Server struct {
	repo           db.Repository
	stateDir       string
	resolver       *Resolver
	projectService *project.Service
	authService    *auth.Service
	hubs           *session.Registry
}

func NewServer(repo db.Repository, stateDir string, projectService *project.Service, authService *auth.Service) *Server {
	hubs := session.NewRegistry(projectService)
	return &Server{
		repo:           repo,
		stateDir:       stateDir,
		resolver:       NewResolver(repo, stateDir, projectService, authService, hubs),
		projectService: projectService,
		authService:    authService,
		hubs:           hubs,
	}
}

// CloseHubs flushes and drops live collaboration hubs (call on shutdown).
func (s *Server) CloseHubs() {
	if s.hubs != nil {
		s.hubs.CloseAll()
	}
}

// contextKey is a type for context keys to avoid collisions.
type contextKey string

// ResponseWriterContextKey is the key to store http.ResponseWriter in context.
const ResponseWriterContextKey contextKey = "responseWriter"

// maxUploadBytes is the multipart form size limit for handleUploadFile (32 MiB).
const maxUploadBytes = 32 << 20

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

	// GraphQL Playground — debug surface; only register in development
	if appenv.IsDevelopment() {
		mux.Handle("/api/graphiql", playground.Handler("GraphQL playground", "/api/graphql"))
	}

	// Compile endpoint
	mux.Handle("/api/compile", auth.Middleware(http.HandlerFunc(s.handleCompile)))

	// Live collaboration WebSocket (session fence + Yjs binary sync)
	mux.Handle("GET /ws/projects/{projectId}", auth.Middleware(http.HandlerFunc(s.handleProjectWS)))

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
	auth.ClearAuthCookie(w)
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

	filePath, err = project.DecodeFilePath(filePath)
	if err != nil {
		writeAPIError(w, mapPathAPIError(err))
		return
	}

	// When a live hub exists, flush collaborative text onto the working tree first.
	if s.hubs != nil {
		if hub := s.hubs.GetIfLive(projectId); hub != nil {
			if flushErr := hub.Flush(); flushErr != nil {
				slog.Error("flush before compile", "project", projectId, "err", flushErr)
				writeAPIError(w, apierrors.Wrap(apierrors.CodeInternal, "failed to flush project before compile", flushErr))
				return
			}
		}
	}

	result, err := compiler.Compile(r.Context(), s.projectService, projectId, filePath)
	if err != nil {
		slog.Error("error compiling project", "project", projectId, "file", filePath, "user", user.UserID, "err", err)
		if errors.Is(err, compiler.ErrLatexmkNotFound) {
			writeAPIError(w, apierrors.Wrap(apierrors.CodeCompileToolMissing, "latexmk not found", err))
			return
		}
		writeAPIError(w, apierrors.Wrap(apierrors.CodeCompileFailed, err.Error(), err))
		return
	}

	// Return JSON response
	if encErr := json.NewEncoder(w).Encode(result); encErr != nil {
		slog.Error("error encoding compile response", "err", encErr)
		return
	}
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

	_, _, user, err := s.resolver.getAndCheckProject(r.Context(), projectIdStr)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	err = r.ParseMultipartForm(maxUploadBytes)
	if err != nil {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "failed to parse form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing file"))
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Debug("close upload form file", "err", closeErr)
		}
	}()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		writeAPIError(w, apierrors.Internal(err))
		return
	}

	// Multipart Filename may include client path separators; jail via the same
	// path validator as GraphQL/download (basename-only upload name is still relative).
	filePath, err := project.ValidateRepoRelativePath(filepath.Base(header.Filename))
	if err != nil {
		writeAPIError(w, mapPathAPIError(err))
		return
	}
	body := string(fileContent)

	// Common path: live hub updates CRDT for text / disk for blobs and fans out tree.event.
	if s.hubs != nil {
		if hub := s.hubs.GetIfLive(projectIdStr); hub != nil {
			if err := hub.SaveTextFile(filePath, body); err != nil {
				if mapped := mapPathAPIError(err); mapped != nil {
					writeAPIError(w, mapped)
					return
				}
				slog.Error("error saving file via hub", "file", filePath, "project", projectIdStr, "err", err)
				writeAPIError(w, apierrors.Internal(err))
				return
			}
			// Notify peers of new/updated path (create or replace).
			if ev, mErr := json.Marshal(map[string]any{
				"type": "tree.event",
				"op":   "create",
				"path": filePath,
			}); mErr == nil {
				hub.BroadcastJSON(ev, "")
			}
			if _, err := hub.Commit("Uploaded file: "+filePath, user.Email); err != nil {
				slog.Error("error committing upload via hub", "file", filePath, "project", projectIdStr, "err", err)
				writeAPIError(w, apierrors.Internal(err))
				return
			}
			if encErr := json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "path": filePath}); encErr != nil {
				slog.Error("error encoding upload response", "err", encErr)
			}
			return
		}
	}

	err = s.projectService.SaveFile(projectIdStr, filePath, body)
	if err != nil {
		if mapped := mapPathAPIError(err); mapped != nil {
			writeAPIError(w, mapped)
			return
		}
		slog.Error("error saving file", "file", filePath, "project", projectIdStr, "err", err)
		writeAPIError(w, apierrors.Internal(err))
		return
	}

	_, err = s.projectService.CommitChanges(projectIdStr, user.Email, "Uploaded file: "+filePath)
	if err != nil {
		slog.Error("error committing file", "file", filePath, "project", projectIdStr, "err", err)
		writeAPIError(w, apierrors.Internal(err))
		return
	}

	if encErr := json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully", "path": filePath}); encErr != nil {
		slog.Error("error encoding upload response", "err", encErr)
		return
	}
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

	// Route is /api/projects/{projectId}/download/{filePath...}; PathValue returns the
	// wildcard segment (ServeMux unescapes path elements; same as projectId).
	rawFilePath := r.PathValue("filePath")
	if rawFilePath == "" {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing file path"))
		return
	}
	// DecodeFilePath still PathUnescape + jail: clients may double-encode, and
	// we must reject absolute / .. / .git components before disk I/O.
	filePath, err := project.DecodeFilePath(rawFilePath)
	if err != nil {
		writeAPIError(w, mapPathAPIError(err))
		return
	}

	fileReader, fileSize, err := s.projectService.ReadFile(projectIdStr, filePath)
	if err != nil {
		if mapped := mapPathAPIError(err); mapped != nil {
			writeAPIError(w, mapped)
			return
		}
		slog.Error("error reading file", "file", filePath, "project", projectIdStr, "err", err)
		writeAPIError(w, apierrors.Internal(err))
		return
	}
	defer func() {
		if closeErr := fileReader.Close(); closeErr != nil {
			slog.Debug("close download reader", "file", filePath, "err", closeErr)
		}
	}()

	// Read a small chunk to detect content type
	// http.DetectContentType needs at most 512 bytes
	var buf [512]byte
	n, err := io.ReadFull(fileReader, buf[:])
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		slog.Error("error peeking file content type", "file", filePath, "project", projectIdStr, "err", err)
		writeAPIError(w, apierrors.Internal(err))
		return
	}
	contentType := http.DetectContentType(buf[:n])

	// Reset the reader to the beginning for full content streaming
	// This requires the underlying reader to be seekable.
	// Since fileReader is an os.File, it is seekable.
	if seeker, ok := fileReader.(io.ReadSeeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			slog.Error("error seeking file after content-type peek", "file", filePath, "project", projectIdStr, "err", err)
			writeAPIError(w, apierrors.Internal(err))
			return
		}
	} else {
		// If not seekable, we would need to handle this differently,
		// e.g., by reading into a buffer and then prepending the buffer
		// to the stream, or by not detecting content type this way.
		// For os.File, this path should not be taken.
		slog.Warn("fileReader is not seekable", "file", filePath)
	}

	// Use basename only so nested paths are not leaked in the download name,
	// and FormatMediaType so quotes/newlines cannot break Content-Disposition.
	base := filepath.Base(filePath)
	cd := mime.FormatMediaType("attachment", map[string]string{"filename": base})
	if cd == "" {
		cd = `attachment; filename="download"`
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", cd)
	w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))

	_, err = io.Copy(w, fileReader) // Stream the file content
	if err != nil {
		slog.Error("error writing file content to response", "file", filePath, "err", err)
		// No need to set status code again, as headers might have been sent
	}
}

func codedErrorPresenter(ctx context.Context, e error) *gqlerror.Error {
	err := graphql.DefaultErrorPresenter(ctx, e)
	if coded, ok := apierrors.As(e); ok {
		if err.Extensions == nil {
			err.Extensions = map[string]any{}
		}
		err.Extensions["code"] = string(coded.Code)
		err.Message = coded.Message
		return err
	}
	if err.Extensions == nil {
		err.Extensions = map[string]any{}
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
			slog.Error("error encoding api error response", "err", encErr)
		}
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	if encErr := json.NewEncoder(w).Encode(apierrors.RESTBody{
		Code:    string(apierrors.CodeInternal),
		Message: err.Error(),
	}); encErr != nil {
		slog.Error("error encoding internal error response", "err", encErr)
	}
}
