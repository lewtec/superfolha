package git

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	cryptossh "golang.org/x/crypto/ssh"
)

// SSHKey is a per-session Ed25519 key held in RAM.
type SSHKey struct {
	Signer     cryptossh.Signer
	Authorized string
}

// NewSessionSSHKey mints a key for one session. The private half never leaves the process.
func NewSessionSSHKey() (*SSHKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	signer, err := cryptossh.NewSignerFromKey(priv)
	if err != nil {
		return nil, err
	}
	line := strings.TrimSpace(string(cryptossh.MarshalAuthorizedKey(signer.PublicKey())))
	return &SSHKey{
		Signer:     signer,
		Authorized: line + " superfolha-session",
	}, nil
}

func (k *SSHKey) method() transport.AuthMethod {
	if k == nil || k.Signer == nil {
		return nil
	}
	pk := &gitssh.PublicKeys{User: "git", Signer: k.Signer}
	pk.HostKeyCallback = tofuCallback
	return pk
}

var tofu sync.Map // hostname → marshaled host key

func tofuCallback(hostname string, _ net.Addr, key cryptossh.PublicKey) error {
	host := hostname
	if h, _, err := net.SplitHostPort(hostname); err == nil {
		host = h
	}
	raw := key.Marshal()
	prev, ok := tofu.Load(host)
	if !ok {
		tofu.Store(host, raw)
		return nil
	}
	if bytes.Equal(prev.([]byte), raw) {
		return nil
	}
	return fmt.Errorf("ssh host key changed for %s", host)
}

// Clone copies remote@branch into dest using HTTP and/or SSH auth.
func Clone(dest, remoteURL, branch string, httpAuth *HTTPAuth, sshKey *SSHKey) error {
	auth := transport.AuthMethod(nil)
	if isSSHURL(remoteURL) {
		auth = sshKey.method()
	} else {
		auth = httpAuth.method()
	}
	opts := &gogit.CloneOptions{URL: remoteURL, Auth: auth}
	if branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
		opts.SingleBranch = true
	}
	_, err := gogit.PlainClone(dest, false, opts)
	if err != nil {
		return fmt.Errorf("clone %s: %w", remoteURL, err)
	}
	return nil
}

func isSSHURL(u string) bool {
	u = strings.ToLower(strings.TrimSpace(u))
	return strings.HasPrefix(u, "ssh://") || strings.Contains(u, "@") && !strings.Contains(u, "://")
}

// Push pushes HEAD to origin with HTTP or SSH credentials.
func Push(repoPath, branch string, httpAuth *HTTPAuth, sshKey *SSHKey) error {
	r, err := openRepo(repoPath)
	if err != nil {
		return err
	}
	auth := transport.AuthMethod(nil)
	if rem, rerr := r.Remote("origin"); rerr == nil && len(rem.Config().URLs) > 0 && isSSHURL(rem.Config().URLs[0]) {
		auth = sshKey.method()
	} else {
		auth = httpAuth.method()
	}
	ref := branch
	if ref == "" {
		head, herr := r.Head()
		if herr != nil {
			return fmt.Errorf("head: %w", herr)
		}
		ref = head.Name().Short()
	}
	err = r.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/heads/" + ref + ":refs/heads/" + ref)},
		Auth:       auth,
	})
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("push origin: %w", err)
	}
	return nil
}
