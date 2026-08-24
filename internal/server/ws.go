package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lewtec/superfolha/internal/appenv"
	"github.com/lewtec/superfolha/internal/paths"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/session"
)

// wsMaxMessageBytes caps a single WebSocket frame. Collaborative text is limited
// by project.MaxCollabTextBytes; leave headroom for y-protocols framing and
// multi-file state vectors without allowing unbounded memory use per message.
const wsMaxMessageBytes = project.MaxCollabTextBytes + 1<<20 // 6 MiB

// wsCheckOrigin rejects cross-site browser WebSocket handshakes in production.
// Cookie-authenticated collab sockets would otherwise be open to CSWSH if
// CheckOrigin always returned true. Development (GO_ENV=development) allows all
// origins so the Vite dev proxy (different port) can connect.
func wsCheckOrigin(r *http.Request) bool {
	if appenv.IsDevelopment() {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser clients often omit Origin; auth still required after upgrade.
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
	CheckOrigin:     wsCheckOrigin,
}

type wsControl struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Text      string `json:"text,omitempty"`
	Path      string `json:"path,omitempty"`
	Content   string `json:"content,omitempty"`
	Message   string `json:"message,omitempty"`
	Update    string `json:"update,omitempty"` // awareness base64
}

func (s *Server) handleProjectWS(w http.ResponseWriter, r *http.Request) {
	if s.hubs == nil {
		http.Error(w, "collaboration unavailable", http.StatusServiceUnavailable)
		return
	}
	projectID := r.PathValue(paths.ParamID)
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	proj, _, user, err := s.resolver.getAndCheckProject(r.Context(), projectID)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	hub, err := s.hubs.GetOrOpen(projectID, proj.UserID)
	if err != nil {
		slog.Error("hub open", "project", projectID, "user", user.UserID, "err", err)
		http.Error(w, "failed to open project session", http.StatusInternalServerError)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade", "err", err)
		return
	}
	conn.SetReadLimit(wsMaxMessageBytes)

	clientID := uuid.NewString()
	client := hub.AddClient(clientID)
	defer func() {
		remaining := hub.RemoveClient(clientID)
		s.hubs.NoteClientLeft(projectID, remaining)
		if err := conn.Close(); err != nil {
			slog.Debug("ws close", "project", projectID, "client", clientID, "err", err)
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for out := range client.Out {
			if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				slog.Debug("ws write deadline", "project", projectID, "err", err)
				return
			}
			mt := websocket.TextMessage
			if out.Binary {
				mt = websocket.BinaryMessage
			}
			if err := conn.WriteMessage(mt, out.Data); err != nil {
				return
			}
		}
	}()

	if hello := marshalWSJSON(map[string]string{
		"type":       "hello",
		"session_id": hub.SessionID,
		"client_id":  clientID,
		"project_id": projectID,
	}); hello != nil {
		client.Out <- session.Outbound{Data: hello}
	}

	for {
		if err := conn.SetReadDeadline(time.Now().Add(120 * time.Second)); err != nil {
			slog.Debug("ws read deadline", "project", projectID, "err", err)
			break
		}
		mt, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		switch mt {
		case websocket.BinaryMessage:
			reply, err := hub.HandleSyncMessage(clientID, data)
			if err != nil {
				continue
			}
			if reply != nil {
				select {
				case client.Out <- session.Outbound{Binary: true, Data: reply}:
				default:
				}
			}
		case websocket.TextMessage:
			var ctrl wsControl
			if err := json.Unmarshal(data, &ctrl); err != nil {
				continue
			}
			if ctrl.Type == "hello.ack" {
				if ctrl.SessionID != "" && ctrl.SessionID != hub.SessionID {
					if errFrame := marshalWSJSON(map[string]string{
						"type":    "error",
						"message": "session_id mismatch",
					}); errFrame != nil {
						select {
						case client.Out <- session.Outbound{Data: errFrame}:
						default:
						}
					}
					continue
				}
				if !hub.MarkClientReady(clientID) {
					continue
				}
				if snap, err := s.treeSnapshotJSON(projectID); err == nil {
					select {
					case client.Out <- session.Outbound{Data: snap}:
					default:
					}
				}
				hub.SendChatHistory(client)
				// Push full CRDT state immediately (bootstrap), then SyncStep1 so
				// the client can still complete a normal two-way handshake.
				// SyncStep1 alone only carries a state vector — an empty client
				// would reply with empty SyncStep2 and never receive file text.
				if full := hub.EncodeFullStateUpdate(); len(full) > 0 {
					select {
					case client.Out <- session.Outbound{Binary: true, Data: full}:
					default:
					}
				}
				select {
				case client.Out <- session.Outbound{Binary: true, Data: hub.EncodeSyncStep1()}:
				default:
				}
				continue
			}
			if !hub.ClientReady(clientID) {
				continue
			}
			switch ctrl.Type {
			case "ping":
				if b := marshalWSJSON(map[string]string{"type": "pong"}); b != nil {
					select {
					case client.Out <- session.Outbound{Data: b}:
					default:
					}
				}
			case "chat.send":
				hub.AppendChat(user.Email, ctrl.Text)
			case "awareness":
				hub.BroadcastJSON(data, clientID)
			case "file.create":
				if ctrl.Path == "" {
					sendWSError(client, "path required")
					continue
				}
				if err := hub.CreateTextFile(ctrl.Path, ctrl.Content); err != nil {
					sendWSError(client, err.Error())
				}
			case "file.delete":
				if ctrl.Path == "" {
					sendWSError(client, "path required")
					continue
				}
				if err := hub.DeleteFile(ctrl.Path); err != nil {
					sendWSError(client, err.Error())
				}
			case "commit.now":
				msg := ctrl.Message
				if msg == "" {
					msg = "Manual commit"
				}
				if _, err := hub.Commit(msg, user.Email); err != nil {
					sendWSError(client, err.Error())
				}
			}
		}
	}
	<-done
}

func marshalWSJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("ws marshal", "err", err)
		return nil
	}
	return b
}

func sendWSError(c *session.Client, msg string) {
	b := marshalWSJSON(map[string]string{"type": "error", "message": msg})
	if b == nil {
		return
	}
	select {
	case c.Out <- session.Outbound{Data: b}:
	default:
	}
}

func (s *Server) treeSnapshotJSON(projectID string) ([]byte, error) {
	files, err := s.projectService.ListFiles(projectID)
	if err != nil {
		return nil, err
	}
	type entry struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
	}
	entries := make([]entry, 0, len(files))
	for _, f := range files {
		entries = append(entries, entry{Path: f.Path, Size: f.Size})
	}
	return json.Marshal(map[string]any{
		"type":  "tree.snapshot",
		"files": entries,
	})
}
