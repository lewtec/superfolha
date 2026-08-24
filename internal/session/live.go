package session

import (
	"errors"
	"log/slog"
	"os"
	"sync"

	"github.com/google/uuid"
	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/remote"
)

var (
	ErrAlreadyLive      = errors.New("session already live for remote and branch")
	ErrNotFound         = errors.New("session not found")
	ErrNotHost          = errors.New("not session host")
	ErrKnockClosed      = errors.New("knock is disabled")
	ErrClone            = errors.New("clone failed")
	errWaitingDeployKey = errors.New("waiting for deploy key")
)

// Live is RAM metadata for one session incarnation.
type Live struct {
	ID        string
	Remote    string
	Branch    string
	HostLogin string
	KnockOn   bool
	Ready     bool
	SSHPublic string
	CloneURL  string
	Auth      *igit.HTTPAuth
	SSH       *igit.SSHKey
	admitted  map[string]struct{}
	knocking  map[string]struct{}
}

// Info is a copy safe to return to handlers.
type Info struct {
	ID        string
	Remote    string
	Branch    string
	HostLogin string
	KnockOn   bool
	Ready     bool
	SSHPublic string
	CloneURL  string
	Knocking  []string
}

func (l *Live) snapshot() Info {
	knocks := make([]string, 0, len(l.knocking))
	for login := range l.knocking {
		knocks = append(knocks, login)
	}
	return Info{
		ID:        l.ID,
		Remote:    l.Remote,
		Branch:    l.Branch,
		HostLogin: l.HostLogin,
		KnockOn:   l.KnockOn,
		Ready:     l.Ready,
		SSHPublic: l.SSHPublic,
		CloneURL:  l.CloneURL,
		Knocking:  knocks,
	}
}

func (l *Live) canOpen(login string) bool {
	if login == "" {
		return false
	}
	if login == l.HostLogin {
		return true
	}
	_, ok := l.admitted[login]
	return ok
}

// CreateCloner clones a remote into dest. Tests replace this.
type CreateCloner func(dest, remoteURL, branch string, auth *igit.HTTPAuth, sshKey *igit.SSHKey) error

// Registry holds live sessions and their hubs.
type Registry struct {
	mu     sync.Mutex
	hubs   map[string]*Hub
	live   map[string]*Live
	byKey  map[string]string
	svc    *project.Service
	cloner CreateCloner
}

// NewRegistry creates an empty hub registry.
func NewRegistry(svc *project.Service) *Registry {
	return &Registry{
		hubs:   make(map[string]*Hub),
		live:   make(map[string]*Live),
		byKey:  make(map[string]string),
		svc:    svc,
		cloner: igit.Clone,
	}
}

// SetCloner replaces the clone function (tests).
func (r *Registry) SetCloner(fn CreateCloner) {
	r.mu.Lock()
	r.cloner = fn
	r.mu.Unlock()
}

