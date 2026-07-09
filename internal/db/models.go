package db

import "time"

// User is the shared domain model (backend-agnostic).
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Project is the shared domain model (backend-agnostic).
type Project struct {
	ID        string
	UserID    string
	Name      string
	GitPath   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateUserParams struct {
	ID           string
	Email        string
	PasswordHash string
}

type CreateProjectParams struct {
	ID      string
	UserID  string
	Name    string
	GitPath string
}
