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

// CompileResult represents the outcome of a LaTeX compilation process.
type CompileResult struct {
	// PDF contains the base64-encoded content of the generated PDF file.
	PDF string `json:"pdf"`
	// Logs contains the combined stdout and stderr output from the latexmk command.
	Logs string `json:"logs"`
	// Synctex contains the base64-encoded content of the generated synctex file (if any).
	Synctex string `json:"synctex"`
	// Success indicates whether the compilation process completed without errors.
	Success bool `json:"success"`
}

// Compile compiles a specific LaTeX file from a project into a PDF using latexmk.
//
// Ideally, this function orchestrates the compilation process in an isolated environment to ensure
// reproducibility and prevent side effects on the source project.
//
// The process involves:
// 1. Creating a unique, temporary directory for the compilation job.
// 2. Retrieving all files from the project via ProjectService.
// 3. Copying these files into the temporary directory, preserving the directory structure.
// 4. Creating a dedicated cache directory for LaTeX auxiliary files.
// 5. Executing `latexmk` with interaction disabled (batchmode) to generate the PDF.
// 6. capturing the generated PDF, logs, and Synctex file.
// 7. Cleaning up the temporary directory.
//
// Parameters:
//   - projectService: Service to access project files.
//   - projectId: The ID of the project containing the file to compile.
//   - filePath: The relative path of the LaTeX file to compile within the project (e.g., "main.tex").
//
// Returns:
//   - *CompileResult: A struct containing the compilation artifacts (PDF, logs, Synctex) and status.
//   - error: An error if the compilation setup fails (e.g., missing latexmk, file system errors).
//     Note that compilation errors (latexmk failure) are reported via the Success field and Logs, not as a returned error.
//
// Dependencies:
//   - Requires `latexmk` to be installed and available in the system PATH.
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

		// Write file content to temporary directory
		outFile, err := os.Create(targetPath)
		if err != nil {
			fileReader.Close()
			return nil, fmt.Errorf("failed to create temporary file %s: %w", targetPath, err)
		}

		_, copyErr := io.Copy(outFile, fileReader)

		// Close explicitly instead of using defer in loop
		fileReader.Close()
		outFile.Close()

		if copyErr != nil {
			return nil, fmt.Errorf("failed to copy file %s to temporary directory: %w", file.Path, copyErr)
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
