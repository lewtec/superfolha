package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/superfolha/internal/project"
)

// Close must not recreate a project tree that was already removed (deleteProject
// race / late idle eviction after RemoveAll).
func TestHubCloseDoesNotRecreateRemovedRoot(t *testing.T) {
	state := t.TempDir()
	svc := project.NewService(state)
	projectID := "33333333-3333-3333-3333-333333333333"
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
	root := h.Root
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("expected root before delete: %v", err)
	}

	// Simulate deleteProject RemoveAll after hub was opened.
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("root should be gone, stat err=%v", err)
	}

	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("Close recreated project root %s (stat err=%v)", root, err)
	}
	// Parent repos/ dir may still exist; ensure no project subtree returned.
	if _, err := os.Stat(filepath.Join(root, "main.tex")); !os.IsNotExist(err) {
		t.Fatalf("Close recreated project files under %s", root)
	}
}

// Normal close still flushes collaborative text when the tree remains.
func TestHubCloseFlushesWhenRootExists(t *testing.T) {
	state := t.TempDir()
	svc := project.NewService(state)
	projectID := "44444444-4444-4444-4444-444444444444"
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
	if err := h.Doc.SetTextServer("main.tex", "closed\n"); err != nil {
		t.Fatal(err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(h.Root, "main.tex"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "closed\n" {
		t.Fatalf("disk after Close = %q", body)
	}
}
