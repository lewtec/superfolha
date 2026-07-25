package sqlite

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/lewtec/superfolha/internal/db"
	moderncsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestIsUniqueViolation_EmailConflict(t *testing.T) {
	repo, err := NewRepository(filepath.Join(t.TempDir(), "unique.db"))
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	ctx := t.Context()
	_, err = repo.CreateUser(ctx, db.CreateUserParams{
		ID: "user-1", Email: "dup@example.com", PasswordHash: "hash-a",
	})
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	_, err = repo.CreateUser(ctx, db.CreateUserParams{
		ID: "user-2", Email: "dup@example.com", PasswordHash: "hash-b",
	})
	if err == nil {
		t.Fatal("second CreateUser with same email: want unique violation, got nil")
	}
	if !repo.IsUniqueViolation(err) {
		t.Fatalf("IsUniqueViolation(email dup)=false, err=%v", err)
	}

	// Auth and callers often wrap driver errors.
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
	repo, err := NewRepository(filepath.Join(t.TempDir(), "pk.db"))
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	ctx := t.Context()
	if _, err := repo.CreateUser(ctx, db.CreateUserParams{
		ID: "same-id", Email: "a@example.com", PasswordHash: "h",
	}); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	_, err = repo.CreateUser(ctx, db.CreateUserParams{
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
	repo := &Repository{}

	if repo.IsUniqueViolation(nil) {
		t.Fatal("nil should not be unique violation")
	}
	// Fragile string matching must not be used — message alone is insufficient.
	if repo.IsUniqueViolation(errors.New("UNIQUE constraint failed: users.email")) {
		t.Fatal("plain error with UNIQUE message must not match without *sqlite.Error")
	}
	if repo.IsUniqueViolation(fmt.Errorf("wrap: %w", errors.New("UNIQUE constraint failed"))) {
		t.Fatal("wrapped plain error must not match")
	}

	// NOT NULL is a constraint failure but not a unique violation.
	r, err := NewRepository(filepath.Join(t.TempDir(), "notnull.db"))
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	_, execErr := r.db.ExecContext(t.Context(),
		`INSERT INTO users (id, email, password_hash) VALUES ('x', NULL, 'h')`)
	if execErr == nil {
		t.Fatal("want NOT NULL error")
	}
	if r.IsUniqueViolation(execErr) {
		t.Fatalf("NOT NULL should not be unique violation, err=%v", execErr)
	}
}
