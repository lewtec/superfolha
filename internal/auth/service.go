package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lewtec/superfolha/internal/db"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Service struct {
	db db.DBTX
}

func NewService(db db.DBTX) *Service {
	return &Service{db: db}
}

type AuthResponse struct {
	User  *db.User
	Token string
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *Service) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
	email = normalizeEmail(email)

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Insert user into database
	q := db.New(s.db)
	dbUser, err := q.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate token
	token, err := GenerateToken(uuid.UUID(dbUser.ID.Bytes).String(), email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:  &dbUser,
		Token: token,
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	email = normalizeEmail(email)

	// Get user from database
	q := db.New(s.db)
	dbUser, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check password
	if !CheckPasswordHash(password, dbUser.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Generate token
	token, err := GenerateToken(uuid.UUID(dbUser.ID.Bytes).String(), dbUser.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:  &dbUser,
		Token: token,
	}, nil
}
