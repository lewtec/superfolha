package compiler

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/superfolha/internal/git"
)

// MockProjectReader is a mock implementation of ProjectReader
type MockProjectReader struct {
	Files map[string]string
}

func (m *MockProjectReader) ListFiles(projectId string) ([]*git.FileInfo, error) {
	var files []*git.FileInfo
	for path, content := range m.Files {
		files = append(files, &git.FileInfo{
			Path: path,
			Size: int64(len(content)),
		})
	}
	return files, nil
}

func (m *MockProjectReader) ReadFile(projectId, filePath string) (io.ReadCloser, int64, error) {
	content, ok := m.Files[filePath]
	if !ok {
		return nil, 0, errors.New("file not found")
	}
	return io.NopCloser(bytes.NewBufferString(content)), int64(len(content)), nil
}

// MockProjectReaderWithCloseTracking tracks if Close was called
type MockProjectReaderWithCloseTracking struct {
	Files       map[string]string
	ClosedFiles map[string]bool
}

type trackedReadCloser struct {
	io.Reader
	path string
	m    *MockProjectReaderWithCloseTracking
}

func (t *trackedReadCloser) Close() error {
	t.m.ClosedFiles[t.path] = true
	return nil
}

func (m *MockProjectReaderWithCloseTracking) ListFiles(projectId string) ([]*git.FileInfo, error) {
	var files []*git.FileInfo
	for path, content := range m.Files {
		files = append(files, &git.FileInfo{
			Path: path,
			Size: int64(len(content)),
		})
	}
	return files, nil
}

func (m *MockProjectReaderWithCloseTracking) ReadFile(projectId, filePath string) (io.ReadCloser, int64, error) {
	content, ok := m.Files[filePath]
	if !ok {
		return nil, 0, errors.New("file not found")
	}
	return &trackedReadCloser{
		Reader: bytes.NewBufferString(content),
		path:   filePath,
		m:      m,
	}, int64(len(content)), nil
}

func TestPrepareCompilationDir(t *testing.T) {
	mockReader := &MockProjectReader{
		Files: map[string]string{
			"main.tex":       "\\documentclass{article}\\begin{document}Hello\\end{document}",
			"chapter1/a.tex": "Chapter 1",
		},
	}

	tmpDir, err := prepareCompilationDir(mockReader, "test-project")
	if err != nil {
		t.Fatalf("prepareCompilationDir failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Verify main.tex
	content, err := os.ReadFile(filepath.Join(tmpDir, "main.tex"))
	if err != nil {
		t.Errorf("main.tex not found in tmpDir: %v", err)
	}
	if string(content) != mockReader.Files["main.tex"] {
		t.Errorf("main.tex content mismatch")
	}

	// Verify chapter1/a.tex
	content, err = os.ReadFile(filepath.Join(tmpDir, "chapter1", "a.tex"))
	if err != nil {
		t.Errorf("chapter1/a.tex not found in tmpDir: %v", err)
	}
	if string(content) != mockReader.Files["chapter1/a.tex"] {
		t.Errorf("chapter1/a.tex content mismatch")
	}
}

func TestResourceLeakFix(t *testing.T) {
	mockReader := &MockProjectReaderWithCloseTracking{
		Files: map[string]string{
			"test.tex": "content",
		},
		ClosedFiles: make(map[string]bool),
	}

	tmpDir, err := prepareCompilationDir(mockReader, "test-project")
	if err != nil {
		t.Fatalf("prepareCompilationDir failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if !mockReader.ClosedFiles["test.tex"] {
		t.Errorf("File reader for test.tex was not closed")
	}
}
