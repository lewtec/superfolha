package compiler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/project"
)

// DefaultCompileTimeout is the maximum time a single latexmk run may take.
// Client disconnect cancels earlier via the request context.
const DefaultCompileTimeout = 3 * time.Minute

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
// The process involves:
// 1. Creating a unique, temporary directory for the compilation job.
// 2. Retrieving all files from the project via ProjectService.
// 3. Copying these files into the temporary directory, preserving the directory structure.
// 4. Creating a dedicated cache directory for LaTeX auxiliary files.
// 5. Executing `latexmk` with interaction disabled (batchmode) to generate the PDF.
// 6. Capturing the generated PDF, logs, and Synctex file.
// 7. Cleaning up the temporary directory (also on cancel/timeout via defer).
//
// ctx is bounded by DefaultCompileTimeout (whichever comes first if ctx already has a deadline).
// Canceling ctx kills the latexmk process via CommandContext.
//
// Returns:
//   - *CompileResult on setup success; latexmk non-zero exit is Success=false with Logs, not an error.
//   - error for setup failures, missing latexmk, or context cancel/deadline during latexmk.
//
// Requires latexmk on PATH.
func Compile(ctx context.Context, projectService *project.Service, projectId string, filePath string) (*CompileResult, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultCompileTimeout)
	defer cancel()

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
	defer os.RemoveAll(tmpDir) // Cleanup (runs on cancel/timeout too)

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

	// Compile with latexmk; cancel/timeout kills the process
	cmd := exec.CommandContext(ctx, "latexmk", "-f", "-interaction=batchmode", fmt.Sprintf("-aux-directory=%s", latexCacheDir), "-pdf", filePath)
	cmd.Dir = tmpDir // Run latexmk in the temporary project directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	logs := stdout.String() + stderr.String()

	// Context cancel/deadline is a hard failure (not a latexmk compile error)
	if err != nil && ctx.Err() != nil {
		return nil, fmt.Errorf("latex compile timed out or was canceled (limit %s): %w", DefaultCompileTimeout, ctx.Err())
	}

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
