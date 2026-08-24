package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
)

func TestVerifyChallenge(t *testing.T) {
	withJWTEnv(t, "challenge-secret", "development")
	ResetChallengeStateForTest()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ch, err := NewChallenge()
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, []byte(ch))
	id, err := VerifyChallenge(ch, base64.RawURLEncoding.EncodeToString(pub), base64.RawURLEncoding.EncodeToString(sig))
	if err != nil {
		t.Fatal(err)
	}
	if id.Login == "" || id.ID == "" {
		t.Fatalf("empty identity: %+v", id)
	}
	if want := Fingerprint(pub); id.Login != want {
		t.Fatalf("login = %q; want %q", id.Login, want)
	}
	_, err = VerifyChallenge(ch, base64.RawURLEncoding.EncodeToString(pub), base64.RawURLEncoding.EncodeToString(sig))
	if !errors.Is(err, ErrChallengeUsed) {
		t.Fatalf("replay: %v", err)
	}
}

func TestVerifyChallengeRejectsBadSig(t *testing.T) {
	withJWTEnv(t, "challenge-secret", "development")
	ResetChallengeStateForTest()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ch, err := NewChallenge()
	if err != nil {
		t.Fatal(err)
	}
	bad := make([]byte, ed25519.SignatureSize)
	_, err = VerifyChallenge(ch, base64.RawURLEncoding.EncodeToString(pub), base64.RawURLEncoding.EncodeToString(bad))
	if !errors.Is(err, ErrBadSignature) {
		t.Fatalf("got %v; want ErrBadSignature", err)
	}
}
