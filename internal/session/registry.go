package session

import (
	"sync"
	"time"

	"github.com/lewtec/superfolha/internal/project"
)

// DefaultIdleTTL is how long a hub stays alive with zero clients (SPEC ~30m).
const DefaultIdleTTL = 30 * time.Minute

// Registry holds live project hubs.
type Registry struct {
	mu      sync.Mutex
	hubs    map[string]*Hub
	idle    map[string]*time.Timer
	svc     *project.Service
	idleTTL time.Duration
}

// NewRegistry creates an empty hub registry.
func NewRegistry(svc *project.Service) *Registry {
	return &Registry{
		hubs:    make(map[string]*Hub),
		idle:    make(map[string]*time.Timer),
		svc:     svc,
		idleTTL: DefaultIdleTTL,
	}
}

// GetIfLive returns a hub only if it is already open.
func (r *Registry) GetIfLive(projectID string) *Hub {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hubs[projectID]
}

// GetOrOpen returns an existing hub or opens one and loads the project tree.
func (r *Registry) GetOrOpen(projectID, ownerEmail string) (*Hub, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t := r.idle[projectID]; t != nil {
		t.Stop()
		delete(r.idle, projectID)
	}
	if h, ok := r.hubs[projectID]; ok {
		if ownerEmail != "" && h.OwnerEmail == "" {
			h.OwnerEmail = ownerEmail
		}
		return h, nil
	}
	h, err := Open(r.svc, projectID, ownerEmail)
	if err != nil {
		return nil, err
	}
	r.hubs[projectID] = h
	return h, nil
}

// NoteClientLeft schedules idle eviction when the last client disconnects.
func (r *Registry) NoteClientLeft(projectID string, remaining int) {
	if remaining > 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if t := r.idle[projectID]; t != nil {
		t.Stop()
	}
	r.idle[projectID] = time.AfterFunc(r.idleTTL, func() {
		r.evict(projectID)
	})
}

func (r *Registry) evict(projectID string) {
	r.mu.Lock()
	h, ok := r.hubs[projectID]
	if !ok {
		r.mu.Unlock()
		return
	}
	if h.ClientCount() > 0 {
		delete(r.idle, projectID)
		r.mu.Unlock()
		return
	}
	delete(r.hubs, projectID)
	if t := r.idle[projectID]; t != nil {
		t.Stop()
		delete(r.idle, projectID)
	}
	r.mu.Unlock()
	_ = h.Close()
}

// CloseProject drops a live hub for projectID (flush + disconnect peers).
// No-op when the hub is not open. Call before deleting the project tree so
// idle eviction or a later flush cannot recreate files under a removed path.
func (r *Registry) CloseProject(projectID string) {
	r.mu.Lock()
	if t := r.idle[projectID]; t != nil {
		t.Stop()
		delete(r.idle, projectID)
	}
	h, ok := r.hubs[projectID]
	if ok {
		delete(r.hubs, projectID)
	}
	r.mu.Unlock()
	if !ok {
		return
	}
	_ = h.Close()
}

// CloseAll flushes and drops every hub (server shutdown).
func (r *Registry) CloseAll() {
	r.mu.Lock()
	for id, t := range r.idle {
		t.Stop()
		delete(r.idle, id)
	}
	hubs := r.hubs
	r.hubs = make(map[string]*Hub)
	r.mu.Unlock()
	for _, h := range hubs {
		_ = h.Close()
	}
}
