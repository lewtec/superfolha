package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/superfolha/internal/crdt"
	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/project"
	ycrdt "github.com/reearth/ygo/crdt"
	ysync "github.com/reearth/ygo/sync"
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

	h, err := Open(svc, projectID, "owner@example.com")
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
	_ = c

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

	h.mu.Lock()
	h.syncLocked = true
	h.mu.Unlock()
	if _, err := h.HandleSyncMessage("c1", []byte{0}); err == nil {
		t.Fatal("expected sync locked")
	}
	h.mu.Lock()
	h.syncLocked = false
	h.mu.Unlock()

	if err := h.CreateTextFile("extra.tex", "x\n"); err != nil {
		t.Fatal(err)
	}
	if h.Doc.Source("extra.tex") != "x\n" {
		t.Fatalf("extra = %q", h.Doc.Source("extra.tex"))
	}
}

func TestHubBootstrapFullStateAndClientStep1(t *testing.T) {
	state := t.TempDir()
	svc := project.NewService(state)
	projectID := "22222222-2222-2222-2222-222222222222"
	if err := svc.InitProjectRepo(projectID); err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveFile(projectID, "main.tex", "\\title{Loaded}\n"); err != nil {
		t.Fatal(err)
	}

	h, err := Open(svc, projectID, "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	full := h.EncodeFullStateUpdate()
	if len(full) == 0 {
		t.Fatal("EncodeFullStateUpdate empty — clients would see empty docs")
	}

	// Empty peer applies the Update frame via ApplySyncMessage.
	fromFull := ycrdt.New()
	if _, err := ysync.ApplySyncMessage(fromFull, full, "remote"); err != nil {
		t.Fatalf("apply full update: %v", err)
	}
	if got := fromFull.GetText(crdt.TextKey("main.tex")).ToString(); got != "\\title{Loaded}\n" {
		t.Fatalf("after full update main.tex = %q", got)
	}

	// Empty client SyncStep1 → server must return SyncStep2 that fills the client.
	h.AddClient("c2")
	if !h.MarkClientReady("c2") {
		t.Fatal("mark ready c2")
	}
	emptyPeer := ycrdt.New()
	step1 := ysync.EncodeSyncStep1(emptyPeer)
	reply, err := h.HandleSyncMessage("c2", step1)
	if err != nil {
		t.Fatalf("HandleSyncMessage step1: %v", err)
	}
	if len(reply) == 0 {
		t.Fatal("server SyncStep2 reply empty")
	}
	if _, err := ysync.ApplySyncMessage(emptyPeer, reply, "remote"); err != nil {
		t.Fatalf("apply step2: %v", err)
	}
	if got := emptyPeer.GetText(crdt.TextKey("main.tex")).ToString(); got != "\\title{Loaded}\n" {
		t.Fatalf("after step2 main.tex = %q", got)
	}
}

func TestRegistryCloseProject(t *testing.T) {
	state := t.TempDir()
	svc := project.NewService(state)
	reg := NewRegistry(svc)
	reg.SetCloner(func(dest, _, _ string, _ *igit.HTTPAuth, _ *igit.SSHKey) error {
		if err := igit.InitRepo(dest); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, "main.tex"), []byte("body\n"), 0o644)
	})
	live, err := reg.Create("owner@example.com", "https://github.com/t/paper", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	projectID := live.ID
	h, err := reg.GetOrOpen(projectID, "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if reg.GetIfLive(projectID) != h {
		t.Fatal("hub should be live after Create")
	}
	c := h.AddClient("c1")
	if c == nil {
		t.Fatal("client")
	}

	reg.CloseProject(projectID)
	if reg.GetIfLive(projectID) != nil {
		t.Fatal("hub must not remain live after CloseProject")
	}
	// Peer channel closed so WS writers stop.
	if _, ok := <-c.Out; ok {
		t.Fatal("client Out should be closed after CloseProject")
	}

	// Safe to call again.
	reg.CloseProject(projectID)
}
