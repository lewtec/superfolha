package project

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lewtec/superfolha/internal/db"
	igit "github.com/lewtec/superfolha/internal/git"
)

//go:embed templates
var templatesFS embed.FS

// ErrFileNotFound is returned when a requested file does not exist in the project.
var ErrFileNotFound = errors.New("file not found")

// Service provides project-related operations, encapsulating Git interactions and Database operations.
type Service struct {
	repoManager *RepositoryManager
	db          db.DBTX
}

// NewService creates a new ProjectService.
func NewService(db db.DBTX, stateDir string) *Service {
	return &Service{
		repoManager: NewRepositoryManager(stateDir),
		db:          db,
	}
}

// GetProjectPath returns the absolute path to a project's Git repository.
func (s *Service) GetProjectPath(projectId string) string {
	return filepath.Join(s.repoManager.stateDir, "repos", projectId)
}

// InitProjectRepo initializes a new Git repository for a project.
func (s *Service) InitProjectRepo(projectId string) error {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return pr.InitRepo()
		}
		return err
	}
	return pr.InitRepo()
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
	decodedPath := filepath.FromSlash(encodedPath)
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

// CreateProject creates a new project in the DB and initializes its Git repository with templates.
func (s *Service) CreateProject(ctx context.Context, userID uuid.UUID, name string, userEmail string) (*db.Project, error) {
	projectUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate project ID: %w", err)
	}
	projectID := projectUUID.String()
	projectPath := s.GetProjectPath(projectID)

	q := db.New(s.db)
	dbProject, err := q.CreateProject(ctx, db.CreateProjectParams{
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
		Name:    name,
		GitPath: projectPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create project in db: %w", err)
	}

	if err := s.InitProjectRepo(projectID); err != nil {
		return nil, fmt.Errorf("failed to init git repo: %w", err)
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
		return nil, fmt.Errorf("failed to copy template files: %w", err)
	}

	_, err = s.CommitChanges(projectID, userEmail, "Initial commit")
	if err != nil {
		return nil, fmt.Errorf("failed to create initial commit: %w", err)
	}

	return &dbProject, nil
}

// DeleteProject deletes a project from the DB and removes its Git repository.
func (s *Service) DeleteProject(ctx context.Context, projectID uuid.UUID) error {
	q := db.New(s.db)
	proj, err := q.GetProject(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		return err
	}

	if err := q.DeleteProject(ctx, proj.ID); err != nil {
		return fmt.Errorf("failed to delete project from db: %w", err)
	}

	// Delete git repository
	repoPath := s.GetProjectPath(projectID.String())
	if err := os.RemoveAll(repoPath); err != nil {
		return fmt.Errorf("failed to delete git repository: %w", err)
	}

	return nil
}

// GetProject retrieves a project from the DB, verifies ownership, and ensures the Git repository exists.
func (s *Service) GetProject(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (*db.Project, error) {
	q := db.New(s.db)
	project, err := q.GetProject(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	if !bytes.Equal(project.UserID.Bytes[:], userID[:]) {
		return nil, fmt.Errorf("not authorized")
	}

	projectPath := s.GetProjectPath(projectID.String())

	// Lazy initialization: Check if repo exists, if not, recover it.
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		user, err := q.GetUserByID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user for repo recovery: %w", err)
		}

		if err := s.InitProjectRepo(projectID.String()); err != nil {
			return nil, fmt.Errorf("failed to init git repo: %w", err)
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
			if err := s.SaveFile(projectID.String(), relativePath, string(content)); err != nil {
				return fmt.Errorf("failed to write template file %s: %w", relativePath, err)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to copy template files: %w", err)
		}

		_, err = s.CommitChanges(projectID.String(), user.Email, "Initial commit (recovery)")
		if err != nil {
			return nil, fmt.Errorf("failed to create initial commit: %w", err)
		}
	}

	return &project, nil
}

// ListProjects lists all projects for a user.
func (s *Service) ListProjects(ctx context.Context, userID uuid.UUID) ([]db.Project, error) {
	q := db.New(s.db)
	return q.GetUserProjects(ctx, pgtype.UUID{Bytes: userID, Valid: true})
}

// TouchProject updates the project's updated_at timestamp.
func (s *Service) TouchProject(ctx context.Context, projectID uuid.UUID) error {
	q := db.New(s.db)
	return q.UpdateProjectTimestamp(ctx, pgtype.UUID{Bytes: projectID, Valid: true})
}
