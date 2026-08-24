package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/session"
)

type Resolver struct {
	Repo           db.Repository
	StateDir       string
	projectService *project.Service
	authService    *auth.Service
	hubs           *session.Registry
}

func NewResolver(repo db.Repository, stateDir string, projectService *project.Service, authService *auth.Service, hubs *session.Registry) *Resolver {
	return &Resolver{
		Repo:           repo,
		StateDir:       stateDir,
		projectService: projectService,
		authService:    authService,
		hubs:           hubs,
	}
}

func (r *Resolver) getAndCheckProject(ctx context.Context, projectID string) (*db.Project, string, *auth.UserContext, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, "", nil, apierrors.WithStatus(apierrors.CodeUnauthenticated, "not authenticated", 401)
	}

	if _, err := uuid.Parse(projectID); err != nil {
		return nil, "", nil, apierrors.New(apierrors.CodeInvalidInput, "invalid project ID")
	}

	info, live := r.hubs.Live(projectID)
	if !live {
		return nil, "", nil, apierrors.New(apierrors.CodeProjectNotFound, "session not found")
	}
	if !r.hubs.CanOpen(projectID, user.Email) {
		return nil, "", nil, apierrors.WithStatus(apierrors.CodeUnauthorized, "not authorized", 403)
	}
	projectPath := r.projectService.GetProjectPath(projectID)
	proj := db.Project{
		ID:      info.ID,
		UserID:  info.HostLogin,
		Name:    info.Remote,
		GitPath: projectPath,
	}
	return &proj, projectPath, user, nil
}
