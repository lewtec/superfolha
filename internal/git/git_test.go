package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFile_Permissions(t *testing.T) {
	// Create a temporary repository directory
	tempDir, err := os.MkdirTemp("", "git-test-repo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // clean up

	// Define the file path and content
	filePath := "testfile.txt"
	content := "hello world"

	// Call the function under test
	err = WriteFile(tempDir, filePath, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify the file was created with the correct permissions
	fullPath := filepath.Join(tempDir, filePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("failed to stat created file: %v", err)
	}

	// The permission should be exactly 0600
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600, got %04o", info.Mode().Perm())
	}
}
