package auth

import (
	"testing"
)

func TestGenerateToken(t *testing.T) {
	// Setup environment using t.Setenv for automatic cleanup
	t.Setenv("JWT_SECRET", "test-secret-key-12345")
	t.Setenv("GO_ENV", "test")

	token, err := GenerateToken("user123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken returned empty string")
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got '%s'", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got '%s'", claims.Email)
	}
}

func TestGetJWTSecretDevFallback(t *testing.T) {
	// Setup environment using t.Setenv for automatic cleanup
	t.Setenv("JWT_SECRET", "")
	t.Setenv("GO_ENV", "development")

	secret, err := getJWTSecret()
	if err != nil {
		t.Fatalf("getJWTSecret failed in dev mode: %v", err)
	}
	if string(secret) != "dev-secret-change-in-production" {
		t.Errorf("Expected dev secret, got '%s'", string(secret))
	}
}

func TestGetJWTSecretProductionMissing(t *testing.T) {
	// Setup environment using t.Setenv for automatic cleanup
	t.Setenv("JWT_SECRET", "")
	t.Setenv("GO_ENV", "production")

	_, err := getJWTSecret()
	if err != ErrJWTSecretRequired {
		t.Errorf("Expected ErrJWTSecretRequired, got %v", err)
	}
}
