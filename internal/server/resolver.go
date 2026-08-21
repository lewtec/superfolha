package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/session"
)

//go:embed templates
var templatesFS embed.FS

// MaxGraphQLFileSize defines the maximum file size (in bytes) for content to be returned directly via GraphQL.
const MaxGraphQLFileSize = project.MaxCollabTextBytes

// HasBinary reports whether content should be treated as binary.
// Delegates to project.IsBinary (shared with CRDT collab classification).
func HasBinary(content []byte, filename string) bool {
	return project.IsBinary(content, filename)
}

type Resolver struct {
	Repo           db.Repository
	StateDir       string
	projectService *project.Service
	authService    *auth.Service
	hubs           *session.Registry
}

func NewResolver(repo db.Repository, stateDir string, projectService *project.Service, authService *auth.Service, hubs *session.Registry) *Resolver {
	return &Resolver{
		Repo:           repo,
		StateDir:       stateDir,
		projectService: projectService,
		authService:    authService,
		hubs:           hubs,
	}
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
	if _, err := os.Stat(projectPath); errors.Is(err, fs.ErrNotExist) {
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
				return fmt.Errorf("read template file %s: %w", path, err)
			}

			relativePath := strings.TrimPrefix(path, templateDir+"/")
			if err := r.projectService.SaveFile(projectID, relativePath, string(content)); err != nil {
				return fmt.Errorf("write template file %s: %w", relativePath, err)
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
