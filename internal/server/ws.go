package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lewtec/superfolha/internal/session"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
	projectID := r.PathValue("projectId")
	if projectID == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	proj, _, user, err := s.resolver.getAndCheckProject(r.Context(), projectID)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	// Owner email for auto-commit (v1 owner-only access ⇒ user is owner).
	ownerEmail := user.Email
	if owner, err := s.repo.GetUserByID(r.Context(), proj.UserID); err == nil {
		ownerEmail = owner.Email
	}

	hub, err := s.hubs.GetOrOpen(projectID, ownerEmail)
	if err != nil {
		slog.Error("hub open", "project", projectID, "user", user.UserID, "err", err)
		http.Error(w, "failed to open project session", http.StatusInternalServerError)
		return
	}
	hub.SetOnCommitted(func() {
		// Request context may be done after the socket closes; use background.
		if err := s.repo.UpdateProjectTimestamp(context.Background(), projectID); err != nil {
			slog.Warn("project timestamp after hub commit", "project", projectID, "err", err)
		}
	})

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade", "err", err)
		return
	}

	clientID := uuid.NewString()
	client := hub.AddClient(clientID)
	defer func() {
		remaining := hub.RemoveClient(clientID)
		s.hubs.NoteClientLeft(projectID, remaining)
		_ = conn.Close()
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for out := range client.Out {
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			mt := websocket.TextMessage
			if out.Binary {
				mt = websocket.BinaryMessage
			}
			if err := conn.WriteMessage(mt, out.Data); err != nil {
				return
			}
		}
	}()

	hello, _ := json.Marshal(map[string]string{
		"type":       "hello",
		"session_id": hub.SessionID,
		"client_id":  clientID,
		"project_id": projectID,
	})
	client.Out <- session.Outbound{Data: hello}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
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
					errFrame, _ := json.Marshal(map[string]string{
						"type":    "error",
						"message": "session_id mismatch",
					})
					select {
					case client.Out <- session.Outbound{Data: errFrame}:
					default:
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
				b, _ := json.Marshal(map[string]string{"type": "pong"})
				select {
				case client.Out <- session.Outbound{Data: b}:
				default:
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

func sendWSError(c *session.Client, msg string) {
	b, _ := json.Marshal(map[string]string{"type": "error", "message": msg})
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
