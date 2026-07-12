package git

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCommitChangesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := InitRepo(dir); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	c1, err := CommitChanges(dir, "alice@example.com", "initial")
	if err != nil {
		t.Fatalf("CommitChanges dirty: %v", err)
	}
	if c1.Hash == "" {
		t.Fatal("expected non-empty commit hash")
	}
	if c1.Message != "initial" {
		t.Fatalf("Message = %q, want %q", c1.Message, "initial")
	}
	// Author.String() is "Name <Email>"; both are set to the author string.
	if c1.Author != "alice@example.com <alice@example.com>" {
		t.Fatalf("Author = %q, want name and email both set to author string", c1.Author)
	}

	// Clean tree returns the last commit (noop).
	c2, err := CommitChanges(dir, "bob@example.com", "noop")
	if err != nil {
		t.Fatalf("CommitChanges clean: %v", err)
	}
	if c2.Hash != c1.Hash {
		t.Fatalf("clean commit hash = %s, want last commit %s", c2.Hash, c1.Hash)
	}
	if c2.Message != "initial" {
		t.Fatalf("clean Message = %q, want last commit message %q", c2.Message, "initial")
	}
}

func TestReadFileNotFound(t *testing.T) {
	dir := t.TempDir()
	if err := InitRepo(dir); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	_, _, err := ReadFile(dir, "missing.txt")
	if !errors.Is(err, ErrGitFileNotFound) {
		t.Fatalf("ReadFile missing: err = %v, want ErrGitFileNotFound", err)
	}
}

func TestReadFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := InitRepo(dir); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}

	path := filepath.Join(dir, "data.txt")
	want := []byte("payload")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := CommitChanges(dir, "alice@example.com", "add data"); err != nil {
		t.Fatalf("CommitChanges: %v", err)
	}

	rc, size, err := ReadFile(dir, "data.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if size != int64(len(want)) {
		t.Fatalf("size = %d, want %d", size, len(want))
	}
	if string(got) != string(want) {
		t.Fatalf("content = %q, want %q", got, want)
	}
}
