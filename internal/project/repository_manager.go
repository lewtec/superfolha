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

// Repository wraps a git.Repository and provides thread-safe operations.
type Repository struct {
	projectID  string
	repoPath   string
	repo       *git.Repository
	sync.Mutex // Protects access to the underlying git.Repository
}

// RepositoryManager manages a collection of Repository instances.
type RepositoryManager struct {
	stateDir string
	repos    map[string]*Repository
	mu       sync.RWMutex // Protects access to the repos map
}

// NewRepositoryManager creates a new RepositoryManager.
func NewRepositoryManager(stateDir string) *RepositoryManager {
	return &RepositoryManager{
		stateDir: stateDir,
		repos:    make(map[string]*Repository),
	}
}

// GetRepo retrieves or initializes a Repository for the given projectID.
func (rm *RepositoryManager) GetRepo(projectID string) (*Repository, error) {
	rm.mu.RLock()
	pr, ok := rm.repos[projectID]
	rm.mu.RUnlock()

	if ok {
		return pr, nil
	}

	// Repository not found, create a new one
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Double-check if it was created by another goroutine while we were waiting for the lock
	pr, ok = rm.repos[projectID]
	if ok {
		return pr, nil
	}

	repoPath := filepath.Join(rm.stateDir, "repos", projectID)

	// Always create the Repository instance
	pr = &Repository{
		projectID: projectID,
		repoPath:  repoPath,
	}

	// Try to open the git repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		// If the error is "repository does not exist", we still return the pr,
		// but with pr.repo = nil, and the error.
		// Other errors are critical.
		if err == git.ErrRepositoryNotExists {
			rm.repos[projectID] = pr // Add to map even if repo doesn't exist yet
			return pr, err           // Return pr and the specific error
		}
		return nil, fmt.Errorf("failed to open git repository at %s: %w", repoPath, err)
	}

	pr.repo = r // Assign the opened repo
	rm.repos[projectID] = pr

	return pr, nil
}

// InitRepo initializes a new Git repository for this project.
func (pr *Repository) InitRepo() error {
	pr.Lock()
	defer pr.Unlock()

	return igit.InitRepo(pr.repoPath)
}

// AddAll stages all changes in the repository.
func (pr *Repository) AddAll() error {
	pr.Lock()
	defer pr.Unlock()

	return igit.AddAll(pr.repoPath)
}

// CommitChanges commits all staged changes in the repository.
func (pr *Repository) CommitChanges(author, message string) (*igit.Commit, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.CommitChanges(pr.repoPath, author, message)
}

// SaveFile writes content to a file within the repository.
func (pr *Repository) SaveFile(filePath, content string) error {
	pr.Lock()
	defer pr.Unlock()

	fullPath := filepath.Join(pr.repoPath, filePath)

	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

// DeleteFile removes a file from the repository.
func (pr *Repository) DeleteFile(filePath string) error {
	pr.Lock()
	defer pr.Unlock()

	fullPath := filepath.Join(pr.repoPath, filePath)
	return os.Remove(fullPath)
}

// ReadFile reads a file from the repository.
func (pr *Repository) ReadFile(filePath string) (io.ReadCloser, int64, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.ReadFile(pr.repoPath, filePath)
}

// ListFiles lists all files in the repository.
func (pr *Repository) ListFiles() ([]*igit.FileInfo, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.ListFiles(pr.repoPath)
}

// GetHistory returns the commit history for the repository.
func (pr *Repository) GetHistory() ([]*igit.Commit, error) {
	pr.Lock()
	defer pr.Unlock()

	return igit.GetHistory(pr.repoPath)
}
