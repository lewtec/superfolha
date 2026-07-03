package auth

import (
	"fmt"

	"context"

	"github.com/lewtec/superfolha/internal/telemetry"

	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost factor for bcrypt hashing
	// 12 provides a good balance between security and performance in 2024
	// Each increment doubles the time required to hash
	BcryptCost = 12

	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 8
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrJWTSecretRequired = errors.New("JWT_SECRET environment variable is required")
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// getJWTSecret retrieves the JWT secret from environment variable
func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	goEnv := os.Getenv("GO_ENV")                                                                   // Get GO_ENV as well
	log.Printf("Auth: getJWTSecret called. JWT_SECRET env: '%s', GO_ENV env: '%s'", secret, goEnv) // Log env vars

	if secret == "" {
		// Fallback for development only - should never be used in production
		if goEnv == "development" { // Use goEnv here
			telemetry.ReportError(context.Background(), fmt.Errorf("Auth: Using development fallback JWT secret."))
			return []byte("dev-secret-change-in-production"), nil
		}
		telemetry.ReportError(context.Background(), fmt.Errorf("Auth: JWT_SECRET environment variable is required and not found."))
		return nil, ErrJWTSecretRequired
	}
	telemetry.ReportError(context.Background(), fmt.Errorf("Auth: Using JWT_SECRET from environment variable."))
	return []byte(secret), nil
}

// HashPassword hashes a password using bcrypt with recommended cost factor
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, email string) (string, error) {
	jwtSecret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	jwtSecret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