// Create clones remote@branch and opens a session. Host retry returns the existing session.
// A failed clone still creates a pending session so the host can add the SSH key and retry.
func (r *Registry) Create(hostLogin, rawRemote, branch string, auth *igit.HTTPAuth) (*Live, error) {
	if err := remote.Validate(rawRemote); err != nil {
		return nil, err
	}
	canon := remote.Canonical(rawRemote)
	cloneURL := remote.TransportURL(rawRemote)
	if branch == "" {
		branch = "main"
	}
	key := remote.Key(canon, branch)

	r.mu.Lock()
	if id, ok := r.byKey[key]; ok {
		existing := r.live[id]
		r.mu.Unlock()
		if existing != nil && existing.HostLogin == hostLogin {
			return existing, nil
		}
		return nil, ErrAlreadyLive
	}
	r.mu.Unlock()

	sshKey, err := igit.NewSessionSSHKey()
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sessionID := id.String()
	dest := r.svc.GetProjectPath(sessionID)
	// SSH remotes need the session pubkey on the forge first. Do not clone yet.
	var cloneErr error
	if remote.IsSSH(rawRemote) {
		cloneErr = errWaitingDeployKey
	} else {
		cloneErr = r.cloner(dest, cloneURL, branch, auth, sshKey)
		if cloneErr != nil {
			if rmErr := os.RemoveAll(dest); rmErr != nil {
				slog.Error("remove failed clone dest", "path", dest, "err", rmErr)
			}
		}
	}

	live := &Live{
		ID:        sessionID,
		Remote:    canon,
		CloneURL:  cloneURL,
		Branch:    branch,
		HostLogin: hostLogin,
		Auth:      auth,
		SSH:       sshKey,
		SSHPublic: sshKey.Authorized,
		admitted:  map[string]struct{}{hostLogin: {}},
		knocking:  map[string]struct{}{},
	}

	if cloneErr == nil {
		h, err := Open(r.svc, sessionID, hostLogin)
		if err != nil {
			if rmErr := os.RemoveAll(dest); rmErr != nil {
				slog.Error("remove failed open dest", "path", dest, "err", rmErr)
			}
			return nil, err
		}
		h.Auth = auth
		h.SSH = sshKey
		h.Branch = branch
		live.Ready = true
		r.mu.Lock()
		if existingID, ok := r.byKey[key]; ok {
			r.mu.Unlock()
			if closeErr := h.Close(); closeErr != nil {
				slog.Error("close raced hub", "err", closeErr)
			}
			if rmErr := os.RemoveAll(dest); rmErr != nil {
				slog.Error("remove raced dest", "path", dest, "err", rmErr)
			}
			existing := r.live[existingID]
			if existing != nil && existing.HostLogin == hostLogin {
				return existing, nil
			}
			return nil, ErrAlreadyLive
		}
		r.byKey[key] = sessionID
		r.live[sessionID] = live
		r.hubs[sessionID] = h
		r.mu.Unlock()
		return live, nil
	}

	r.mu.Lock()
	if existingID, ok := r.byKey[key]; ok {
		r.mu.Unlock()
		existing := r.live[existingID]
		if existing != nil && existing.HostLogin == hostLogin {
			return existing, nil
		}
		return nil, ErrAlreadyLive
	}
	r.byKey[key] = sessionID
	r.live[sessionID] = live
	r.mu.Unlock()
	return live, nil
}

// RetryClone tries to clone a pending session after the host added the SSH key.
func (r *Registry) RetryClone(sessionID, hostLogin string, auth *igit.HTTPAuth) error {
	r.mu.Lock()
	l, ok := r.live[sessionID]
	r.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	if l.HostLogin != hostLogin {
		return ErrNotHost
	}
	if l.Ready {
		return nil
	}
	if auth != nil {
		l.Auth = auth
	}
	dest := r.svc.GetProjectPath(sessionID)
	if err := r.cloner(dest, l.CloneURL, l.Branch, l.Auth, l.SSH); err != nil {
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			slog.Error("remove failed retry dest", "path", dest, "err", rmErr)
		}
		return errors.Join(ErrClone, err)
	}
	h, err := Open(r.svc, sessionID, hostLogin)
	if err != nil {
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			slog.Error("remove failed retry open", "path", dest, "err", rmErr)
		}
		return err
	}
	h.Auth = l.Auth
	h.SSH = l.SSH
	h.Branch = l.Branch
	r.mu.Lock()
	l.Ready = true
	r.hubs[sessionID] = h
	r.mu.Unlock()
	return nil
}

// GetIfLive returns a hub only if it is already open.
func (r *Registry) GetIfLive(sessionID string) *Hub {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hubs[sessionID]
}

// Live returns session metadata.
func (r *Registry) Live(sessionID string) (Info, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.live[sessionID]
	if !ok {
		return Info{}, false
	}
	return l.snapshot(), true
}

// CanOpen reports whether login may open the editor/WS.
func (r *Registry) CanOpen(sessionID, login string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.live[sessionID]
	if !ok {
		return false
	}
	return l.canOpen(login)
}

// GetOrOpen returns an existing hub. Sessions are created only via Create.
func (r *Registry) GetOrOpen(sessionID, ownerEmail string) (*Hub, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h, ok := r.hubs[sessionID]
	if !ok {
		return nil, ErrNotFound
	}
	if ownerEmail != "" && h.OwnerEmail == "" {
		h.OwnerEmail = ownerEmail
	}
	return h, nil
}

