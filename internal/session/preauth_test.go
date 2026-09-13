package session

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestMintParsePreauth(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-preauth")
	t.Setenv("GO_ENV", "development")

	tok, err := MintPreauth("sess-1")
	if err != nil {
		t.Fatalf("MintPreauth: %v", err)
	}
	got, err := ParsePreauth(tok)
	if err != nil {
		t.Fatalf("ParsePreauth: %v", err)
	}
	if got != "sess-1" {
		t.Fatalf("session = %q, want sess-1", got)
	}
}

func TestParsePreauthRejectsNonHMAC(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-preauth")
	t.Setenv("GO_ENV", "development")

	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"sid": "sess-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	sig := base64.RawURLEncoding.EncodeToString([]byte("not-a-real-sig"))
	rsToken := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + "." + sig

	if _, err := ParsePreauth(rsToken); err == nil {
		t.Fatal("expected error for RS256 token")
	}
}

func TestParsePreauthRejectsGarbage(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-preauth")
	t.Setenv("GO_ENV", "development")

	if _, err := ParsePreauth(""); err != ErrPreauthInvalid {
		t.Fatalf("empty: %v", err)
	}
	if _, err := ParsePreauth("not.a.jwt"); err != ErrPreauthInvalid {
		t.Fatalf("garbage: %v", err)
	}
}
