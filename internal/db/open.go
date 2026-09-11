package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	xdb "github.com/lewtec/lewkit/x/db"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Sentinel for empty SQLite path (errors.Is).
var ErrEmptyPath = errors.New("sqlite path is empty")

const sqlitePragmas = "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"

type repository struct {
	conn *xdb.Conn[Queries]
}

// OpenRepository migrates and opens SQLite at path via lewkit x/db.
func OpenRepository(ctx context.Context, path string) (Repository, error) {
	url, err := FileURL(path)
	if err != nil {
		return nil, err
	}
	var a xdb.Arg[Queries]
	if err := a.Parse(url); err != nil {
		return nil, err
	}
	return OpenArg(ctx, &a)
}

// OpenArg migrates a.URL() and wraps it as Repository.
func OpenArg(ctx context.Context, a *xdb.Arg[Queries]) (Repository, error) {
	if a == nil || a.Value() == nil || a.Value().URL() == "" {
		return nil, ErrEmptyPath
	}
	if err := Open(ctx, a); err != nil {
		return nil, err
	}
	// Owner-only: DB stores password hashes and session material.
	if err := ensureOwnerOnlyFile(sqliteFilePath(a.Value().URL())); err != nil {
		return nil, errors.Join(err, a.Value().Close())
	}
	return &repository{conn: a.Value()}, nil
}

// FileURL is a file: SQLite URL with the usual pragmas.
func FileURL(path string) (string, error) {
	url, _, err := sqliteURL(path)
	return url, err
}

func sqliteURL(path string) (url, filePath string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", ErrEmptyPath
	}
	if strings.HasPrefix(path, "file:") {
		return ensurePragmas(path), sqliteFilePath(path), nil
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", fmt.Errorf("create sqlite directory %s: %w", dir, err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("absolute sqlite path: %w", err)
	}
	return ensurePragmas("file:" + abs), abs, nil
}

func sqliteFilePath(dsn string) string {
	s := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	return s
}

func ensurePragmas(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&" + sqlitePragmas
	}
	return dsn + "?" + sqlitePragmas
}

func ensureOwnerOnlyFile(path string) error {
	if path == "" || path == ":memory:" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat sqlite file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod sqlite file 0600: %w", err)
	}
	return nil
}

func (r *repository) q() Queries {
	return r.conn.Queries()
}

func (r *repository) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	if arg.ID == "" {
		u7, err := uuid.NewV7()
		if err != nil {
			return User{}, fmt.Errorf("generate user id: %w", err)
		}
		arg.ID = u7.String()
	}
	return r.q().CreateUser(ctx, arg)
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	return r.q().GetUserByEmail(ctx, email)
}

func (r *repository) GetUserByID(ctx context.Context, id string) (User, error) {
	return r.q().GetUserByID(ctx, id)
}

func (r *repository) CreateProject(ctx context.Context, arg CreateProjectParams) (Project, error) {
	return r.q().CreateProject(ctx, arg)
}

func (r *repository) GetProject(ctx context.Context, id string) (Project, error) {
	return r.q().GetProject(ctx, id)
}

func (r *repository) GetUserProjects(ctx context.Context, userID string) ([]Project, error) {
	return r.q().GetUserProjects(ctx, sql.NullString{String: userID, Valid: userID != ""})
}

func (r *repository) UpdateProjectTimestamp(ctx context.Context, id string) error {
	return r.q().UpdateProjectTimestamp(ctx, id)
}

func (r *repository) DeleteProject(ctx context.Context, id string) error {
	return r.q().DeleteProject(ctx, id)
}

// IsUniqueViolation reports unique/primary-key constraint failures via
// modernc *sqlite.Error codes (2067 UNIQUE, 1555 PRIMARYKEY). Bare
// SQLITE_CONSTRAINT (19) is intentionally excluded — it also covers
// CHECK/FK/NOT NULL. String matching on err.Error() is avoided.
func (r *repository) IsUniqueViolation(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	switch se.Code() {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
		return true
	default:
		return false
	}
}

func (r *repository) Close() error {
	if r.conn == nil {
		return nil
	}
	return r.conn.Close()
}
