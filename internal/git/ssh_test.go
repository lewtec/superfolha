package git

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestParseSessionSSHKeyStable(t *testing.T) {
	seed := bytes.Repeat([]byte{1}, 32)
	a, err := ParseSessionSSHKey(seed)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseSessionSSHKey(seed)
	if err != nil {
		t.Fatal(err)
	}
	if a.Authorized != b.Authorized {
		t.Fatalf("authorized mismatch\n%s\n%s", a.Authorized, b.Authorized)
	}
	if !strings.HasPrefix(a.Authorized, "ssh-ed25519 ") {
		t.Fatalf("authorized = %q", a.Authorized)
	}
	if _, err := ParseSessionSSHKey([]byte("nope")); !errors.Is(err, ErrBadSSHSeed) {
		t.Fatalf("short seed: %v; want ErrBadSSHSeed", err)
	}
}

func TestAuthFailed(t *testing.T) {
	// These strings mimic go-git / x/crypto/ssh handshake text AuthFailed matches.
	var (
		errPermDeniedPublickey = errors.New("permission denied (publickey)")
		errSSHHandshakeAuth    = errors.New("ssh: handshake failed: ssh: unable to authenticate")
		errEmptyRepo           = errors.New("remote repository is empty")
	)
	if AuthFailed(nil) {
		t.Fatal("nil")
	}
	if !AuthFailed(errPermDeniedPublickey) {
		t.Fatal("publickey")
	}
	if !AuthFailed(errSSHHandshakeAuth) {
		t.Fatal("handshake")
	}
	if AuthFailed(errEmptyRepo) {
		t.Fatal("empty repo is not auth")
	}
}

func TestTabSignerSignSentinels(t *testing.T) {
	t.Parallel()
	var none *TabSigner
	if _, err := none.Sign(nil, nil); !errors.Is(err, ErrNoSignerTab) {
		t.Fatalf("nil TabSigner: %v; want ErrNoSignerTab", err)
	}
	empty := &TabSigner{}
	if _, err := empty.Sign(nil, nil); !errors.Is(err, ErrNoSignerTab) {
		t.Fatalf("missing Ask: %v; want ErrNoSignerTab", err)
	}
	short := &TabSigner{Ask: func([]byte) ([]byte, error) { return []byte("nope"), nil }}
	if _, err := short.Sign(nil, []byte("data")); !errors.Is(err, ErrBadSSHSignature) {
		t.Fatalf("short signature: %v; want ErrBadSSHSignature", err)
	}
}

func TestTofuCallbackHostKeyChanged(t *testing.T) {
	host := "janitor-tofu.example:22"
	keyHost := "janitor-tofu.example"
	tofu.Delete(host)
	tofu.Delete(keyHost)
	t.Cleanup(func() {
		tofu.Delete(host)
		tofu.Delete(keyHost)
	})

	a, err := NewSessionSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewSessionSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	if err := tofuCallback(host, nil, a.Signer.PublicKey()); err != nil {
		t.Fatalf("first pin: %v", err)
	}
	if err := tofuCallback(host, nil, a.Signer.PublicKey()); err != nil {
		t.Fatalf("same key: %v", err)
	}
	err = tofuCallback(host, nil, b.Signer.PublicKey())
	if !errors.Is(err, ErrHostKeyChanged) {
		t.Fatalf("changed key: %v; want ErrHostKeyChanged", err)
	}
}

func TestParseAuthorized(t *testing.T) {
	k, err := NewSessionSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	pub, line, err := ParseAuthorized(k.Authorized)
	if err != nil {
		t.Fatal(err)
	}
	if pub == nil || !strings.HasPrefix(line, "ssh-ed25519 ") {
		t.Fatalf("line = %q", line)
	}
	if _, _, err := ParseAuthorized("ssh-rsa AAAA"); !errors.Is(err, ErrBadSSHPublic) {
		t.Fatalf("rsa: %v", err)
	}
}
