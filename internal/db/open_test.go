package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	moderncsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestOpenRepositorySetsOwnerOnlyMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "superfolha.db")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenRepository(t.Context(), path)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Errorf("repo.Close(): %v", err)
		}
	})

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("db mode=%o want 0600", perm)
	}
}

func TestEnsureOwnerOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
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

func TestOpenRepositoryEmptyPath(t *testing.T) {
	_, err := OpenRepository(t.Context(), "")
	if !errors.Is(err, ErrEmptyPath) {
		t.Fatalf("error = %v, want %v", err, ErrEmptyPath)
	}
}

func TestIsUniqueViolation_EmailConflict(t *testing.T) {
	repo, err := OpenRepository(t.Context(), filepath.Join(t.TempDir(), "unique.db"))
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Errorf("repo.Close(): %v", err)
		}
	})

	ctx := t.Context()
	_, err = repo.CreateUser(ctx, CreateUserParams{
		ID: "user-1", Email: "dup@example.com", PasswordHash: "hash-a",
	})
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	_, err = repo.CreateUser(ctx, CreateUserParams{
		ID: "user-2", Email: "dup@example.com", PasswordHash: "hash-b",
	})
	if err == nil {
		t.Fatal("second CreateUser with same email: want unique violation, got nil")
	}
	if !repo.IsUniqueViolation(err) {
		t.Fatalf("IsUniqueViolation(email dup)=false, err=%v", err)
	}

	if !repo.IsUniqueViolation(fmt.Errorf("create user: %w", err)) {
		t.Fatal("IsUniqueViolation should unwrap via errors.As")
	}

	var se *moderncsqlite.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *sqlite.Error, got %T: %v", err, err)
	}
	if se.Code() != sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		t.Fatalf("Code()=%d, want SQLITE_CONSTRAINT_UNIQUE (%d)", se.Code(), sqlite3.SQLITE_CONSTRAINT_UNIQUE)
	}
}

func TestIsUniqueViolation_PrimaryKey(t *testing.T) {
	repo, err := OpenRepository(t.Context(), filepath.Join(t.TempDir(), "pk.db"))
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Errorf("repo.Close(): %v", err)
		}
	})

	ctx := t.Context()
	if _, err := repo.CreateUser(ctx, CreateUserParams{
		ID: "same-id", Email: "a@example.com", PasswordHash: "h",
	}); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	_, err = repo.CreateUser(ctx, CreateUserParams{
		ID: "same-id", Email: "b@example.com", PasswordHash: "h",
	})
	if err == nil {
		t.Fatal("duplicate primary key: want error")
	}
	if !repo.IsUniqueViolation(err) {
		t.Fatalf("IsUniqueViolation(pk dup)=false, err=%v", err)
	}

	var se *moderncsqlite.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *sqlite.Error, got %T: %v", err, err)
	}
	if se.Code() != sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY {
		t.Fatalf("Code()=%d, want SQLITE_CONSTRAINT_PRIMARYKEY (%d)", se.Code(), sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY)
	}
}

func TestIsUniqueViolation_NonUnique(t *testing.T) {
	repo := &repository{}

	if repo.IsUniqueViolation(nil) {
		t.Fatal("nil should not be unique violation")
	}
	if repo.IsUniqueViolation(errors.New("UNIQUE constraint failed: users.email")) {
		t.Fatal("plain error with UNIQUE message must not match without *sqlite.Error")
	}

	path := filepath.Join(t.TempDir(), "notnull.db")
	r, err := OpenRepository(t.Context(), path)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Errorf("repo.Close(): %v", err)
		}
	})

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("conn.Close(): %v", err)
		}
	})
	_, execErr := conn.ExecContext(t.Context(),
		`INSERT INTO users (id, email, password_hash) VALUES ('x', NULL, 'h')`)
	if execErr == nil {
		t.Fatal("want NOT NULL error")
	}
	if r.IsUniqueViolation(execErr) {
		t.Fatalf("NOT NULL should not be unique violation, err=%v", execErr)
	}
}
