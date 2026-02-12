package project

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	igit "github.com/lewtec/superfolha/internal/git"
)

//go:embed templates
var templatesFS embed.FS

// ErrFileNotFound is returned when a requested file does not exist in the project.
var ErrFileNotFound = errors.New("file not found")

// Service provides project-related operations, encapsulating Git interactions.
type Service struct {
	db          db.DBTX
	repoManager *RepositoryManager
}

// NewService creates a new ProjectService.
func NewService(db db.DBTX, stateDir string) *Service {
	return &Service{
		db:          db,
		repoManager: NewRepositoryManager(stateDir),
	}
}

// GetProjectPath returns the absolute path to a project's Git repository.
func (s *Service) GetProjectPath(projectId string) string {
	return filepath.Join(s.repoManager.stateDir, "repos", projectId)
}

// ensureRepoInitialized ensures the repo exists and is initialized with templates if needed.
func (s *Service) ensureRepoInitialized(projectID string, userEmail string) error {
	projectPath := s.GetProjectPath(projectID)

	// Check if repo exists
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		return nil // Repo exists
	}

	// Initialize repo using repoManager (creates dir and runs git init)
	// We call InitRepo on the repo object which handles concurrent access
	pr, err := s.repoManager.GetRepo(projectID)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			if initErr := pr.InitRepo(); initErr != nil {
				return fmt.Errorf("failed to init git repo: %w", initErr)
			}
		} else {
			return fmt.Errorf("failed to get repo: %w", err)
		}
	} else {
		// Repo object exists, maybe verify it's init? usually GetRepo handles this.
		// If GetRepo returned no error, it means it opened existing repo.
		// But here we are inside !os.IsNotExist check, so logic is:
		// if NOT exist -> init.
		// Wait, os.Stat failed with IsNotExist.
		// So we must init.
		if initErr := pr.InitRepo(); initErr != nil {
			return fmt.Errorf("failed to init git repo: %w", initErr)
		}
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
		if err := s.SaveFile(projectID, relativePath, string(content)); err != nil {
			return fmt.Errorf("failed to write template file %s: %w", relativePath, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to copy template files: %w", err)
	}

	// Initial commit
	_, err = s.CommitChanges(projectID, userEmail, "Initial commit")
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	return nil
}

// CreateProject creates a new project in the database and initializes the git repository.
func (s *Service) CreateProject(ctx context.Context, userID uuid.UUID, name string) (*db.Project, error) {
	// Generate project ID
	projectUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate project ID: %w", err)
	}
	projectID := projectUUID.String()
	projectPath := s.GetProjectPath(projectID)

	// Create project in database
	q := db.New(s.db)
	dbProject, err := q.CreateProject(ctx, db.CreateProjectParams{
		ID:      pgtype.UUID{Bytes: projectUUID, Valid: true},
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
		Name:    name,
		GitPath: projectPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create project in db: %w", err)
	}

	// Get user email from context for initial commit
	user, ok := auth.GetUserFromContext(ctx)
	userEmail := "system@superfolha.local"
	if ok {
		userEmail = user.Email
	}

	// Initialize git repo and templates
	if err := s.ensureRepoInitialized(projectID, userEmail); err != nil {
		// Rollback DB creation if git init fails?
		//Ideally yes, but for now just return error. The DB record will point to non-existent repo.
		// A transaction could be used but git ops are outside tx.
		// We could delete the DB record here.
		_ = q.DeleteProject(ctx, dbProject.ID)
		return nil, err
	}

	return &dbProject, nil
}

// DeleteProject deletes a project from the database and removes the git repository.
func (s *Service) DeleteProject(ctx context.Context, projectID string, userID uuid.UUID) error {
	// Verify ownership first
	project, err := s.GetProjectForUser(ctx, projectID, userID)
	if err != nil {
		return err
	}

	// Delete from DB
	q := db.New(s.db)
	if err := q.DeleteProject(ctx, project.ID); err != nil {
		return fmt.Errorf("failed to delete project from db: %w", err)
	}

	// Delete git repository
	projectPath := s.GetProjectPath(projectID)
	if err := os.RemoveAll(projectPath); err != nil {
		return fmt.Errorf("failed to delete git repository: %w", err)
	}

	return nil
}

// GetProject retrieves a project from the database.
func (s *Service) GetProject(ctx context.Context, projectID string) (*db.Project, error) {
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	q := db.New(s.db)
	project, err := q.GetProject(ctx, pgtype.UUID{Bytes: projectUUID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	return &project, nil
}

// GetProjectForUser retrieves a project and verifies that the user owns it.
// It also ensures the git repository is initialized (lazy init).
func (s *Service) GetProjectForUser(ctx context.Context, projectID string, userID uuid.UUID) (*db.Project, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(project.UserID.Bytes[:], userID[:]) {
		return nil, fmt.Errorf("not authorized")
	}

	// Get user email from context for potential lazy init commit
	user, ok := auth.GetUserFromContext(ctx)
	userEmail := "system@superfolha.local"
	if ok {
		userEmail = user.Email
	}

	// Ensure git repo exists (lazy init)
	if err := s.ensureRepoInitialized(projectID, userEmail); err != nil {
		return nil, fmt.Errorf("failed to ensure git repo: %w", err)
	}

	return project, nil
}

// UpdateProjectTimestamp updates the updated_at timestamp of a project.
func (s *Service) UpdateProjectTimestamp(ctx context.Context, projectID string) error {
	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	q := db.New(s.db)
	return q.UpdateProjectTimestamp(ctx, pgtype.UUID{Bytes: projectUUID, Valid: true})
}

// DetectBinary checks if the content is binary.
func (s *Service) DetectBinary(content []byte) bool {
	contentType := http.DetectContentType(content)
	return !strings.HasPrefix(contentType, "text/")
}

// InitProjectRepo initializes a new Git repository for a project (Wrapper for manual init if needed).
// Ideally prefer using CreateProject or GetProjectForUser which handle this automatically.
func (s *Service) InitProjectRepo(projectId string) error {
	// This method kept for compatibility or manual usage, but implementation
	// should align with ensureRepoInitialized logic if possible, or just call it with dummy email.
	return s.ensureRepoInitialized(projectId, "system@superfolha.local")
}

// SaveFile writes content to a file within a project's repository and stages it.
func (s *Service) SaveFile(projectId, filePath, content string) error {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return err
	}
	return pr.SaveFile(filePath, content)
}

// DeleteFile removes a file from a project's repository and stages the deletion.
func (s *Service) DeleteFile(projectId, filePath string) error {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return err
	}
	return pr.DeleteFile(filePath)
}

// CommitChanges commits all staged changes in a project's repository.
func (s *Service) CommitChanges(projectId, author, message string) (*igit.Commit, error) {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return nil, err
	}
	return pr.CommitChanges(author, message)
}

// ReadFile reads a file from a project's repository.
func (s *Service) ReadFile(projectId, filePath string) (io.ReadCloser, int64, error) {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return nil, 0, err
	}
	reader, size, err := pr.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, igit.ErrGitFileNotFound) {
			return nil, 0, ErrFileNotFound
		}
		return nil, 0, err
	}
	return reader, size, nil
}

// DecodeFilePath decodes a URL-encoded file path.
func DecodeFilePath(encodedPath string) (string, error) {
	// filepath.FromSlash returns only one value (the string result), not two.
	// The error handling for filepath.FromSlash is incorrect.
	decodedPath := filepath.FromSlash(encodedPath)
	// No error is returned by filepath.FromSlash, so no error check needed here.
	return decodedPath, nil
}

// ListFiles lists all files in a project's repository.
func (s *Service) ListFiles(projectId string) ([]*igit.FileInfo, error) {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return nil, err
	}
	return pr.ListFiles()
}

// GetHistory returns the commit history for a project.
func (s *Service) GetHistory(projectId string) ([]*igit.Commit, error) {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return nil, err
	}
	return pr.GetHistory()
}
