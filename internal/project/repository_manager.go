package project

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-git/go-git/v5"
	igit "github.com/lewtec/superfolha/internal/git"
)

// ProjectRepository wraps a git.Repository and provides thread-safe operations.
type ProjectRepository struct {
	projectId  string
	repoPath   string
	repo       *git.Repository
	sync.Mutex // Protects access to the underlying git.Repository
}

// RepositoryManager manages a collection of ProjectRepository instances.
type RepositoryManager struct {
	stateDir string
	repos    map[string]*ProjectRepository
	mu       sync.RWMutex // Protects access to the repos map
}

// NewRepositoryManager creates a new RepositoryManager.
func NewRepositoryManager(stateDir string) *RepositoryManager {
	return &RepositoryManager{
		stateDir: stateDir,
		repos:    make(map[string]*ProjectRepository),
	}
}

// GetRepo retrieves or initializes a ProjectRepository for the given projectId.
func (rm *RepositoryManager) GetRepo(projectId string) (*ProjectRepository, error) {
	rm.mu.RLock()
	pr, ok := rm.repos[projectId]
	rm.mu.RUnlock()

	if ok {
		return pr, nil
	}

	// Repository not found, create a new one
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Double-check if it was created by another goroutine while we were waiting for the lock
	pr, ok = rm.repos[projectId]
	if ok {
		return pr, nil
	}

	repoPath := filepath.Join(rm.stateDir, "repos", projectId)
	
	// Always create the ProjectRepository instance
	pr = &ProjectRepository{
		projectId: projectId,
		repoPath:  repoPath,
	}

	// Try to open the git repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		// If the error is "repository does not exist", we still return the pr,
		// but with pr.repo = nil, and the error.
		// Other errors are critical.
		if err == git.ErrRepositoryNotExists {
			rm.repos[projectId] = pr // Add to map even if repo doesn't exist yet
			return pr, err // Return pr and the specific error
		}
		return nil, fmt.Errorf("failed to open git repository at %s: %w", repoPath, err)
	}

	pr.repo = r // Assign the opened repo
	rm.repos[projectId] = pr

	return pr, nil
}

// InitRepo initializes a new Git repository for this project.
func (pr *ProjectRepository) InitRepo() error {
	pr.Lock()
	defer pr.Unlock()

	return igit.InitRepo(pr.repoPath)
}

// AddAll stages all changes in the repository.
func (pr *ProjectRepository) AddAll() error {
	pr.Lock()
	defer pr.Unlock()

	return igit.AddAll(pr.repoPath)
}

// CommitChanges commits all staged changes in the repository.
func (pr *ProjectRepository) CommitChanges(author, message string) (*igit.Commit, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.CommitChanges(pr.repoPath, author, message)
}

// SaveFile writes content to a file within the repository.
func (pr *ProjectRepository) SaveFile(filePath, content string) error {
	pr.Lock()
	defer pr.Unlock()

	fullPath := filepath.Join(pr.repoPath, filePath)

	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// DeleteFile removes a file from the repository.
func (pr *ProjectRepository) DeleteFile(filePath string) error {
	pr.Lock()
	defer pr.Unlock()

	fullPath := filepath.Join(pr.repoPath, filePath)
	return os.Remove(fullPath)
}

// ReadFile reads a file from the repository.
func (pr *ProjectRepository) ReadFile(filePath string) (io.ReadCloser, int64, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.ReadFile(pr.repoPath, filePath)
}

// ListFiles lists all files in the repository.
func (pr *ProjectRepository) ListFiles() ([]*igit.FileInfo, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.ListFiles(pr.repoPath)
}

// GetHistory returns the commit history for the repository.
func (pr *ProjectRepository) GetHistory() ([]*igit.Commit, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.GetHistory(pr.repoPath)
}
