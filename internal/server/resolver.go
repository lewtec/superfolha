package server

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/git"
)

//go:embed templates
var templatesFS embed.FS

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
	absPath, err := filepath.Abs(filepath.Join(r.StateDir, "repos", projectID))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get absolute path: %v\n", err)
		return filepath.Join(r.StateDir, "repos", projectID)
	}
	return absPath
}

func (r *Resolver) getAndCheckProject(ctx context.Context, projectID string) (*db.Project, string, *auth.UserContext, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, "", nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if !bytes.Equal(project.UserID.Bytes[:], userUUID[:]) {
		return nil, "", nil, fmt.Errorf("not authorized")
	}

	projectPath := r.getProjectPath(projectID)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		if err := git.InitRepo(projectPath); err != nil {
			return nil, "", nil, fmt.Errorf("failed to re-init git repo: %w", err)
		}
	}

	return &project, projectPath, user, nil
}
