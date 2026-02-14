package server

import (
	"log"
	"net/http"
	"strings"

	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project" // Import project package
)

// MaxGraphQLFileSize defines the maximum file size (in bytes) for content to be returned directly via GraphQL.
const MaxGraphQLFileSize = 1024 * 1024 * 5 // 5 MB

// HasBinary checks if the file content appears to be binary using http.DetectContentType.
func HasBinary(content []byte, filename string) bool {
	// http.DetectContentType sniffs the first 512 bytes.
	// If the content is shorter, it sniffs the entire content.
	contentType := http.DetectContentType(content)

	// Log the detected content type for debugging
	log.Printf("HasBinary: File %s detected content type: %s", filename, contentType)

	// If the content type starts with "text/", it's considered text.
	// Otherwise, it's considered binary.
	// application/octet-stream is the default for unknown binary types.
	if strings.HasPrefix(contentType, "text/") {
		return false
	}

	return true
}

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB             db.DBTX
	StateDir       string
	projectService *project.Service // Added projectService
	authService    *auth.Service
}

func NewResolver(db db.DBTX, stateDir string, projectService *project.Service, authService *auth.Service) *Resolver {
	return &Resolver{
		DB:             db,
		StateDir:       stateDir,
		projectService: projectService, // Initialize projectService
		authService:    authService,
	}
}

func (r *Resolver) getProjectPath(projectID string) string {
	return r.projectService.GetProjectPath(projectID)
}
