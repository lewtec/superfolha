package session

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lewtec/superfolha/internal/crdt"
	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/project"
	ysync "github.com/reearth/ygo/sync"
)

const (
	flushDebounce  = 1500 * time.Millisecond
	commitDebounce = 30 * time.Second
	chatCap        = 200
	autoCommitMsg  = "Auto-commit: live session"
)

var (
	ErrHubClosed             = errors.New("hub closed")
	ErrCommitInProgress      = errors.New("commit already in progress")
	ErrClientNotSessionReady = errors.New("client not session-ready")
	ErrSyncLocked            = errors.New("sync locked")
)

// Client is a connected browser peer.
type Client struct {
	ID    string
	Out   chan Outbound
	Ready bool
}

// Outbound is a message to one client.
type Outbound struct {
	Binary bool
	Data   []byte
}

// ChatMessage is a RAM-only session chat line.
type ChatMessage struct {
	From string `json:"from"`
	Text string `json:"text"`
	At   int64  `json:"at"` // unix ms
}

// Hub is the per-project actor: CRDT text map + clients + flush/commit serialization.
type Hub struct {
	ProjectID  string
	SessionID  string
	Doc        *crdt.ProjectDoc
	Root       string
	OwnerEmail string
	Auth       *igit.HTTPAuth
	SSH        *igit.SSHKey
	Branch     string

	svc *project.Service

	mu            sync.Mutex
	clients       map[string]*Client
	flushTimer    *time.Timer
	commitTimer   *time.Timer
	unsub         func()
	syncLocked    bool
	closing       bool
	dirty         bool
	pushFailUntil time.Time
	chat          []ChatMessage
	onCommitted   func() // optional hook (e.g. touch project timestamp)
}

// Open loads project files into a new hub.
func Open(svc *project.Service, projectID, ownerEmail string) (*Hub, error) {
	files, err := crdt.ReadAllProjectFiles(svc, projectID)
	if err != nil {
		return nil, err
	}
	doc := crdt.New()
	if err := doc.LoadFromFiles(files); err != nil {
		return nil, err
	}
	h := &Hub{
		ProjectID:  projectID,
		SessionID:  projectID,
		Doc:        doc,
		Root:       svc.GetProjectPath(projectID),
		OwnerEmail: ownerEmail,
		svc:        svc,
		clients:    make(map[string]*Client),
		chat:       make([]ChatMessage, 0, 32),
	}
	h.unsub = doc.Doc.OnUpdate(func(update []byte, origin any) {
		// Load/bootstrap transactions are not live collab edits.
		if origin == crdt.OriginLoad {
			return
		}
		originID, _ := origin.(string)
		frame := ysync.EncodeUpdate(update)
		// Server-originated updates broadcast to everyone; client origin skips sender.
		skip := originID
		if originID == crdt.OriginServer {
			skip = ""
		}
		h.broadcast(frame, true, skip)
		h.scheduleFlush()
		h.markDirtyAndScheduleCommit()
		h.emitSyncStatus("dirty")
	})
	return h, nil
}

// SetOnCommitted registers a callback after a successful hub commit.
func (h *Hub) SetOnCommitted(fn func()) {
	h.mu.Lock()
	h.onCommitted = fn
	h.mu.Unlock()
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
	h.flushTimer = time.AfterFunc(flushDebounce, func() {
		if err := h.Flush(); err != nil {
			slog.Error("hub flush", "project", h.ProjectID, "err", err)
			h.emitSyncStatus("flush_error")
			return
		}
		h.emitSyncStatus("synced")
	})
}

func (h *Hub) markDirtyAndScheduleCommit() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closing {
		return
	}
	h.dirty = true
	if h.commitTimer != nil {
		h.commitTimer.Stop()
	}
	wait := commitDebounce
	if left := time.Until(h.pushFailUntil); left > wait {
		wait = left
	}
	h.commitTimer = time.AfterFunc(wait, func() {
		if _, err := h.Commit(autoCommitMsg, ""); err != nil {
			slog.Error("hub auto-commit", "project", h.ProjectID, "err", err)
			h.emitSyncStatus("commit_error")
			return
		}
		h.emitSyncStatus("committed")
	})
}

// Flush writes collaborative text to the project working tree.
func (h *Hub) Flush() error {
	h.mu.Lock()
	doc := h.Doc
	root := h.Root
	h.mu.Unlock()
	if doc == nil {
		return ErrHubClosed
	}
	return doc.FlushToDir(root)
}

