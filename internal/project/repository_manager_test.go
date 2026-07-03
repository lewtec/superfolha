package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRepository_SaveFile_Permissions(t *testing.T) {
	// Create a temporary directory for the state dir
	stateDir, err := os.MkdirTemp("", "project-test-statedir-*")
	if err != nil {
		t.Fatalf("failed to create temp state dir: %v", err)
	}
	defer os.RemoveAll(stateDir) // clean up

	// Create a new RepositoryManager
	rm := NewRepositoryManager(stateDir)
	projectId := "test-project-1"

	// Get (and implicitly initialize internal state for) the repository
	pr, err := rm.GetRepo(projectId)
	if err != nil && err.Error() != "repository does not exist" {
		t.Fatalf("failed to get repo: %v", err)
	}

	// We must explicitly init repo
	err = pr.InitRepo()
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	// Define the file path and content
	filePath := "testdoc.tex"
	content := "\\documentclass{article}\n\\begin{document}\nHello\n\\end{document}"

	// Call SaveFile which we are testing
	err = pr.SaveFile(filePath, content)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	// Verify the file was created with the correct permissions
	fullPath := filepath.Join(pr.repoPath, filePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("failed to stat created file: %v", err)
	}

	// The permission should be exactly 0600
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600, got %04o", info.Mode().Perm())
	}
}
