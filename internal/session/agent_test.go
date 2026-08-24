package session

import (
	"testing"

	igit "github.com/lewtec/superfolha/internal/git"
)

func TestSamePublicIgnoresComment(t *testing.T) {
	k, err := igit.NewSessionSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	if !SamePublic(k.Authorized, k.Authorized) {
		t.Fatal("same key")
	}
	other, err := igit.NewSessionSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	if SamePublic(k.Authorized, other.Authorized) {
		t.Fatal("different keys")
	}
}

func TestSignBridgeCompletes(t *testing.T) {
	var sentID string
	var sent []byte
	var b *SignBridge
	b = NewSignBridge(func(id string, data []byte) {
		sentID = id
		sent = data
		sig := make([]byte, 64)
		go b.Complete(id, sig, nil)
	})
	// 64-byte dummy; Ask does not check length
	sig, err := b.Ask([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if sentID == "" || string(sent) != "hello" {
		t.Fatalf("send id=%q data=%q", sentID, sent)
	}
	if len(sig) != 64 {
		t.Fatalf("sig len %d", len(sig))
	}
}