// Commit flushes text, locks CRDT sync, commits git, unlocks.
// author empty → OwnerEmail.
func (h *Hub) Commit(message, author string) (string, error) {
	if message == "" {
		message = autoCommitMsg
	}
	if author == "" {
		author = h.OwnerEmail
	}
	if author == "" {
		author = "superfolha"
	}

	h.mu.Lock()
	if h.syncLocked {
		h.mu.Unlock()
		return "", ErrCommitInProgress
	}
	h.syncLocked = true
	h.emitSyncStatusLocked("committing")
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.syncLocked = false
		h.mu.Unlock()
	}()

	if err := h.Flush(); err != nil {
		return "", err
	}

	c, err := h.svc.CommitChanges(h.ProjectID, author, message)
	if err != nil {
		return "", err
	}
	if h.Auth != nil || h.SSH != nil {
		if err := igit.Push(h.Root, h.Branch, h.Auth, h.SSH); err != nil {
			pub := ""
			if h.SSH != nil {
				pub = h.SSH.Authorized
			}
			h.mu.Lock()
			h.pushFailUntil = time.Now().Add(commitDebounce)
			h.mu.Unlock()
			h.broadcastJSONMap(map[string]any{
				"type":       "push.error",
				"message":    err.Error(),
				"ssh_public": pub,
			}, "")
			return "", err
		}
	}
	h.mu.Lock()
	h.dirty = false
	if h.commitTimer != nil {
		h.commitTimer.Stop()
		h.commitTimer = nil
	}
	cb := h.onCommitted
	h.mu.Unlock()
	if cb != nil {
		cb()
	}
	h.broadcastJSONMap(map[string]any{
		"type":    "commit.done",
		"hash":    c.Hash,
		"message": message,
		"author":  author,
	}, "")
	return c.Hash, nil
}

// SaveTextFile writes text to CRDT + disk (common path for GraphQL/WS).
func (h *Hub) SaveTextFile(path, content string) error {
	path = filepath.ToSlash(path)
	if _, err := project.ValidateRepoRelativePath(path); err != nil {
		return err
	}
	if project.IsBinary([]byte(content), path) {
		// Blobs should use disk-only upload path.
		return h.svc.SaveFile(h.ProjectID, path, content)
	}
	if int64(len(content)) > project.MaxCollabTextBytes {
		return h.svc.SaveFile(h.ProjectID, path, content)
	}
	if err := h.Doc.SetTextServer(path, content); err != nil {
		return err
	}
	// Immediate disk write for structure/content consistency (also scheduled flush).
	return h.svc.SaveFile(h.ProjectID, path, content)
}

// DeleteFile removes path from disk and CRDT text map.
func (h *Hub) DeleteFile(path string) error {
	path = filepath.ToSlash(path)
	if err := h.Doc.RemoveText(path); err != nil {
		return err
	}
	if err := h.svc.DeleteFile(h.ProjectID, path); err != nil {
		return err
	}
	h.broadcastJSONMap(map[string]any{
		"type": "tree.event",
		"op":   "delete",
		"path": path,
	}, "")
	h.markDirtyAndScheduleCommit()
	return nil
}

// CreateTextFile creates an empty/collaborative text file.
func (h *Hub) CreateTextFile(path, content string) error {
	path = filepath.ToSlash(path)
	if err := h.SaveTextFile(path, content); err != nil {
		return err
	}
	h.broadcastJSONMap(map[string]any{
		"type": "tree.event",
		"op":   "create",
		"path": path,
		"kind": "text",
	}, "")
	return nil
}

// SyncLocked reports whether CRDT apply is blocked.
func (h *Hub) SyncLocked() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.syncLocked
}

// DisconnectAll closes every peer channel (kick everyone).
func (h *Hub) DisconnectAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, c := range h.clients {
		close(c.Out)
		delete(h.clients, id)
	}
}