// RedeemPreauth admits login if the token matches sessionID.
func (r *Registry) RedeemPreauth(sessionID, login, token string) error {
	sid, err := ParsePreauth(token)
	if err != nil {
		return err
	}
	if sid != sessionID {
		return ErrPreauthSession
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.live[sessionID]
	if !ok {
		return ErrNotFound
	}
	l.admitted[login] = struct{}{}
	delete(l.knocking, login)
	return nil
}

// Knock queues login. No-op if already admitted.
func (r *Registry) Knock(sessionID, login string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.live[sessionID]
	if !ok {
		return ErrNotFound
	}
	if l.canOpen(login) {
		return nil
	}
	if !l.KnockOn {
		return ErrKnockClosed
	}
	l.knocking[login] = struct{}{}
	return nil
}

// Admit lets the host admit a knocking (or any) login.
func (r *Registry) Admit(sessionID, hostLogin, target string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.live[sessionID]
	if !ok {
		return ErrNotFound
	}
	if l.HostLogin != hostLogin {
		return ErrNotHost
	}
	l.admitted[target] = struct{}{}
	delete(l.knocking, target)
	return nil
}

// SetKnock enables or disables the knock door. Host only.
func (r *Registry) SetKnock(sessionID, hostLogin string, on bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.live[sessionID]
	if !ok {
		return ErrNotFound
	}
	if l.HostLogin != hostLogin {
		return ErrNotHost
	}
	l.KnockOn = on
	if !on {
		l.knocking = map[string]struct{}{}
	}
	return nil
}

// KickAll disconnects WS clients. Admits and preauth stay valid.
func (r *Registry) KickAll(sessionID, hostLogin string) error {
	r.mu.Lock()
	h, ok := r.hubs[sessionID]
	l, liveOK := r.live[sessionID]
	r.mu.Unlock()
	if !ok || !liveOK {
		return ErrNotFound
	}
	if l.HostLogin != hostLogin {
		return ErrNotHost
	}
	h.DisconnectAll()
	return nil
}

// ListFor returns sessions the login may open.
func (r *Registry) ListFor(login string) []Info {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Info, 0)
	for _, l := range r.live {
		if l.canOpen(login) {
			out = append(out, l.snapshot())
		}
	}
	return out
}

// End removes the session. Host only. Working tree is deleted.
func (r *Registry) End(sessionID, hostLogin string) error {
	r.mu.Lock()
	l, ok := r.live[sessionID]
	if !ok {
		r.mu.Unlock()
		return ErrNotFound
	}
	if l.HostLogin != hostLogin {
		r.mu.Unlock()
		return ErrNotHost
	}
	key := remote.Key(l.Remote, l.Branch)
	h := r.hubs[sessionID]
	delete(r.live, sessionID)
	delete(r.hubs, sessionID)
	delete(r.byKey, key)
	r.mu.Unlock()
	if h != nil {
		if err := h.Close(); err != nil {
			slog.Error("close session hub", "session", sessionID, "err", err)
		}
	}
	if err := os.RemoveAll(r.svc.GetProjectPath(sessionID)); err != nil {
		slog.Error("remove session tree", "session", sessionID, "err", err)
	}
	return nil
}

// NoteClientLeft is a no-op: sessions stay until the host ends them.
func (r *Registry) NoteClientLeft(sessionID string, remaining int) {}

// CloseProject drops a live hub (alias of process-local teardown without host check).
func (r *Registry) CloseProject(sessionID string) {
	r.mu.Lock()
	l := r.live[sessionID]
	h := r.hubs[sessionID]
	delete(r.hubs, sessionID)
	delete(r.live, sessionID)
	if l != nil {
		delete(r.byKey, remote.Key(l.Remote, l.Branch))
	}
	r.mu.Unlock()
	if h != nil {
		if err := h.Close(); err != nil {
			slog.Error("close project hub", "project", sessionID, "err", err)
		}
	}
}

// CloseAll flushes and drops every hub (server shutdown).
func (r *Registry) CloseAll() {
	r.mu.Lock()
	hubs := r.hubs
	r.hubs = make(map[string]*Hub)
	r.live = make(map[string]*Live)
	r.byKey = make(map[string]string)
	r.mu.Unlock()
	for id, h := range hubs {
		if err := h.Close(); err != nil {
			slog.Error("close hub on shutdown", "project", id, "err", err)
		}
	}
}
