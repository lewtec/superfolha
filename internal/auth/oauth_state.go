package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const oauthStateTTL = 10 * time.Minute

type oauthStateClaims struct {
	Next string `json:"next"`
	jwt.RegisteredClaims
}

// MintOAuthState signs a short-lived OAuth CSRF state (optional next path).
func MintOAuthState(next string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &oauthStateClaims{
		Next: next,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(oauthStateTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	return tok.SignedString(secret)
}

// ParseOAuthState returns the next path stored in state.
func ParseOAuthState(state string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}
	parsed, err := jwt.ParseWithClaims(state, &oauthStateClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigning, t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*oauthStateClaims)
	if !ok || !parsed.Valid {
		return "", ErrInvalidToken
	}
	return claims.Next, nil
}
