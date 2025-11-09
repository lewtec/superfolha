package server

import (
	"database/sql"
	"path/filepath"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB       *sql.DB
	StateDir string
}

func NewResolver(db *sql.DB, stateDir string) *Resolver {
	return &Resolver{
		DB:       db,
		StateDir: stateDir,
	}
}

func (r *Resolver) getProjectPath(projectID string) string {
	return filepath.Join(r.StateDir, "repos", projectID)
}
