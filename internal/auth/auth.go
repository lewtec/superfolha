package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lewtec/superfolha/internal/appenv"
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
	ErrUnexpectedSigning = errors.New("unexpected signing method")

	jwtSecretOnce = new(sync.Once)
	jwtSecret     []byte
	jwtSecretErr  error
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// getJWTSecret retrieves the JWT secret from the environment once and caches it.
// Logs only on the first resolution so authenticated requests do not spam logs.
func getJWTSecret() ([]byte, error) {
	jwtSecretOnce.Do(func() {
		secret := os.Getenv("JWT_SECRET")
		isDev := appenv.IsDevelopment()
		slog.Info("resolving JWT secret", "jwt_secret_set", secret != "", "is_development", isDev)

		if secret == "" {
			// Fallback for development only - should never be used in production
			if isDev {
				slog.Info("using development fallback JWT secret")
				jwtSecret = []byte("dev-secret-change-in-production")
				return
			}
			slog.Error("JWT_SECRET environment variable is required and not found")
			jwtSecretErr = ErrJWTSecretRequired
			return
		}
		slog.Info("using JWT_SECRET from environment variable")
		jwtSecret = []byte(secret)
	})
	return jwtSecret, jwtSecretErr
}

// resetJWTSecretCache clears the cached secret so tests can change env vars.
// Not for production use.
func resetJWTSecretCache() {
	jwtSecretOnce = new(sync.Once)
	jwtSecret = nil
	jwtSecretErr = nil
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
	secret, err := getJWTSecret()
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
	return token.SignedString(secret)
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Reject non-HMAC algorithms (classic alg confusion).
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigning, token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
