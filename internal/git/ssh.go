package git

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	cryptossh "golang.org/x/crypto/ssh"
)

// ErrBadSSHSeed is returned when a session seed is not 32 bytes.
var ErrBadSSHSeed = errors.New("invalid ssh seed")

// ErrBadSSHPublic is returned when an authorized_keys line is not ssh-ed25519.
var ErrBadSSHPublic = errors.New("invalid ssh public key")

// SessionSSH is git SSH auth. Product path is a tab signer; tests may use SSHKey.
type SessionSSH interface {
	AuthMethod() transport.AuthMethod
	AuthorizedKey() string
}

// SSHKey is a full Ed25519 key (tests). The product path does not store the private half.
type SSHKey struct {
	Signer     cryptossh.Signer
	Authorized string
}

// NewSessionSSHKey mints a key for one session.
func NewSessionSSHKey() (*SSHKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return sshKeyFromPriv(priv)
}

// ParseSessionSSHKey rebuilds a session key from a 32-byte seed.
func ParseSessionSSHKey(seed []byte) (*SSHKey, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, ErrBadSSHSeed
	}
	return sshKeyFromPriv(ed25519.NewKeyFromSeed(seed))
}

func sshKeyFromPriv(priv ed25519.PrivateKey) (*SSHKey, error) {
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

func (k *SSHKey) AuthMethod() transport.AuthMethod {
	if k == nil || k.Signer == nil {
		return nil
	}
	return publicKeys(k.Signer)
}

func (k *SSHKey) AuthorizedKey() string {
	if k == nil {
		return ""
	}
	return k.Authorized
}

var _ gitssh.AuthMethod = (*timedSSH)(nil)

func publicKeys(signer cryptossh.Signer) transport.AuthMethod {
	if signer == nil {
		return nil
	}
	pk := &gitssh.PublicKeys{User: "git", Signer: signer}
	pk.HostKeyCallback = tofuCallback
	return &timedSSH{inner: pk}
}

const sshDialTimeout = 20 * time.Second

type timedSSH struct {
	inner *gitssh.PublicKeys
}

func (t *timedSSH) Name() string   { return t.inner.Name() }
func (t *timedSSH) String() string { return t.inner.String() }

func (t *timedSSH) ClientConfig() (*cryptossh.ClientConfig, error) {
	cfg, err := t.inner.ClientConfig()
	if err != nil {
		return nil, err
	}
	cfg.Timeout = sshDialTimeout
	return cfg, nil
}

// ParseAuthorized accepts an ssh-ed25519 authorized_keys line.
func ParseAuthorized(line string) (cryptossh.PublicKey, string, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, "", ErrBadSSHPublic
	}
	pub, comment, _, _, err := cryptossh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return nil, "", ErrBadSSHPublic
	}
	if pub.Type() != cryptossh.KeyAlgoED25519 {
		return nil, "", ErrBadSSHPublic
	}
	out := strings.TrimSpace(string(cryptossh.MarshalAuthorizedKey(pub)))
	if comment == "" {
		comment = "superfolha-session"
	}
	return pub, out + " " + comment, nil
}

// TabSigner asks a browser tab to sign SSH session data.
type TabSigner struct {
	Pub  cryptossh.PublicKey
	Line string
	Ask  func(data []byte) ([]byte, error)
}

func (t *TabSigner) PublicKey() cryptossh.PublicKey { return t.Pub }

func (t *TabSigner) Sign(_ io.Reader, data []byte) (*cryptossh.Signature, error) {
	if t == nil || t.Ask == nil {
		return nil, errors.New("no signer tab")
	}
	sig, err := t.Ask(data)
	if err != nil {
		return nil, err
	}
	if len(sig) != ed25519.SignatureSize {
		return nil, errors.New("bad ssh signature")
	}
	return &cryptossh.Signature{Format: cryptossh.KeyAlgoED25519, Blob: sig}, nil
}

func (t *TabSigner) AuthMethod() transport.AuthMethod { return publicKeys(t) }

func (t *TabSigner) AuthorizedKey() string {
	if t == nil {
		return ""
	}
	return t.Line
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

// Clone copies remote@branch into dest over SSH.
func Clone(dest, remoteURL, branch string, ssh SessionSSH) error {
	auth := transport.AuthMethod(nil)
	if ssh != nil {
		auth = ssh.AuthMethod()
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

// CloneLocal copies a filesystem git repo into dest, or inits dest if src is not a repo.
func CloneLocal(dest, srcPath, branch string) error {
	if _, err := gogit.PlainOpen(srcPath); err == nil {
		return Clone(dest, "file://"+srcPath, branch, nil)
	}
	return InitRepo(dest)
}

// Push pushes HEAD to origin over SSH.
func Push(repoPath, branch string, ssh SessionSSH) error {
	r, err := openRepo(repoPath)
	if err != nil {
		return err
	}
	auth := transport.AuthMethod(nil)
	if ssh != nil {
		auth = ssh.AuthMethod()
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

// Fetch pulls from origin over SSH. Used to tell "no write" from "no key".
func Fetch(repoPath, branch string, ssh SessionSSH) error {
	r, err := openRepo(repoPath)
	if err != nil {
		return err
	}
	auth := transport.AuthMethod(nil)
	if ssh != nil {
		auth = ssh.AuthMethod()
	}
	opts := &gogit.FetchOptions{RemoteName: "origin", Auth: auth}
	if branch != "" {
		opts.RefSpecs = []config.RefSpec{config.RefSpec("refs/heads/" + branch + ":refs/remotes/origin/" + branch)}
	}
	err = r.Fetch(opts)
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch origin: %w", err)
	}
	return nil
}

// LsRemote lists refs on the remote over SSH. No local repo required.
func LsRemote(remoteURL string, ssh SessionSSH) error {
	auth := transport.AuthMethod(nil)
	if ssh != nil {
		auth = ssh.AuthMethod()
	}
	rem := gogit.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	})
	_, err := rem.List(&gogit.ListOptions{Auth: auth})
	if err != nil {
		return fmt.Errorf("ls-remote %s: %w", remoteURL, err)
	}
	return nil
}

// AuthFailed reports SSH public-key / permission denied.
// go-git wraps transport.ErrAuthenticationRequired on HTTP only. SSH
// returns a bare x/crypto/ssh handshake fmt.Errorf, so we also match
// the usual strings.
func AuthFailed(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, transport.ErrAuthenticationRequired) || errors.Is(err, transport.ErrAuthorizationFailed) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "permission denied") ||
		strings.Contains(s, "publickey") ||
		strings.Contains(s, "unable to authenticate") ||
		strings.Contains(s, "authentication required") ||
		strings.Contains(s, "authorization failed")
}
