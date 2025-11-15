package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/git"
	"os"
)

// Register is the resolver for the register field.
func (r *mutationResolver) Register(ctx context.Context, email string, password string) (*AuthPayload, error) {
	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Insert user into database
	q := db.New(r.DB)
	dbUser, err := q.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token
	token, err := auth.GenerateToken(fmt.Sprintf("%x", dbUser.ID.Bytes), email)
	if err != nil {
		return nil, err
	}

	return &AuthPayload{
		Token: token,
		User: &User{
			ID:    fmt.Sprintf("%x", dbUser.ID.Bytes),
			Email: dbUser.Email,
		},
	}, nil
}

// Login is the resolver for the login field.
func (r *mutationResolver) Login(ctx context.Context, email string, password string) (*AuthPayload, error) {
	// Get user from database
	q := db.New(r.DB)
	dbUser, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	if !auth.CheckPasswordHash(password, dbUser.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate token
	token, err := auth.GenerateToken(fmt.Sprintf("%x", dbUser.ID.Bytes), dbUser.Email)
	if err != nil {
		return nil, err
	}

	return &AuthPayload{
		Token: token,
		User: &User{
			ID:    fmt.Sprintf("%x", dbUser.ID.Bytes),
			Email: dbUser.Email,
		},
	}, nil
}

// CreateProject is the resolver for the createProject field.
func (r *mutationResolver) CreateProject(ctx context.Context, name string) (*Project, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	projectUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate project ID: %w", err)
	}
	projectID := projectUUID.String()
	projectPath := r.getProjectPath(projectID)

	// Create project in database
	q := db.New(r.DB)
	dbProject, err := q.CreateProject(ctx, db.CreateProjectParams{
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
		Name:    name,
		GitPath: projectPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create project in db: %w", err)
	}

	// Initialize git repo
	if err := git.InitRepo(projectPath); err != nil {
		return nil, fmt.Errorf("failed to init git repo: %w", err)
	}

	// Create main.tex template
	template := `\documentclass{article}
\usepackage[utf8]{inputenc}

\title{Untitled}
\author{}
\date{}

\begin{document}

\maketitle

\section{Introduction}

Your content here.

\end{document}
`
	if err := git.WriteFile(projectPath, "main.tex", template); err != nil {
		return nil, fmt.Errorf("failed to write template file: %w", err)
	}

	// Initial commit
	_, err = git.CommitChanges(projectPath, user.Email, "Initial commit")
	if err != nil {
		return nil, fmt.Errorf("failed to create initial commit: %w", err)
	}

	return &Project{
		ID:        fmt.Sprintf("%x", dbProject.ID.Bytes),
		Name:      dbProject.Name,
		CreatedAt: dbProject.CreatedAt.Time,
		UpdatedAt: dbProject.UpdatedAt.Time,
	}, nil
}

// DeleteProject is the resolver for the deleteProject field.
func (r *mutationResolver) DeleteProject(ctx context.Context, id string) (bool, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return false, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(id)
	if err != nil {
		return false, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return false, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	if project.UserID.Bytes != userUUID {
		return false, fmt.Errorf("not authorized")
	}

	// Delete project from database
	if err := q.DeleteProject(ctx, pgProjectID); err != nil {
		return false, fmt.Errorf("failed to delete project from db: %w", err)
	}

	// Delete git repository
	projectPath := r.getProjectPath(id)
	if err := os.RemoveAll(projectPath); err != nil {
		return false, fmt.Errorf("failed to delete git repository: %w", err)
	}

	return true, nil
}

// SaveFile is the resolver for the saveFile field.
func (r *mutationResolver) SaveFile(ctx context.Context, projectID string, path string, content string) (*File, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if project.UserID.Bytes != userUUID {
		return nil, fmt.Errorf("not authorized")
	}

	// Write file to git repository
	projectPath := r.getProjectPath(projectID)
	if err := git.WriteFile(projectPath, path, content); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &File{
		Path:    path,
		Content: content,
	}, nil
}

// DeleteFile is the resolver for the deleteFile field.
func (r *mutationResolver) DeleteFile(ctx context.Context, projectID string, path string) (bool, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return false, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return false, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return false, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	if project.UserID.Bytes != userUUID {
		return false, fmt.Errorf("not authorized")
	}

	// Delete file from git repository
	projectPath := r.getProjectPath(projectID)
	if err := git.DeleteFile(projectPath, path); err != nil {
		return false, fmt.Errorf("failed to delete file: %w", err)
	}

	return true, nil
}

// Commit is the resolver for the commit field.
func (r *mutationResolver) Commit(ctx context.Context, projectID string, message string) (*Commit, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if project.UserID.Bytes != userUUID {
		return nil, fmt.Errorf("not authorized")
	}

	// Commit changes
	projectPath := r.getProjectPath(projectID)
	commit, err := git.CommitChanges(projectPath, user.Email, message)
	if err != nil {
		return nil, fmt.Errorf("failed to commit changes: %w", err)
	}

	// Update project timestamp
	if err := q.UpdateProjectTimestamp(ctx, pgProjectID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to update project timestamp: %v\n", err)
	}

	return &Commit{
		Hash:    commit.Hash,
		Message: commit.Message,
		Author:  commit.Author,
		Date:    commit.Date,
	}, nil
}

// Me is the resolver for the me field.
func (r *queryResolver) Me(ctx context.Context) (*User, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, nil
	}

	userID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	q := db.New(r.DB)
	dbUser, err := q.GetUserByID(ctx, pgUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &User{
		ID:    fmt.Sprintf("%x", dbUser.ID.Bytes),
		Email: dbUser.Email,
	}, nil
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
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	q := db.New(r.DB)
	dbProject, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if dbProject.UserID.Bytes != userUUID {
		return nil, fmt.Errorf("not authorized")
	}

	return &Project{
		ID:        fmt.Sprintf("%x", dbProject.ID.Bytes),
		Name:      dbProject.Name,
		CreatedAt: dbProject.CreatedAt.Time,
		UpdatedAt: dbProject.UpdatedAt.Time,
	}, nil
}

// Files is the resolver for the files field.
func (r *queryResolver) Files(ctx context.Context, projectID string) ([]*File, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if project.UserID.Bytes != userUUID {
		return nil, fmt.Errorf("not authorized")
	}

	// Get files from git repository
	projectPath := r.getProjectPath(projectID)
	gitFiles, err := git.ListFiles(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	files := make([]*File, len(gitFiles))
	for i, f := range gitFiles {
		files[i] = &File{
			Path:    f.Path,
			Content: f.Content,
		}
	}

	return files, nil
}

// File is the resolver for the file field.
func (r *queryResolver) File(ctx context.Context, projectID string, path string) (*File, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if project.UserID.Bytes != userUUID {
		return nil, fmt.Errorf("not authorized")
	}

	// Read file from git repository
	projectPath := r.getProjectPath(projectID)
	content, err := git.ReadFile(projectPath, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &File{
		Path:    path,
		Content: content,
	}, nil
}

// History is the resolver for the history field.
func (r *queryResolver) History(ctx context.Context, projectID string) ([]*Commit, error) {
	user, ok := auth.GetUserFromContext(ctx)
	if !ok {
		return nil, errors.New("not authenticated")
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	pgProjectID := pgtype.UUID{Bytes: projectUUID, Valid: true}

	// Verify ownership
	q := db.New(r.DB)
	project, err := q.GetProject(ctx, pgProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}

	userUUID, err := uuid.Parse(user.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if project.UserID.Bytes != userUUID {
		return nil, fmt.Errorf("not authorized")
	}

	// Get commit history from git repository
	projectPath := r.getProjectPath(projectID)
	gitCommits, err := git.GetHistory(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	commits := make([]*Commit, len(gitCommits))
	for i, c := range gitCommits {
		commits[i] = &Commit{
			Hash:    c.Hash,
			Message: c.Message,
			Author:  c.Author,
			Date:    c.Date,
		}
	}

	return commits, nil
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
