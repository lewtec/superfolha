package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/project"
)

//go:embed templates
var templatesFS embed.FS

// MaxGraphQLFileSize defines the maximum file size (in bytes) for content to be returned directly via GraphQL.
const MaxGraphQLFileSize = 1024 * 1024 * 5 // 5 MB

// binaryExtensions lists well-known binary file extensions (lowercase, with dot).
// Kept in sync with the idea behind frontend/src/utils/fileUtils.ts BINARY_EXTENSIONS.
var binaryExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".bmp": {}, ".webp": {}, ".ico": {},
	".pdf": {},
	".zip": {}, ".tar": {}, ".gz": {}, ".rar": {}, ".7z": {},
	".exe": {}, ".dll": {}, ".bin": {}, ".out": {},
	".mp3": {}, ".wav": {}, ".ogg": {}, ".flac": {},
	".mp4": {}, ".avi": {}, ".mkv": {}, ".mov": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".otf": {},
	".sqlite": {}, ".db": {},
}

// HasBinary reports whether content should be treated as binary.
// Known binary extensions (from filename) win so short/empty blobs are not
// misclassified; otherwise falls back to http.DetectContentType (text/* => not binary).
func HasBinary(content []byte, filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := binaryExtensions[ext]; ok {
		return true
	}
	contentType := http.DetectContentType(content)
	return !strings.HasPrefix(contentType, "text/")
}

type Resolver struct {
	Repo           db.Repository
	StateDir       string
	projectService *project.Service
	authService    *auth.Service
}

func NewResolver(repo db.Repository, stateDir string, projectService *project.Service, authService *auth.Service) *Resolver {
	return &Resolver{
		Repo:           repo,
		StateDir:       stateDir,
		projectService: projectService,
		authService:    authService,
	}
}

// requireUser returns the authenticated user, or an error if missing/invalid.
// Me intentionally does not use this: unauthenticated Me returns (nil, nil).
func requireUser(ctx context.Context) (*auth.UserContext, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, apierrors.WithStatus(apierrors.CodeUnauthenticated, "not authenticated", 401)
	}
	if _, err := uuid.Parse(user.UserID); err != nil {
		return nil, apierrors.New(apierrors.CodeInvalidInput, "invalid user ID")
	}
	return user, nil
}

func toGraphQLUser(id, email string) *User {
	return &User{ID: id, Email: email}
}

func toGraphQLProject(p db.Project) *Project {
	return &Project{
		ID:        p.ID,
		Name:      p.Name,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func toGraphQLCommit(c *igit.Commit) *Commit {
	return &Commit{
		Hash:    c.Hash,
		Message: c.Message,
		Author:  c.Author,
		Date:    c.Date,
	}
}

// authPayloadFromResponse sets the session cookie and builds AuthPayload.
// op is used only in the missing-ResponseWriter log line (e.g. "Register", "Login").
func authPayloadFromResponse(ctx context.Context, authResp *auth.AuthResponse, op string) (*AuthPayload, error) {
	w, ok := ctx.Value(ResponseWriterContextKey).(http.ResponseWriter)
	if !ok {
		slog.Error("http.ResponseWriter not found in context", "op", op)
		return nil, apierrors.New(apierrors.CodeInternal, "response writer not available")
	}
	setAuthCookie(w, authResp.Token)
	return &AuthPayload{
		User: toGraphQLUser(authResp.User.ID, authResp.User.Email),
	}, nil
}

func (r *Resolver) getAndCheckProject(ctx context.Context, projectID string) (*db.Project, string, *auth.UserContext, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, "", nil, apierrors.WithStatus(apierrors.CodeUnauthenticated, "not authenticated", 401)
	}

	if _, err := uuid.Parse(projectID); err != nil {
		return nil, "", nil, apierrors.New(apierrors.CodeInvalidInput, "invalid project ID")
	}

	project, err := r.Repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, "", nil, apierrors.New(apierrors.CodeProjectNotFound, "project not found")
	}

	if project.UserID != user.UserID {
		return nil, "", nil, apierrors.WithStatus(apierrors.CodeUnauthorized, "not authorized", 403)
	}

	projectPath := r.projectService.GetProjectPath(projectID)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		if err := r.projectService.InitProjectRepo(projectID); err != nil {
			return nil, "", nil, apierrors.Internal(err)
		}

		templateDir := "templates/simple"
		err = fs.WalkDir(templatesFS, templateDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			content, err := templatesFS.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read template file %s: %w", path, err)
			}

			relativePath := strings.TrimPrefix(path, templateDir+"/")
			if err := r.projectService.SaveFile(projectID, relativePath, string(content)); err != nil {
				return fmt.Errorf("failed to write template file %s: %w", relativePath, err)
			}
			return nil
		})
		if err != nil {
			return nil, "", nil, apierrors.Internal(err)
		}
		_, err = r.projectService.CommitChanges(projectID, user.Email, "Initial commit")
		if err != nil {
			return nil, "", nil, apierrors.Internal(err)
		}
	}

	return &project, projectPath, user, nil
}
