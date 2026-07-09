package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/db"
)

var (
	ErrEmailTaken         = fmt.Errorf("email already registered")
	ErrInvalidCredentials = fmt.Errorf("invalid credentials")
)

type Service struct {
	repo db.Repository
}

func NewService(repo db.Repository) *Service {
	return &Service{repo: repo}
}

type AuthResponse struct {
	User  *db.User
	Token string
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
	email = normalizeEmail(email)

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Generate id in app so SQLite (no uuidv7 default) and Postgres stay aligned.
	// Postgres still accepts explicit UUID inserts on tables with DEFAULT uuidv7().
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user id: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		ID:           id.String(),
		Email:        email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		if s.repo.IsUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Always use the row returned from the DB (covers DEFAULT uuidv7() path).
	token, err := GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: &user, Token: token}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	email = normalizeEmail(email)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: &user, Token: token}, nil
}
