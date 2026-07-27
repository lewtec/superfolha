package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/apierrors"
	"github.com/lewtec/superfolha/internal/db"
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
	if email == "" {
		return nil, apierrors.New(apierrors.CodeInvalidInput, "email is required")
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		if errors.Is(err, ErrPasswordTooShort) {
			return nil, apierrors.New(apierrors.CodePasswordTooShort, "password too short")
		}
		return nil, apierrors.Internal(err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, apierrors.Internal(fmt.Errorf("generate user id: %w", err))
	}

	user, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		ID:           id.String(),
		Email:        email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		if s.repo.IsUniqueViolation(err) {
			return nil, apierrors.New(apierrors.CodeEmailTaken, "email already registered")
		}
		return nil, apierrors.Internal(err)
	}

	token, err := GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, apierrors.Internal(err)
	}

	return &AuthResponse{User: &user, Token: token}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	email = normalizeEmail(email)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// Burn bcrypt cost comparable to a real password check so unknown
		// emails are not free to probe relative to wrong passwords.
		CheckPasswordHash(password, dummyBcryptHash)
		return nil, apierrors.New(apierrors.CodeInvalidCredentials, "invalid credentials")
	}

	if !CheckPasswordHash(password, user.PasswordHash) {
		return nil, apierrors.New(apierrors.CodeInvalidCredentials, "invalid credentials")
	}

	token, err := GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, apierrors.Internal(err)
	}

	return &AuthResponse{User: &user, Token: token}, nil
}
