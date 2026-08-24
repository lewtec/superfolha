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
