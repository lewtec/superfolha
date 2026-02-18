package server

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http" // Added net/http import for DetectContentType
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project" // Import project package
)

//go:embed templates
var templatesFS embed.FS

// MaxGraphQLFileSize defines the maximum file size (in bytes) for content to be returned directly via GraphQL.
const MaxGraphQLFileSize = 1024 * 1024 * 5 // 5 MB

// HasBinary checks if the file content appears to be binary using http.DetectContentType.
func HasBinary(content []byte, filename string) bool {
	// http.DetectContentType sniffs the first 512 bytes.
	// If the content is shorter, it sniffs the entire content.
	contentType := http.DetectContentType(content)

	// Log the detected content type for debugging
	log.Printf("HasBinary: File %s detected content type: %s", filename, contentType)

	// If the content type starts with "text/", it's considered text.
	// Otherwise, it's considered binary.
	// application/octet-stream is the default for unknown binary types.
	if strings.HasPrefix(contentType, "text/") {
		return false
	}

	return true
}

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB             db.DBTX
	StateDir       string
	projectService *project.Service // Added projectService
	authService    *auth.Service
}

func NewResolver(db db.DBTX, stateDir string, projectService *project.Service, authService *auth.Service) *Resolver {
	return &Resolver{
		DB:             db,
		StateDir:       stateDir,
		projectService: projectService, // Initialize projectService
		authService:    authService,
	}
}

func (r *Resolver) getAndCheckProject(ctx context.Context, projectID string) (*db.Project, string, *auth.UserContext, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, "", nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if !bytes.Equal(project.UserID.Bytes[:], userUUID[:]) {
		return nil, "", nil, fmt.Errorf("not authorized")
	}

	projectPath := r.projectService.GetProjectPath(projectID)
	// Check if repo exists, if not, initialize it and copy templates
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		if err := r.projectService.InitProjectRepo(projectID); err != nil {
			return nil, "", nil, fmt.Errorf("failed to init git repo: %w", err)
		}

		// Copy template files
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
		// Initial commit
		_, err = r.projectService.CommitChanges(projectID, user.Email, "Initial commit")
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to create initial commit: %w", err)
		}
	}

	return &project, projectPath, user, nil
}
