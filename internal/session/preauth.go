package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lewtec/superfolha/internal/auth"
)

const preauthTTL = 7 * 24 * time.Hour

var (
	ErrPreauthInvalid = errors.New("invalid preauth")
	ErrPreauthSession = errors.New("preauth session mismatch")
)

type preauthClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// MintPreauth returns a signed token that admits one signed-in user to sessionID.
func MintPreauth(sessionID string) (string, error) {
	secret, err := auth.JWTSecret()
	if err != nil {
		return "", err
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &preauthClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(preauthTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	return tok.SignedString(secret)
}

// ParsePreauth returns the session id bound in token.
func ParsePreauth(token string) (sessionID string, err error) {
	secret, err := auth.JWTSecret()
	if err != nil {
		return "", err
	}
	parsed, err := jwt.ParseWithClaims(token, &preauthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", auth.ErrUnexpectedSigning, t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return "", ErrPreauthInvalid
	}
	claims, ok := parsed.Claims.(*preauthClaims)
	if !ok || !parsed.Valid || claims.SessionID == "" {
		return "", ErrPreauthInvalid
	}
	return claims.SessionID, nil
}
