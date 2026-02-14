package compiler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/git"
)

type CompileResult struct {
	PDF     string `json:"pdf"`
	Logs    string `json:"logs"`
	Synctex string `json:"synctex"`
	Success bool   `json:"success"`
}

// ProjectReader defines the interface for reading project files.
// This allows decoupling from the concrete ProjectService implementation.
type ProjectReader interface {
	ListFiles(projectId string) ([]*git.FileInfo, error)
	ReadFile(projectId, filePath string) (io.ReadCloser, int64, error)
}

// Compile compiles a specific LaTeX file from a project
func Compile(projectReader ProjectReader, projectId string, filePath string) (*CompileResult, error) {
	// Check if latexmk command exists
	_, err := exec.LookPath("latexmk")
	if err != nil {
		return nil, fmt.Errorf("latexmk command not found: %w", err)
	}

	// Prepare compilation directory
	tmpDir, err := prepareCompilationDir(projectReader, projectId)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir) // Cleanup

	// Create a cache directory for latexmk aux files
	latexCacheDir := filepath.Join(tmpDir, "latex-cache")
	if err := os.MkdirAll(latexCacheDir, 0755); err != nil {
		return nil, err
	}

	// Compile with latexmk
	logs, cmdErr := runLatexmk(tmpDir, filePath, latexCacheDir)

	return collectCompilationResults(tmpDir, filePath, logs, cmdErr)
}

func prepareCompilationDir(projectReader ProjectReader, projectId string) (string, error) {
	// Create temporary directory for compilation
	compileUUID, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate compile ID: %w", err)
	}
	compileID := compileUUID.String()
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("compile-%s", compileID))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	// Get all files from the project
	projectFiles, err := projectReader.ListFiles(projectId)
	if err != nil {
		os.RemoveAll(tmpDir) // Clean up on failure
		return "", fmt.Errorf("failed to list project files: %w", err)
	}

	// Copy project files to the temporary directory
	for _, file := range projectFiles {
		if err := copyFileToDir(projectReader, projectId, file, tmpDir); err != nil {
			os.RemoveAll(tmpDir) // Clean up on failure
			return "", err
		}
	}
	return tmpDir, nil
}

func copyFileToDir(projectReader ProjectReader, projectId string, file *git.FileInfo, tmpDir string) error {
	// Ensure directory structure is maintained
	targetPath := filepath.Join(tmpDir, file.Path)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for file %s: %w", file.Path, err)
	}

	// Read file content
	fileReader, _, err := projectReader.ReadFile(projectId, file.Path)
	if err != nil {
		return fmt.Errorf("failed to read file %s from project: %w", file.Path, err)
	}
	defer fileReader.Close() // Safe here because this function handles one file

	// Write file content to temporary directory
	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary file %s: %w", targetPath, err)
	}
	defer outFile.Close() // Safe here

	if _, err := io.Copy(outFile, fileReader); err != nil {
		return fmt.Errorf("failed to copy file %s to temporary directory: %w", file.Path, err)
	}

	return nil
}

func runLatexmk(tmpDir, filePath, latexCacheDir string) (string, error) {
	// The main file to compile is now directly filePath
	cmd := exec.Command("latexmk", "-f", "-interaction=batchmode", fmt.Sprintf("-aux-directory=%s", latexCacheDir), "-pdf", filePath)
	cmd.Dir = tmpDir // Run latexmk in the temporary project directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	logs := stdout.String() + stderr.String()

	return logs, err
}

func collectCompilationResults(tmpDir, filePath, logs string, cmdErr error) (*CompileResult, error) {
	// Check if PDF was generated
	pdfPath := filepath.Join(tmpDir, strings.TrimSuffix(filePath, ".tex")+".pdf")
	_, pdfErr := os.Stat(pdfPath) // Check if file exists

	// Success means command succeeded AND PDF exists
	success := cmdErr == nil && pdfErr == nil

	result := &CompileResult{
		Logs:    logs,
		Success: success,
	}

	// Read PDF if it exists (regardless of success status)
	if pdfErr == nil {
		if pdfData, err := os.ReadFile(pdfPath); err == nil {
			result.PDF = base64.StdEncoding.EncodeToString(pdfData)
		}
	}

	// Read synctex if exists
	synctexPath := filepath.Join(tmpDir, strings.TrimSuffix(filePath, ".tex")+".synctex.gz")
	if synctexData, err := os.ReadFile(synctexPath); err == nil {
		result.Synctex = base64.StdEncoding.EncodeToString(synctexData)
	}

	return result, nil
}

// ToJSON converts the result to JSON
func (r *CompileResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
