package compiler

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type CompileResult struct {
	PDF     string `json:"pdf"`
	Logs    string `json:"logs"`
	Synctex string `json:"synctex"`
	Success bool   `json:"success"`
}

// ExtractTarball extracts a .tar.gz file to a directory
func extractTarball(tarballPath, destPath string) error {
	file, err := os.Open(tarballPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// FindMainTexFile finds the main .tex file
func findMainTexFile(dir string) (string, error) {
	// First, look for main.tex
	mainTex := filepath.Join(dir, "main.tex")
	if _, err := os.Stat(mainTex); err == nil {
		return "main.tex", nil
	}

	// Otherwise, find any .tex file in the root
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".tex") {
			return file.Name(), nil
		}
	}

	return "", fmt.Errorf("no .tex file found")
}

// Compile compiles a LaTeX project from a tarball
func Compile(tarballData []byte) (*CompileResult, error) {
	// Create temporary directory
	compileID := uuid.New().String()
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("compile-%s", compileID))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir) // Cleanup

	// Save tarball
	tarballPath := filepath.Join(tmpDir, "project.tar.gz")
	if err := os.WriteFile(tarballPath, tarballData, 0644); err != nil {
		return nil, err
	}

	// Extract tarball
	extractDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, err
	}
	if err := extractTarball(tarballPath, extractDir); err != nil {
		return nil, err
	}

	// Find main .tex file
	mainFile, err := findMainTexFile(extractDir)
	if err != nil {
		return nil, err
	}

	// Compile with pdflatex
	cmd := exec.Command("pdflatex", "-synctex=1", "-interaction=nonstopmode", mainFile)
	cmd.Dir = extractDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	logs := stdout.String() + stderr.String()

	success := err == nil

	result := &CompileResult{
		Logs:    logs,
		Success: success,
	}

	// Read PDF if successful
	pdfPath := filepath.Join(extractDir, strings.TrimSuffix(mainFile, ".tex")+".pdf")
	if pdfData, err := os.ReadFile(pdfPath); err == nil {
		result.PDF = base64.StdEncoding.EncodeToString(pdfData)
	}

	// Read synctex if exists
	synctexPath := filepath.Join(extractDir, strings.TrimSuffix(mainFile, ".tex")+".synctex.gz")
	if synctexData, err := os.ReadFile(synctexPath); err == nil {
		result.Synctex = base64.StdEncoding.EncodeToString(synctexData)
	}

	return result, nil
}

// ToJSON converts the result to JSON
func (r *CompileResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
