package project

import (
	"fmt"
	"path/filepath"

	igit "github.com/lewtec/superfolha/internal/git" // Reusing existing git types and functions
)

// Service provides project-related operations, encapsulating Git interactions.
type Service struct {
	repoManager *RepositoryManager
}

// NewService creates a new ProjectService.
func NewService(repoManager *RepositoryManager) *Service {
	return &Service{
		repoManager: repoManager,
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
		// If repo doesn't exist, InitRepo will create it
		if err.Error() == fmt.Sprintf("failed to open git repository at %s: repository does not exist", s.GetProjectPath(projectId)) {
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
func (s *Service) ReadFile(projectId, filePath string) (string, error) {
	pr, err := s.repoManager.GetRepo(projectId)
	if err != nil {
		return "", err
	}
	return pr.ReadFile(filePath)
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
