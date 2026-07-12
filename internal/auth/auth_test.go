package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func withJWTEnv(t *testing.T, secret, goEnv string) {
	t.Helper()
	resetJWTSecretCache()
	t.Setenv("JWT_SECRET", secret)
	t.Setenv("GO_ENV", goEnv)
	t.Cleanup(resetJWTSecretCache)
}

func TestValidateToken_ValidHS256(t *testing.T) {
	withJWTEnv(t, "test-secret-for-jwt-validation", "")

	token, err := GenerateToken("user-1", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %q, want user@example.com", claims.Email)
	}
}

func TestValidateToken_RejectsNoneAlg(t *testing.T) {
	withJWTEnv(t, "test-secret-for-jwt-validation", "")

	// Craft an unsigned token with alg=none (classic confusion vector).
	header, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]string{"user_id": "evil", "email": "evil@example.com"})
	noneToken := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "."

	_, err := ValidateToken(noneToken)
	if err == nil {
		t.Fatal("expected error for alg=none token, got nil")
	}
	if !errors.Is(err, ErrUnexpectedSigning) && !strings.Contains(err.Error(), "unexpected signing method") {
		// jwt/v5 may wrap or reject none before keyfunc; either rejection is fine.
		if !strings.Contains(strings.ToLower(err.Error()), "none") &&
			!strings.Contains(strings.ToLower(err.Error()), "signing") &&
			!strings.Contains(strings.ToLower(err.Error()), "algorithm") {
			t.Logf("rejected with: %v (acceptable if not accepted)", err)
		}
	}
}

func TestValidateToken_RejectsNonHMAC(t *testing.T) {
	withJWTEnv(t, "test-secret-for-jwt-validation", "")

	// Header claims HS256 but we forge RS256-looking method via raw header alg=RS256
	// without a valid signature — keyfunc must reject before accepting the secret.
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	payload, _ := json.Marshal(map[string]any{
		"user_id": "evil",
		"email":   "evil@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	// Dummy signature so Parse reaches keyfunc / method checks.
	sig := base64.RawURLEncoding.EncodeToString([]byte("not-a-real-sig"))
	rsToken := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." + sig

	_, err := ValidateToken(rsToken)
	if err == nil {
		t.Fatal("expected error for RS256 token, got nil")
	}
}

func TestValidateToken_EmptyAndInvalid(t *testing.T) {
	withJWTEnv(t, "test-secret-for-jwt-validation", "")

	if _, err := ValidateToken(""); err == nil {
		t.Error("expected error for empty token")
	}
	if _, err := ValidateToken("not.a.jwt"); err == nil {
		t.Error("expected error for garbage token")
	}
	if _, err := ValidateToken("a.b.c"); err == nil {
		t.Error("expected error for malformed token parts")
	}
}

func TestValidateToken_TamperedSignature(t *testing.T) {
	withJWTEnv(t, "test-secret-for-jwt-validation", "")

	token, err := GenerateToken("user-1", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token parts: %d", len(parts))
	}
	parts[2] = base64.RawURLEncoding.EncodeToString([]byte("tampered-signature"))
	_, err = ValidateToken(strings.Join(parts, "."))
	if err == nil {
		t.Fatal("expected error for tampered signature")
	}
}

func TestGenerateToken_DevFallback(t *testing.T) {
	withJWTEnv(t, "", "development")

	token, err := GenerateToken("dev-user", "dev@example.com")
	if err != nil {
		t.Fatalf("GenerateToken with dev fallback: %v", err)
	}
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken with dev fallback: %v", err)
	}
	if claims.UserID != "dev-user" {
		t.Errorf("UserID = %q, want dev-user", claims.UserID)
	}
}

func TestGenerateToken_MissingSecret(t *testing.T) {
	withJWTEnv(t, "", "production")

	_, err := GenerateToken("user", "user@example.com")
	if !errors.Is(err, ErrJWTSecretRequired) {
		t.Fatalf("err = %v, want ErrJWTSecretRequired", err)
	}
}

func TestGetJWTSecret_LoadsOnce(t *testing.T) {
	withJWTEnv(t, "once-secret", "")

	s1, err1 := getJWTSecret()
	s2, err2 := getJWTSecret()
	if err1 != nil || err2 != nil {
		t.Fatalf("getJWTSecret errors: %v, %v", err1, err2)
	}
	if string(s1) != "once-secret" || string(s2) != "once-secret" {
		t.Fatalf("secrets = %q, %q", s1, s2)
	}
	// Changing env after first load must not affect the cache.
	t.Setenv("JWT_SECRET", "different-secret")
	s3, err3 := getJWTSecret()
	if err3 != nil {
		t.Fatalf("getJWTSecret: %v", err3)
	}
	if string(s3) != "once-secret" {
		t.Errorf("cached secret changed after env update: %q", s3)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	withJWTEnv(t, "test-secret-for-jwt-validation", "")

	secret, err := getJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	claims := &Claims{
		UserID: "user-1",
		Email:  "user@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(signed); err == nil {
		t.Fatal("expected error for expired token")
	}
}
