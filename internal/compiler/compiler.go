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
	"github.com/lewtec/superfolha/internal/project"
)

type CompileResult struct {
	PDF     string `json:"pdf"`
	Logs    string `json:"logs"`
	Synctex string `json:"synctex"`
	Success bool   `json:"success"`
}

// Compile compiles a specific LaTeX file from a project
func Compile(projectService *project.Service, projectId string, filePath string) (*CompileResult, error) {
	// Check if latexmk command exists
	_, err := exec.LookPath("latexmk")
	if err != nil {
		return nil, fmt.Errorf("latexmk command not found: %w", err)
	}

	// Create temporary directory for compilation
	compileUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate compile ID: %w", err)
	}
	compileID := compileUUID.String()
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("compile-%s", compileID))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir) // Cleanup

	// Get all files from the project
	projectFiles, err := projectService.ListFiles(projectId)
	if err != nil {
		return nil, fmt.Errorf("failed to list project files: %w", err)
	}

	// Copy project files to the temporary directory
	for _, file := range projectFiles {
		// Ensure directory structure is maintained
		targetPath := filepath.Join(tmpDir, file.Path)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for file %s: %w", file.Path, err)
		}

		// Read file content
		fileReader, _, err := projectService.ReadFile(projectId, file.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s from project: %w", file.Path, err)
		}
		// Close the reader after copying
		// Note: defer in a loop can be problematic if many files, but for typical LaTeX projects, it should be fine.
		// For very large projects, consider closing immediately after io.Copy.
		defer fileReader.Close()

		// Write file content to temporary directory
		outFile, err := os.Create(targetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary file %s: %w", targetPath, err)
		}
		defer outFile.Close() // Close the writer

		if _, err := io.Copy(outFile, fileReader); err != nil {
			return nil, fmt.Errorf("failed to copy file %s to temporary directory: %w", file.Path, err)
		}
	}

	// Create a cache directory for latexmk aux files
	latexCacheDir := filepath.Join(tmpDir, "latex-cache")
	if err := os.MkdirAll(latexCacheDir, 0755); err != nil {
		return nil, err
	}

	// Compile with latexmk
	// The main file to compile is now directly filePath
	cmd := exec.Command("latexmk", "-f", "-interaction=batchmode", fmt.Sprintf("-aux-directory=%s", latexCacheDir), "-pdf", filePath)
	cmd.Dir = tmpDir // Run latexmk in the temporary project directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	logs := stdout.String() + stderr.String()

	// Check if PDF was generated
	pdfPath := filepath.Join(tmpDir, strings.TrimSuffix(filePath, ".tex")+".pdf")
	_, pdfErr := os.Stat(pdfPath) // Check if file exists

	success := err == nil && pdfErr == nil

	result := &CompileResult{
		Logs:    logs,
		Success: success,
	}

	// Read PDF if successful
	if pdfErr == nil { // Only attempt to read if the file exists
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
