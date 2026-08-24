package auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	challengeTTL = 2 * time.Minute
	challengeAud = "superfolha-login"
)

var (
	ErrChallengeInvalid = errors.New("invalid login challenge")
	ErrChallengeUsed    = errors.New("login challenge already used")
	ErrBadPublicKey     = errors.New("invalid ed25519 public key")
	ErrBadSignature     = errors.New("invalid ed25519 signature")
)

type challengeClaims struct {
	jwt.RegisteredClaims
}

var (
	usedMu  sync.Mutex
	usedJTI = map[string]time.Time{}
)

// NewChallenge issues a short-lived signed nonce. The client signs this string.
func NewChallenge() (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &challengeClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id.String(),
			Audience:  jwt.ClaimStrings{challengeAud},
			ExpiresAt: jwt.NewNumericDate(now.Add(challengeTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	return tok.SignedString(secret)
}

// Identity is a challenge-sign principal. Login is a short fingerprint.
type Identity struct {
	ID    string
	Login string
}

// Fingerprint returns ed25519:<16 hex chars> from a raw 32-byte public key.
func Fingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return "ed25519:" + hex.EncodeToString(sum[:8])
}

// VerifyChallenge checks the challenge JWT and the Ed25519 signature over it.
func VerifyChallenge(challenge, publicKeyB64, signatureB64 string) (Identity, error) {
	var zero Identity
	pubRaw, err := base64.RawURLEncoding.DecodeString(publicKeyB64)
	if err != nil || len(pubRaw) != ed25519.PublicKeySize {
		pubRaw, err = base64.StdEncoding.DecodeString(publicKeyB64)
	}
	if err != nil || len(pubRaw) != ed25519.PublicKeySize {
		return zero, ErrBadPublicKey
	}
	sig, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		sig, err = base64.StdEncoding.DecodeString(signatureB64)
	}
	if err != nil || len(sig) != ed25519.SignatureSize {
		return zero, ErrBadSignature
	}

	secret, err := getJWTSecret()
	if err != nil {
		return zero, err
	}
	parsed, err := jwt.ParseWithClaims(challenge, &challengeClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: %v", ErrUnexpectedSigning, t.Header["alg"])
		}
		return secret, nil
	}, jwt.WithAudience(challengeAud))
	if err != nil || !parsed.Valid {
		return zero, ErrChallengeInvalid
	}
	claims, ok := parsed.Claims.(*challengeClaims)
	if !ok || claims.ID == "" {
		return zero, ErrChallengeInvalid
	}
	if err := consumeJTI(claims.ID, claims.ExpiresAt); err != nil {
		return zero, err
	}
	if !ed25519.Verify(ed25519.PublicKey(pubRaw), []byte(challenge), sig) {
		return zero, ErrBadSignature
	}
	sum := sha256.Sum256(pubRaw)
	return Identity{
		ID:    hex.EncodeToString(sum[:]),
		Login: Fingerprint(pubRaw),
	}, nil
}

func consumeJTI(id string, exp *jwt.NumericDate) error {
	usedMu.Lock()
	defer usedMu.Unlock()
	now := time.Now()
	for k, until := range usedJTI {
		if until.Before(now) {
			delete(usedJTI, k)
		}
	}
	if _, ok := usedJTI[id]; ok {
		return ErrChallengeUsed
	}
	until := now.Add(challengeTTL)
	if exp != nil && exp.Time.After(now) {
		until = exp.Time
	}
	usedJTI[id] = until
	return nil
}

// ResetChallengeStateForTest clears used challenge ids. Tests only.
func ResetChallengeStateForTest() {
	usedMu.Lock()
	usedJTI = map[string]time.Time{}
	usedMu.Unlock()
}
