package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/superfolha/internal/project"
)

func TestHubFenceAndFlush(t *testing.T) {
	state := t.TempDir()
	svc := project.NewService(state)
	projectID := "11111111-1111-1111-1111-111111111111"
	if err := svc.InitProjectRepo(projectID); err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveFile(projectID, "main.tex", "hello\n"); err != nil {
		t.Fatal(err)
	}

	h, err := Open(svc, projectID)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	if h.SessionID == "" {
		t.Fatal("missing session id")
	}
	if h.Doc.Source("main.tex") != "hello\n" {
		t.Fatalf("source = %q", h.Doc.Source("main.tex"))
	}

	c := h.AddClient("c1")
	if c.Ready {
		t.Fatal("client must not be ready before hello.ack")
	}
	if _, err := h.HandleSyncMessage("c1", []byte{0, 1, 2}); err == nil {
		t.Fatal("expected reject before ready")
	}
	if !h.MarkClientReady("c1") {
		t.Fatal("mark ready")
	}
	if !h.ClientReady("c1") {
		t.Fatal("should be ready")
	}

	// Mutate via server path and flush.
	if err := h.Doc.SetTextServer("main.tex", "world\n"); err != nil {
		t.Fatal(err)
	}
	if err := h.Flush(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(h.Root, "main.tex"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "world\n" {
		t.Fatalf("disk = %q", body)
	}

	// Sync lock blocks apply.
	h.SetSyncLocked(true)
	if _, err := h.HandleSyncMessage("c1", []byte{0}); err == nil {
		t.Fatal("expected sync locked")
	}
	h.SetSyncLocked(false)
}
