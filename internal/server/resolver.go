package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
)

//go:embed templates
var templatesFS embed.FS

// MaxGraphQLFileSize defines the maximum file size (in bytes) for content to be returned directly via GraphQL.
const MaxGraphQLFileSize = 1024 * 1024 * 5 // 5 MB

// HasBinary checks if the file content appears to be binary using http.DetectContentType.
func HasBinary(content []byte, filename string) bool {
	contentType := http.DetectContentType(content)
	log.Printf("HasBinary: File %s detected content type: %s", filename, contentType)
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

func (r *Resolver) getAndCheckProject(ctx context.Context, projectID string) (*db.Project, string, *auth.UserContext, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, "", nil, errors.New("not authenticated")
	}

	if _, err := uuid.Parse(projectID); err != nil {
		return nil, "", nil, fmt.Errorf("invalid project ID: %w", err)
	}

	project, err := r.Repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("project not found")
	}

	if project.UserID != user.UserID {
		return nil, "", nil, fmt.Errorf("not authorized")
	}

	projectPath := r.projectService.GetProjectPath(projectID)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		if err := r.projectService.InitProjectRepo(projectID); err != nil {
			return nil, "", nil, fmt.Errorf("failed to init git repo: %w", err)
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
			return nil, "", nil, fmt.Errorf("failed to copy template files: %w", err)
		}
		_, err = r.projectService.CommitChanges(projectID, user.Email, "Initial commit")
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to create initial commit: %w", err)
		}
	}

	return &project, projectPath, user, nil
}
