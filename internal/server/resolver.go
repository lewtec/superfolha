package server

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath" // Added filepath import
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

// File extensions typically associated with binary files that shouldn't be read as text
var binaryExtensions = map[string]bool{
	".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true, ".exe": true,
	".dll": true, ".bin": true, ".mp3": true, ".wav": true, ".ogg": true, ".mp4": true,
	".avi": true, ".mkv": true, ".mov": true, ".woff": true, ".tif": true, ".tiff": true,
	".ico": true,
}

// HasBinary checks if the file content appears to be binary based on content sniff and extension.
func HasBinary(content []byte, filename string) bool {
	// Check by extension first for common binary types
	ext := strings.ToLower(filepath.Ext(filename))
	if binaryExtensions[ext] {
		return true
	}

	// Sniff the content for binary data (null bytes or high density of non-printable chars)
	// Only check the first 512 bytes for performance
	sampleSize := len(content)
	if sampleSize > 512 {
		sampleSize = 512
	}

	nonPrintableCount := 0
	for i := 0; i < sampleSize; i++ {
		// Null byte indicates binary
		if content[i] == 0 {
			return true
		}
		// Count non-printable ASCII characters (excluding common whitespace)
		if content[i] < 32 && content[i] != 9 && content[i] != 10 && content[i] != 13 {
			nonPrintableCount++
		}
	}
	// If more than 10% of the sample are non-printable, consider it binary
	if sampleSize > 0 && float64(nonPrintableCount)/float64(sampleSize) > 0.1 {
		return true
	}

	return false
}

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.


type Resolver struct {
	DB             db.DBTX
	StateDir       string
	projectService *project.Service // Added projectService
}

func NewResolver(db db.DBTX, stateDir string, projectService *project.Service) *Resolver {
	return &Resolver{
		DB:             db,
		StateDir:       stateDir,
		projectService: projectService, // Initialize projectService
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