// AddClient registers a peer (not Ready until hello.ack).
func (h *Hub) AddClient(id string) *Client {
	c := &Client{ID: id, Out: make(chan Outbound, 64)}
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
func (h *Hub) HandleSyncMessage(clientID string, msg []byte) ([]byte, error) {
	if !h.ClientReady(clientID) {
		return nil, ErrClientNotSessionReady
	}
	if h.SyncLocked() {
		return nil, ErrSyncLocked
	}
	reply, err := ysync.ApplySyncMessage(h.Doc.Doc, msg, clientID)
	if err == nil {
		// Client updates that apply successfully are acked via protocol reply;
		// UI "Synced" follows next flush/broadcast of status.
		h.emitSyncStatus("synced")
	}
	return reply, err
}

// EncodeSyncStep1 for server-initiated handshake after hello.ack.
func (h *Hub) EncodeSyncStep1() []byte {
	return ysync.EncodeSyncStep1(h.Doc.Doc)
}

// EncodeFullStateUpdate wraps the entire ygo document as a y-protocols Update
// frame so a freshly connected empty client receives all collaborative text.
func (h *Hub) EncodeFullStateUpdate() []byte {
	raw := h.Doc.EncodeStateAsUpdate()
	if len(raw) == 0 {
		return nil
	}
	return ysync.EncodeUpdate(raw)
}

// AppendChat stores and fans out a chat message.
func (h *Hub) AppendChat(from, text string) {
	if text == "" {
		return
	}
	msg := ChatMessage{From: from, Text: text, At: time.Now().UnixMilli()}
	h.mu.Lock()
	h.chat = append(h.chat, msg)
	if len(h.chat) > chatCap {
		h.chat = h.chat[len(h.chat)-chatCap:]
	}
	// copy for send outside lock
	payload, err := json.Marshal(map[string]any{
		"type": "chat.message",
		"from": msg.From,
		"text": msg.Text,
		"at":   msg.At,
	})
	h.mu.Unlock()
	if err != nil {
		slog.Error("marshal chat.message", "project", h.ProjectID, "err", err)
		return
	}
	h.broadcast(payload, false, "")
}

// ChatHistory returns a copy of the ring buffer.
func (h *Hub) ChatHistory() []ChatMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]ChatMessage, len(h.chat))
	copy(out, h.chat)
	return out
}

// SendChatHistory sends chat.history to one client.
func (h *Hub) SendChatHistory(c *Client) {
	hist := h.ChatHistory()
	b, err := json.Marshal(map[string]any{
		"type":     "chat.history",
		"messages": hist,
	})
	if err != nil {
		slog.Error("marshal chat.history", "project", h.ProjectID, "err", err)
		return
	}
	select {
	case c.Out <- Outbound{Data: b}:
	default:
	}
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

func (h *Hub) broadcastJSONMap(m map[string]any, skip string) {
	b, err := json.Marshal(m)
	if err != nil {
		slog.Error("marshal hub broadcast", "project", h.ProjectID, "err", err)
		return
	}
	h.broadcast(b, false, skip)
}

func (h *Hub) emitSyncStatus(status string) {
	h.mu.Lock()
	h.emitSyncStatusLocked(status)
	h.mu.Unlock()
}

func (h *Hub) emitSyncStatusLocked(status string) {
	b, err := json.Marshal(map[string]string{"type": "sync.status", "status": status})
	if err != nil {
		slog.Error("marshal sync.status", "project", h.ProjectID, "err", err)
		return
	}
	for _, c := range h.clients {
		if !c.Ready {
			continue
		}
		select {
		case c.Out <- Outbound{Data: b}:
		default:
		}
	}
}

// Close flushes, optional final commit, unsubscribes, closes clients.
// If the project root is already gone (e.g. deleteProject removed the tree
// before a late idle close), skip flush/commit so Close cannot resurrect
// the directory via FlushToDir or an unconditional MkdirAll.
func (h *Hub) Close() error {
	h.mu.Lock()
	h.closing = true
	if h.flushTimer != nil {
		h.flushTimer.Stop()
		h.flushTimer = nil
	}
	if h.commitTimer != nil {
		h.commitTimer.Stop()
		h.commitTimer = nil
	}
	if h.unsub != nil {
		h.unsub()
		h.unsub = nil
	}
	for id, c := range h.clients {
		close(c.Out)
		delete(h.clients, id)
	}
	dirty := h.dirty
	doc := h.Doc
	root := h.Root
	h.mu.Unlock()

	if root != "" {
		if _, err := os.Stat(root); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
	}

	if doc != nil {
		if err := doc.FlushToDir(root); err != nil {
			return err
		}
	}
	if dirty {
		if _, err := h.Commit(autoCommitMsg, ""); err != nil {
			// Best-effort final commit; still close.
			slog.Warn("final hub commit on close", "project", h.ProjectID, "err", err)
		}
	}
	return nil
}
