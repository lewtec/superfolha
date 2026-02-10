package project

import (
	"errors"
	"io"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	igit "github.com/lewtec/superfolha/internal/git"
)

// ErrFileNotFound is returned when a requested file does not exist in the project.
var ErrFileNotFound = errors.New("file not found")

// Service provides project-related operations, encapsulating Git interactions.
type Service struct {
	repoManager *RepositoryManager
}

// NewService creates a new Service.
func NewService(stateDir string) *Service {
	return &Service{
		repoManager: NewRepositoryManager(stateDir),
	}
}

// GetProjectPath returns the absolute path to a project's Git repository.
func (s *Service) GetProjectPath(projectID string) string {
	return filepath.Join(s.repoManager.stateDir, "repos", projectID)
}

// InitProjectRepo initializes a new Git repository for a project.
func (s *Service) InitProjectRepo(projectID string) error {
	pr, err := s.repoManager.GetRepo(projectID)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			return pr.InitRepo()
		}
		return err
	}
	return pr.InitRepo()
}

// SaveFile writes content to a file within a project's repository and stages it.
func (s *Service) SaveFile(projectID, filePath, content string) error {
	pr, err := s.repoManager.GetRepo(projectID)
	if err != nil {
		return err
	}
	return pr.SaveFile(filePath, content)
}

// DeleteFile removes a file from a project's repository and stages the deletion.
func (s *Service) DeleteFile(projectID, filePath string) error {
	pr, err := s.repoManager.GetRepo(projectID)
	if err != nil {
		return err
	}
	return pr.DeleteFile(filePath)
}

// CommitChanges commits all staged changes in a project's repository.
func (s *Service) CommitChanges(projectID, author, message string) (*igit.Commit, error) {
	pr, err := s.repoManager.GetRepo(projectID)
	if err != nil {
		return nil, err
	}
	return pr.CommitChanges(author, message)
}

// ReadFile reads a file from a project's repository.
func (s *Service) ReadFile(projectID, filePath string) (io.ReadCloser, int64, error) {
	pr, err := s.repoManager.GetRepo(projectID)
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

// ListFiles lists all files in a project's repository.
func (s *Service) ListFiles(projectID string) ([]*igit.FileInfo, error) {
	pr, err := s.repoManager.GetRepo(projectID)
	if err != nil {
		return nil, err
	}
	return pr.ListFiles()
}

// GetHistory returns the commit history for a project.
func (s *Service) GetHistory(projectID string) ([]*igit.Commit, error) {
	pr, err := s.repoManager.GetRepo(projectID)
	if err != nil {
		return nil, err
	}
	return pr.GetHistory()
}

// DecodeFilePath decodes a URL-encoded file path.
func DecodeFilePath(encodedPath string) (string, error) {
	decodedPath := filepath.FromSlash(encodedPath)
	return decodedPath, nil
}
