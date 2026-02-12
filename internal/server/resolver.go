package server

import (
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
)

// MaxGraphQLFileSize defines the maximum file size (in bytes) for content to be returned directly via GraphQL.
const MaxGraphQLFileSize = 1024 * 1024 * 5 // 5 MB

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB             db.DBTX
	StateDir       string
	projectService *project.Service
	authService    *auth.Service
}

func NewResolver(db db.DBTX, stateDir string, projectService *project.Service, authService *auth.Service) *Resolver {
	return &Resolver{
		DB:             db,
		StateDir:       stateDir,
		projectService: projectService,
		authService:    authService,
	}
}

func (r *Resolver) getProjectPath(projectID string) string {
	return r.projectService.GetProjectPath(projectID)
}
