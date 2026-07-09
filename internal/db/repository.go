package db

import "context"

// Repository is the storage abstraction implemented by sqlite and postgres backends
// (same idea as ciborg's dual repository packages).
type Repository interface {
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)

	CreateProject(ctx context.Context, arg CreateProjectParams) (Project, error)
	GetProject(ctx context.Context, id string) (Project, error)
	GetUserProjects(ctx context.Context, userID string) ([]Project, error)
	UpdateProjectTimestamp(ctx context.Context, id string) error
	DeleteProject(ctx context.Context, id string) error

	// IsUniqueViolation reports whether err is a unique constraint failure.
	IsUniqueViolation(err error) bool

	Close() error
}
