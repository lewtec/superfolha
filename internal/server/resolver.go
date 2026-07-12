package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
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

// HasBinary checks if the file content appears to be binary using http.DetectContentType.
func HasBinary(content []byte, filename string) bool {
	contentType := http.DetectContentType(content)
	if strings.HasPrefix(contentType, "text/") {
		return false
	}
	return true
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

func (r *Resolver) getProjectPath(projectID string) string {
	return r.projectService.GetProjectPath(projectID)
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
