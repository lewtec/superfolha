package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lewtec/superfolha/internal/db"
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

func (s *Service) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
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
	// Get user from database
	q := db.New(s.db)
	dbUser, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	if !CheckPasswordHash(password, dbUser.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
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

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*db.User, error) {
	q := db.New(s.db)
	user, err := q.GetUserByID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, err
	}
	return &user, nil
}
