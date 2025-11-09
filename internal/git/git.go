package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Commit struct {
	Hash    string
	Message string
	Author  string
	Date    time.Time
}

type FileInfo struct {
	Path    string
	Content string
}

// InitRepo creates a new git repository
func InitRepo(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = path
	return cmd.Run()
}

// AddAll stages all changes
func AddAll(repoPath string) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	return cmd.Run()
}

// Commit creates a new commit
func CommitChanges(repoPath, author, message string) (*Commit, error) {
	// Configure git user for this repo if not set
	exec.Command("git", "config", "user.name", author).Dir = repoPath
	exec.Command("git", "config", "user.email", author).Dir = repoPath

	// Add all changes
	if err := AddAll(repoPath); err != nil {
		return nil, err
	}

	// Create commit
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Get the commit hash
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashCmd.Dir = repoPath
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return nil, err
	}

	return &Commit{
		Hash:    strings.TrimSpace(string(hashOutput)),
		Message: message,
		Author:  author,
		Date:    time.Now(),
	}, nil
}

// GetHistory returns commit history
func GetHistory(repoPath string) ([]*Commit, error) {
	cmd := exec.Command("git", "log", "--pretty=format:%H|%s|%an|%at", "-n", "50")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return []*Commit{}, nil // Return empty if no commits yet
	}

	lines := strings.Split(string(output), "\n")
	commits := make([]*Commit, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}

		var timestamp int64
		fmt.Sscanf(parts[3], "%d", &timestamp)

		commits = append(commits, &Commit{
			Hash:    parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    time.Unix(timestamp, 0),
		})
	}

	return commits, nil
}

// ReadFile reads a file from the repository
func ReadFile(repoPath, filePath string) (string, error) {
	fullPath := filepath.Join(repoPath, filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes a file to the repository
func WriteFile(repoPath, filePath, content string) error {
	fullPath := filepath.Join(repoPath, filePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}

// DeleteFile removes a file from the repository
func DeleteFile(repoPath, filePath string) error {
	fullPath := filepath.Join(repoPath, filePath)
	return os.Remove(fullPath)
}

// ListFiles lists all files in the repository (excluding .git)
func ListFiles(repoPath string) ([]*FileInfo, error) {
	var files []*FileInfo

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		files = append(files, &FileInfo{
			Path:    relPath,
			Content: string(content),
		})

		return nil
	})

	return files, err
}
