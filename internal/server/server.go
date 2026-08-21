package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/compiler"
	"github.com/lewtec/superfolha/internal/db"
	appi18n "github.com/lewtec/superfolha/internal/i18n"
	"github.com/lewtec/superfolha/internal/paths"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/session"
	"github.com/lewtec/superfolha/internal/web"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type Server struct {
	repo           db.Repository
	stateDir       string
	resolver       *Resolver
	projectService *project.Service
	authService    *auth.Service
	hubs           *session.Registry
	bundle         *goi18n.Bundle
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
		bundle:         appi18n.NewBundle(),
	}
}

// CloseHubs flushes and drops live collaboration hubs (call on shutdown).
func (s *Server) CloseHubs() {
	if s.hubs != nil {
		s.hubs.CloseAll()
	}
}

// maxUploadBytes is the multipart form size limit for handleUploadFile (32 MiB).
const maxUploadBytes = 32 << 20

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle(paths.PatternStatic, http.StripPrefix("/", http.FileServer(http.FS(web.StaticFS))))

	mux.HandleFunc(paths.PatternLanding, s.handleLanding)
	mux.HandleFunc(paths.PatternLoginGet, s.handleLoginGet)
	mux.HandleFunc(paths.PatternLoginPost, s.handleLoginPost)
	mux.HandleFunc(paths.PatternRegisterGet, s.handleRegisterGet)
	mux.HandleFunc(paths.PatternRegisterPost, s.handleRegisterPost)
	mux.HandleFunc(paths.PatternLogout, s.handleLogoutPage)
	mux.HandleFunc(paths.PatternLang, s.handleLang)

	mux.HandleFunc(paths.PatternProjectsGet, s.requirePageUser(s.handleProjectsGet))
	mux.HandleFunc(paths.PatternProjectsPost, s.requirePageUser(s.handleProjectsPost))
	mux.HandleFunc(paths.PatternProjectDelete, s.requirePageUser(s.handleProjectDelete))
	mux.HandleFunc(paths.PatternEditorGet, s.requirePageUser(s.handleEditorGet))

	mux.Handle(paths.PatternCompile, http.HandlerFunc(s.handleCompile))
	mux.Handle(paths.PatternProjectWS, http.HandlerFunc(s.handleProjectWS))
	mux.Handle(paths.PatternUpload, http.HandlerFunc(s.handleUploadFile))
	mux.Handle(paths.PatternDownload, http.HandlerFunc(s.handleDownloadFile))
	mux.HandleFunc(paths.PatternAPILogout, s.handleLogout)

	return auth.Middleware(s.dropGhostUsers(mux))
}

// dropGhostUsers treats a valid JWT for a missing user as anonymous and
// clears the cookie (fresh state-dir after a leftover authToken).
func (s *Server) dropGhostUsers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if _, err := s.repo.GetUserByID(r.Context(), u.UserID); err != nil {
			auth.ClearAuthCookie(w)
			r = r.WithContext(auth.ContextWithoutUser(r.Context()))
		}
		next.ServeHTTP(w, r)
	})
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

	projectIdStr := r.PathValue(paths.ParamID)
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

	projectIdStr := r.PathValue(paths.ParamID)
	if projectIdStr == "" {
		writeAPIError(w, apierrors.New(apierrors.CodeInvalidInput, "missing project ID"))
		return
	}

	_, _, _, err := s.resolver.getAndCheckProject(r.Context(), projectIdStr)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	// Wildcard filePath: ServeMux unescapes path elements (same as id).
	rawFilePath := r.PathValue(paths.ParamFilePath)
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
