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

// NewService creates a new ProjectService.
func NewService(stateDir string) *Service {
	return &Service{
		repoManager: NewRepositoryManager(stateDir),
	}
}

// GetProjectPath returns the absolute path to a project's Git repository.
func (s *Service) GetProjectPath(projectId string) string {
	return filepath.Join(s.repoManager.stateDir, "repos", projectId)
}

// InitProjectRepo initializes a new Git repository for a project.
// Safe when missing: GetRepo still returns a usable ProjectRepository with ErrRepositoryNotExists.
func (s *Service) InitProjectRepo(projectId string) error {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return pr.InitRepo()
		}
		return err
	}
	return pr.InitRepo()
}

// SaveFile writes content to a file on disk in a project's repository.
// It does not stage; CommitChanges stages later.
func (s *Service) SaveFile(projectId, filePath, content string) error {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return err
	}
	return pr.SaveFile(filePath, content)
}

// DeleteFile removes a file on disk from a project's repository.
// It does not stage; CommitChanges stages later.
func (s *Service) DeleteFile(projectId, filePath string) error {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return err
	}
	return pr.DeleteFile(filePath)
}

// CommitChanges stages and commits all worktree changes in a project's repository.
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
