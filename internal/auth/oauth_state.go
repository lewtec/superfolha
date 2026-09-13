package auth

import (
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
	parsed, err := jwt.ParseWithClaims(state, &oauthStateClaims{}, HMACKeyfunc(secret))
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*oauthStateClaims)
	if !ok || !parsed.Valid {
		return "", ErrInvalidToken
	}
	return claims.Next, nil
}
