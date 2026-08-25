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
	ErrAlreadyLive = errors.New("session already live for remote and branch")
	ErrNotFound    = errors.New("session not found")
	ErrNotHost     = errors.New("not session host")
	ErrKnockClosed = errors.New("knock is disabled")
	ErrClone         = errors.New("clone failed")
	ErrUnauthorized  = errors.New("sessions.key_unauthorized")
	ErrNoWrite       = errors.New("sessions.key_read_only")
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
type CreateCloner func(dest, remoteURL, branch string, ssh igit.SessionSSH) error

// ProbePusher pushes HEAD after clone. Tests replace this.
type ProbePusher func(dest, branch string, ssh igit.SessionSSH) error

// PullTester fetches origin. Tests replace this.
type PullTester func(dest, branch string, ssh igit.SessionSSH) error

// RemoteLister lists refs on a remote. Tests replace this.
type RemoteLister func(remoteURL string, ssh igit.SessionSSH) error

// Registry holds live sessions and their hubs.
type Registry struct {
	mu     sync.Mutex
	hubs   map[string]*Hub
	live   map[string]*Live
	byKey  map[string]string
	svc    *project.Service
	cloner CreateCloner
	prober ProbePusher
	puller PullTester
	lister RemoteLister
}

// NewRegistry creates an empty hub registry.
func NewRegistry(svc *project.Service) *Registry {
	return &Registry{
		hubs:   make(map[string]*Hub),
		live:   make(map[string]*Live),
		byKey:  make(map[string]string),
		svc:    svc,
		cloner: igit.Clone,
		prober: igit.Push,
		puller: igit.Fetch,
		lister: igit.LsRemote,
	}
}

// SetCloner replaces the clone function (tests).
func (r *Registry) SetCloner(fn CreateCloner) {
	r.mu.Lock()
	r.cloner = fn
	r.mu.Unlock()
}

// SetProber replaces the probe push (tests).
func (r *Registry) SetProber(fn ProbePusher) {
	r.mu.Lock()
	r.prober = fn
	r.mu.Unlock()
}

// SetPuller replaces the post-probe fetch (tests).
func (r *Registry) SetPuller(fn PullTester) {
	r.mu.Lock()
	r.puller = fn
	r.mu.Unlock()
}

// SetLister replaces the pre-unauthorized ls-remote (tests).
func (r *Registry) SetLister(fn RemoteLister) {
	r.mu.Lock()
	r.lister = fn
	r.mu.Unlock()
}

// Create records a pending session. Clone happens on the signer socket.
func (r *Registry) Create(hostLogin, rawRemote, branch, sshPublic string) (*Live, error) {
	if err := remote.Validate(rawRemote); err != nil {
		return nil, err
	}
	_, pubLine, err := igit.ParseAuthorized(sshPublic)
	if err != nil {
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

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sessionID := id.String()
	live := &Live{
		ID:        sessionID,
		Remote:    canon,
		CloneURL:  cloneURL,
		Branch:    branch,
		HostLogin: hostLogin,
		SSHPublic: pubLine,
		admitted:  map[string]struct{}{hostLogin: {}},
		knocking:  map[string]struct{}{},
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

// CloneAndProbe clones, seeds main.tex, probe-pushes, then opens the hub.
func (r *Registry) CloneAndProbe(sessionID, hostLogin string, ssh igit.SessionSSH) error {
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
	if ssh == nil {
		return ErrNoSigner
	}
	if !SamePublic(ssh.AuthorizedKey(), l.SSHPublic) {
		return ErrNoSigner
	}
	dest := r.svc.GetProjectPath(sessionID)
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := r.cloner(dest, l.CloneURL, l.Branch, ssh); err != nil {
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			slog.Error("remove failed clone dest", "path", dest, "err", rmErr)
		}
		lsErr := r.lister(l.CloneURL, ssh)
		if lsErr != nil && (igit.AuthFailed(err) || igit.AuthFailed(lsErr)) {
			return ErrUnauthorized
		}
		return errors.Join(ErrClone, err)
	}
	if err := r.svc.EnsureMainTeX(sessionID); err != nil {
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			slog.Error("remove failed seed dest", "path", dest, "err", rmErr)
		}
		return err
	}
	if err := r.prober(dest, l.Branch, ssh); err != nil {
		pullErr := r.puller(dest, l.Branch, ssh)
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			slog.Error("remove failed probe dest", "path", dest, "err", rmErr)
		}
		if pullErr == nil {
			return ErrNoWrite
		}
		if igit.AuthFailed(err) || igit.AuthFailed(pullErr) {
			return ErrUnauthorized
		}
		return errors.Join(ErrClone, err)
	}
	h, err := Open(r.svc, sessionID, hostLogin)
	if err != nil {
		if rmErr := os.RemoveAll(dest); rmErr != nil {
			slog.Error("remove failed open dest", "path", dest, "err", rmErr)
		}
		return err
	}
	h.Branch = l.Branch
	h.SSHPublic = l.SSHPublic
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
