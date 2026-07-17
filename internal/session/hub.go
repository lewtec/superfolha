package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/crdt"
	"github.com/lewtec/superfolha/internal/project"
	ysync "github.com/reearth/ygo/sync"
)

// Client is a connected browser peer.
type Client struct {
	ID string
	Out chan Outbound
	// Ready is true after the client has acked hello for this hub session.
	Ready bool
}

// Outbound is a message to one client.
type Outbound struct {
	Binary bool
	Data   []byte
}

// Hub is the per-project actor: CRDT text map + clients + flush/commit serialization.
type Hub struct {
	ProjectID string
	SessionID string
	Doc       *crdt.ProjectDoc
	Root      string // absolute path to project git working tree

	svc *project.Service

	mu          sync.Mutex
	clients     map[string]*Client
	flushTimer  *time.Timer
	unsub       func()
	syncLocked  bool // true while commit is in progress
	closing     bool
}

// Open loads project files into a new hub.
func Open(svc *project.Service, projectID string) (*Hub, error) {
	files, err := crdt.ReadAllProjectFiles(svc, projectID)
	if err != nil {
		return nil, err
	}
	doc := crdt.New()
	if err := doc.LoadFromFiles(files); err != nil {
		return nil, err
	}
	h := &Hub{
		ProjectID: projectID,
		SessionID: uuid.NewString(),
		Doc:       doc,
		Root:      svc.GetProjectPath(projectID),
		svc:       svc,
		clients:   make(map[string]*Client),
	}
	h.unsub = doc.Doc.OnUpdate(func(update []byte, origin any) {
		originID, _ := origin.(string)
		frame := ysync.EncodeUpdate(update)
		h.broadcast(frame, true, originID)
		h.scheduleFlush()
	})
	return h, nil
}

func (h *Hub) scheduleFlush() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closing {
		return
	}
	if h.flushTimer != nil {
		h.flushTimer.Stop()
	}
	// SPEC default ~1–2s; use 1.5s mid-range.
	h.flushTimer = time.AfterFunc(1500*time.Millisecond, func() {
		_ = h.Flush()
	})
}

// Flush writes collaborative text to the project working tree.
func (h *Hub) Flush() error {
	h.mu.Lock()
	doc := h.Doc
	root := h.Root
	h.mu.Unlock()
	if doc == nil {
		return fmt.Errorf("hub closed")
	}
	return doc.FlushToDir(root)
}

// SyncLocked reports whether CRDT apply is blocked (e.g. commit in progress).
func (h *Hub) SyncLocked() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.syncLocked
}

// SetSyncLocked freezes or unfreezes CRDT apply (commit window).
func (h *Hub) SetSyncLocked(v bool) {
	h.mu.Lock()
	h.syncLocked = v
	h.mu.Unlock()
}

// AddClient registers a peer (not Ready until hello.ack).
func (h *Hub) AddClient(id string) *Client {
	c := &Client{
		ID:  id,
		Out: make(chan Outbound, 64),
	}
	h.mu.Lock()
	h.clients[id] = c
	h.mu.Unlock()
	return c
}

// RemoveClient unregisters a peer. Returns remaining client count.
func (h *Hub) RemoveClient(id string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.clients[id]; ok {
		close(c.Out)
		delete(h.clients, id)
	}
	return len(h.clients)
}

// ClientCount returns connected peers.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// MarkClientReady marks the peer as having accepted this hub session_id.
func (h *Hub) MarkClientReady(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.clients[id]
	if !ok {
		return false
	}
	c.Ready = true
	return true
}

// ClientReady reports whether the peer may apply CRDT/control.
func (h *Hub) ClientReady(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.clients[id]
	return ok && c.Ready
}

// HandleSyncMessage applies a y-protocols sync frame.
// Peers that have not acked hello are ignored. Applies are dropped while syncLocked.
func (h *Hub) HandleSyncMessage(clientID string, msg []byte) ([]byte, error) {
	if !h.ClientReady(clientID) {
		return nil, fmt.Errorf("client not session-ready")
	}
	if h.SyncLocked() {
		return nil, fmt.Errorf("sync locked")
	}
	return ysync.ApplySyncMessage(h.Doc.Doc, msg, clientID)
}

// EncodeSyncStep1 for server-initiated handshake after hello.ack.
func (h *Hub) EncodeSyncStep1() []byte {
	return ysync.EncodeSyncStep1(h.Doc.Doc)
}

func (h *Hub) broadcast(data []byte, binary bool, skipClientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, c := range h.clients {
		if id == skipClientID || !c.Ready {
			continue
		}
		select {
		case c.Out <- Outbound{Binary: binary, Data: data}:
		default:
		}
	}
}

// BroadcastJSON sends a text frame to all ready clients (or all but skip).
func (h *Hub) BroadcastJSON(data []byte, skipClientID string) {
	h.broadcast(data, false, skipClientID)
}

// Close flushes, unsubscribes, and closes client channels.
func (h *Hub) Close() error {
	h.mu.Lock()
	h.closing = true
	if h.flushTimer != nil {
		h.flushTimer.Stop()
		h.flushTimer = nil
	}
	if h.unsub != nil {
		h.unsub()
		h.unsub = nil
	}
	for id, c := range h.clients {
		close(c.Out)
		delete(h.clients, id)
	}
	doc := h.Doc
	root := h.Root
	h.mu.Unlock()
	if doc != nil {
		return doc.FlushToDir(root)
	}
	return nil
}
