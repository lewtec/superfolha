package session

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	igit "github.com/lewtec/superfolha/internal/git"
	cryptossh "golang.org/x/crypto/ssh"
)

const signWait = 30 * time.Second

var (
	ErrSignTimeout = errors.New("signer tab timeout")
	ErrNoSigner    = errors.New("no session key on this tab")
)

// SignBridge RPCs ssh.sign to one tab and waits for ssh.sign.ok.
type SignBridge struct {
	send func(id string, data []byte)

	mu      sync.Mutex
	pending map[string]chan signResult
}

type signResult struct {
	sig []byte
	err error
}

// NewSignBridge sends ssh.sign frames through send.
func NewSignBridge(send func(id string, data []byte)) *SignBridge {
	return &SignBridge{send: send, pending: map[string]chan signResult{}}
}

// Ask is the TabSigner callback.
func (b *SignBridge) Ask(data []byte) ([]byte, error) {
	if b == nil || b.send == nil {
		return nil, ErrNoSigner
	}
	id := uuid.NewString()
	ch := make(chan signResult, 1)
	b.mu.Lock()
	b.pending[id] = ch
	b.mu.Unlock()
	b.send(id, data)
	timer := time.NewTimer(signWait)
	defer timer.Stop()
	select {
	case r := <-ch:
		return r.sig, r.err
	case <-timer.C:
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, ErrSignTimeout
	}
}

// Complete finishes a sign request from the tab.
func (b *SignBridge) Complete(id string, sig []byte, err error) {
	if b == nil {
		return
	}
	b.mu.Lock()
	ch := b.pending[id]
	delete(b.pending, id)
	b.mu.Unlock()
	if ch == nil {
		return
	}
	ch <- signResult{sig: sig, err: err}
}

// TabSSH builds a SessionSSH bound to this bridge and public line.
func TabSSH(line string, bridge *SignBridge) (igit.SessionSSH, error) {
	pub, canon, err := igit.ParseAuthorized(line)
	if err != nil {
		return nil, err
	}
	return &igit.TabSigner{Pub: pub, Line: canon, Ask: bridge.Ask}, nil
}

// SamePublic reports whether offered matches the session public key.
func SamePublic(offered, want string) bool {
	a, _, err := igit.ParseAuthorized(offered)
	if err != nil {
		return false
	}
	b, _, err := igit.ParseAuthorized(want)
	if err != nil {
		return false
	}
	return cryptossh.FingerprintSHA256(a) == cryptossh.FingerprintSHA256(b)
}

// SignOKFrame is ssh.sign for the tab.
func SignOKFrame(id string, data []byte) []byte {
	b, err := json.Marshal(map[string]string{
		"type": "ssh.sign",
		"id":   id,
		"data": base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		return nil
	}
	return b
}
