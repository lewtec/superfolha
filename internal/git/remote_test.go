package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

func seedRepo(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := InitRepo(dir); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := CommitChanges(dir, "alice@example.com", "seed"); err != nil {
		t.Fatalf("CommitChanges: %v", err)
	}
	return dir
}

func TestCloneAndCloneHTTPFileRepo(t *testing.T) {
	src := seedRepo(t, "hello")
	url := "file://" + src

	destSSH := filepath.Join(t.TempDir(), "ssh")
	if err := Clone(destSSH, url, "", nil); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	destHTTP := filepath.Join(t.TempDir(), "http")
	if err := CloneHTTP(destHTTP, url, "", nil); err != nil {
		t.Fatalf("CloneHTTP: %v", err)
	}

	gotSSH, err := os.ReadFile(filepath.Join(destSSH, "note.txt"))
	if err != nil {
		t.Fatalf("read ssh clone: %v", err)
	}
	gotHTTP, err := os.ReadFile(filepath.Join(destHTTP, "note.txt"))
	if err != nil {
		t.Fatalf("read http clone: %v", err)
	}
	if string(gotSSH) != "hello" || string(gotHTTP) != "hello" {
		t.Fatalf("clone contents ssh=%q http=%q, want hello", gotSSH, gotHTTP)
	}
}

func TestPushAndPushOriginFileRepo(t *testing.T) {
	src := seedRepo(t, "v1")
	bare := filepath.Join(t.TempDir(), "bare.git")
	if _, err := gogit.PlainClone(bare, true, &gogit.CloneOptions{URL: "file://" + src}); err != nil {
		t.Fatalf("bare clone: %v", err)
	}

	work := filepath.Join(t.TempDir(), "work")
	if err := Clone(work, "file://"+bare, "", nil); err != nil {
		t.Fatalf("Clone work: %v", err)
	}
	if err := os.WriteFile(filepath.Join(work, "note.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := CommitChanges(work, "alice@example.com", "update"); err != nil {
		t.Fatalf("CommitChanges: %v", err)
	}
	if err := Push(work, "", nil); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if err := PushOrigin(work, "", nil); err != nil {
		t.Fatalf("PushOrigin already-up-to-date: %v", err)
	}

	check := filepath.Join(t.TempDir(), "check")
	if err := CloneHTTP(check, "file://"+bare, "", nil); err != nil {
		t.Fatalf("CloneHTTP check: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(check, "note.txt"))
	if err != nil {
		t.Fatalf("read check: %v", err)
	}
	if string(got) != "v2" {
		t.Fatalf("pushed content = %q, want v2", got)
	}
}
