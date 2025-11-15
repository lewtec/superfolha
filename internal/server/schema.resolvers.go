package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
)

// Register is the resolver for the register field.
func (r *mutationResolver) Register(ctx context.Context, email string, password string) (*AuthPayload, error) {
	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// TODO: Insert user into database
	userID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	_ = hashedPassword

	// Generate token
	token, err := auth.GenerateToken(userID.String(), email)
	if err != nil {
		return nil, err
	}

	return &AuthPayload{
		Token: token,
		User: &User{
			ID:    userID.String(),
			Email: email,
		},
	}, nil
}

// Login is the resolver for the login field.
func (r *mutationResolver) Login(ctx context.Context, email string, password string) (*AuthPayload, error) {
	panic(fmt.Errorf("not implemented: Login - login"))
}

// CreateProject is the resolver for the createProject field.
func (r *mutationResolver) CreateProject(ctx context.Context, name string) (*Project, error) {
	panic(fmt.Errorf("not implemented: CreateProject - createProject"))
}

// DeleteProject is the resolver for the deleteProject field.
func (r *mutationResolver) DeleteProject(ctx context.Context, id string) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteProject - deleteProject"))
}

// SaveFile is the resolver for the saveFile field.
func (r *mutationResolver) SaveFile(ctx context.Context, projectID string, path string, content string) (*File, error) {
	panic(fmt.Errorf("not implemented: SaveFile - saveFile"))
}

// DeleteFile is the resolver for the deleteFile field.
func (r *mutationResolver) DeleteFile(ctx context.Context, projectID string, path string) (bool, error) {
	panic(fmt.Errorf("not implemented: DeleteFile - deleteFile"))
}

// Commit is the resolver for the commit field.
func (r *mutationResolver) Commit(ctx context.Context, projectID string, message string) (*Commit, error) {
	panic(fmt.Errorf("not implemented: Commit - commit"))
}

// Me is the resolver for the me field.
func (r *queryResolver) Me(ctx context.Context) (*User, error) {
	panic(fmt.Errorf("not implemented: Me - me"))
}

// Projects is the resolver for the projects field.
func (r *queryResolver) Projects(ctx context.Context) ([]*Project, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	q := db.New(r.DB)
	dbProjects, err := q.GetUserProjects(ctx, pgUserID)
	if err != nil {
		return nil, err
	}

	projects := make([]*Project, len(dbProjects))
	for i, p := range dbProjects {
		projects[i] = &Project{
			ID:        fmt.Sprintf("%x", p.ID.Bytes),
			Name:      p.Name,
			CreatedAt: p.CreatedAt.Time,
			UpdatedAt: p.UpdatedAt.Time,
		}
	}

	return projects, nil
}

// Project is the resolver for the project field.
func (r *queryResolver) Project(ctx context.Context, id string) (*Project, error) {
	panic(fmt.Errorf("not implemented: Project - project"))
}

// Files is the resolver for the files field.
func (r *queryResolver) Files(ctx context.Context, projectID string) ([]*File, error) {
	panic(fmt.Errorf("not implemented: Files - files"))
}

// File is the resolver for the file field.
func (r *queryResolver) File(ctx context.Context, projectID string, path string) (*File, error) {
	panic(fmt.Errorf("not implemented: File - file"))
}

// History is the resolver for the history field.
func (r *queryResolver) History(ctx context.Context, projectID string) ([]*Commit, error) {
	panic(fmt.Errorf("not implemented: History - history"))
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }