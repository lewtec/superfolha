package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	xtest "github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	moderncsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestOpenRepositorySetsOwnerOnlyMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "superfolha.db")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))
	repo, err := OpenRepository(t.Context(), path)
	require.NoError(t, err)
	xtest.CloseOnCleanup(t, repo)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o600), info.Mode().Perm())
}

func TestEnsureOwnerOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
	require.NoError(t, ensureOwnerOnlyFile(path))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o600), info.Mode().Perm())
	require.NoError(t, ensureOwnerOnlyFile(""))
	require.NoError(t, ensureOwnerOnlyFile(":memory:"))
}

func TestOpenRepositoryEmptyPath(t *testing.T) {
	_, err := OpenRepository(t.Context(), "")
	assert.ErrorIs(t, err, ErrEmptyPath)
}

func TestIsUniqueViolation_EmailConflict(t *testing.T) {
	repo, err := OpenRepository(t.Context(), filepath.Join(t.TempDir(), "unique.db"))
	require.NoError(t, err)
	xtest.CloseOnCleanup(t, repo)

	ctx := t.Context()
	_, err = repo.CreateUser(ctx, CreateUserParams{
		ID: "user-1", Email: "dup@example.com", PasswordHash: "hash-a",
	})
	require.NoError(t, err)

	_, err = repo.CreateUser(ctx, CreateUserParams{
		ID: "user-2", Email: "dup@example.com", PasswordHash: "hash-b",
	})
	require.Error(t, err)
	assert.True(t, repo.IsUniqueViolation(err), "email dup: %v", err)
	assert.True(t, repo.IsUniqueViolation(fmt.Errorf("create user: %w", err)), "should unwrap")

	var se *moderncsqlite.Error
	require.ErrorAs(t, err, &se)
	assert.Equal(t, sqlite3.SQLITE_CONSTRAINT_UNIQUE, se.Code())
}

func TestIsUniqueViolation_PrimaryKey(t *testing.T) {
	repo, err := OpenRepository(t.Context(), filepath.Join(t.TempDir(), "pk.db"))
	require.NoError(t, err)
	xtest.CloseOnCleanup(t, repo)

	ctx := t.Context()
	_, err = repo.CreateUser(ctx, CreateUserParams{
		ID: "same-id", Email: "a@example.com", PasswordHash: "h",
	})
	require.NoError(t, err)
	_, err = repo.CreateUser(ctx, CreateUserParams{
		ID: "same-id", Email: "b@example.com", PasswordHash: "h",
	})
	require.Error(t, err)
	assert.True(t, repo.IsUniqueViolation(err), "pk dup: %v", err)

	var se *moderncsqlite.Error
	require.ErrorAs(t, err, &se)
	assert.Equal(t, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, se.Code())
}

func TestIsUniqueViolation_NonUnique(t *testing.T) {
	repo := &repository{}
	assert.False(t, repo.IsUniqueViolation(nil))
	assert.False(t, repo.IsUniqueViolation(errors.New("UNIQUE constraint failed: users.email")))

	path := filepath.Join(t.TempDir(), "notnull.db")
	r, err := OpenRepository(t.Context(), path)
	require.NoError(t, err)
	xtest.CloseOnCleanup(t, r)

	conn, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	xtest.CloseOnCleanup(t, conn)
	_, execErr := conn.ExecContext(t.Context(),
		`INSERT INTO users (id, email, password_hash) VALUES ('x', NULL, 'h')`)
	require.Error(t, execErr)
	assert.False(t, r.IsUniqueViolation(execErr), "NOT NULL: %v", execErr)
}
