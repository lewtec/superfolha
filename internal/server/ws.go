package server

import (
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
		// Same-origin SPA + JWT cookie; tighten if multi-origin deploys appear.
		return true
	},
}

type wsControl struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Text      string `json:"text,omitempty"`
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

	_, _, user, err := s.resolver.getAndCheckProject(r.Context(), projectID)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	hub, err := s.hubs.GetOrOpen(projectID)
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

	// hello only — no CRDT until hello.ack (session fence).
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
				// Tree snapshot + Yjs sync step1 after fence.
				if snap, err := s.treeSnapshotJSON(projectID); err == nil {
					select {
					case client.Out <- session.Outbound{Data: snap}:
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
				b, _ := json.Marshal(map[string]string{"type": "pong"})
				select {
				case client.Out <- session.Outbound{Data: b}:
				default:
				}
			case "chat.send":
				b, _ := json.Marshal(map[string]any{
					"type": "chat.message",
					"text": ctrl.Text,
					"from": user.Email,
				})
				hub.BroadcastJSON(b, "")
			case "awareness":
				hub.BroadcastJSON(data, clientID)
			}
		}
	}
	<-done
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
