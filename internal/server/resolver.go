package server

import (
	"path/filepath"
	"github.com/lewtec/superfolha/internal/db"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB       db.DBTX
	StateDir string
}

func NewResolver(db db.DBTX, stateDir string) *Resolver {
	return &Resolver{
		DB:       db,
		StateDir: stateDir,
	}
}

func (r *Resolver) getProjectPath(projectID string) string {
	return filepath.Join(r.StateDir, "repos", projectID)
}
