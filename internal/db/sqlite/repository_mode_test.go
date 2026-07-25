package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRepositorySetsOwnerOnlyMode(t *testing.T) {
	dir := t.TempDir()
	// Pre-create a world-readable file to prove we tighten mode on open.
	path := filepath.Join(dir, "superfolha.db")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := NewRepository(path)
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("db mode=%o want 0600", perm)
	}
}

func TestEnsureOwnerOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.db")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureOwnerOnlyFile(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
	if err := ensureOwnerOnlyFile(""); err != nil {
		t.Fatal(err)
	}
	if err := ensureOwnerOnlyFile(":memory:"); err != nil {
		t.Fatal(err)
	}
}
