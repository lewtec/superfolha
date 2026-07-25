package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migrate_sqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/lewtec/superfolha/internal/db"
	"modernc.org/sqlite"
	_ "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Repository struct {
	db      *sql.DB
	queries *Queries
}

func NewRepository(path string) (*Repository, error) {
	dsn, filePath, err := buildDSN(path)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if err := conn.Ping(); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Error("failed to close sqlite connection on ping error", "err", closeErr)
		}
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := runMigrations(conn); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Error("failed to close sqlite connection on migration error", "err", closeErr)
		}
		return nil, err
	}

	// Owner-only: DB stores password hashes and session material.
	if err := ensureOwnerOnlyFile(filePath); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Error("failed to close sqlite connection on chmod error", "err", closeErr)
		}
		return nil, err
	}

	return &Repository{db: conn, queries: New(conn)}, nil
}

// buildDSN returns a modernc DSN and the filesystem path of the DB file (for chmod).
func buildDSN(path string) (dsn, filePath string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", fmt.Errorf("sqlite path is empty")
	}
	if strings.HasPrefix(path, "file:") {
		filePath = sqliteFilePath(path)
		return ensurePragmas(path), filePath, nil
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

// sqliteFilePath extracts the filesystem path from a file: URI (best-effort).
func sqliteFilePath(dsn string) string {
	s := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	return s
}

// ensureOwnerOnlyFile sets mode 0600 on path when it is a regular file.
// Missing path is ignored (in-memory / non-file DSNs).
func ensureOwnerOnlyFile(path string) error {
	if path == "" || path == ":memory:" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
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

func ensurePragmas(dsn string) string {
	const extras = "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	if strings.Contains(dsn, "_pragma=") {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&" + extras
	}
	return dsn + "?" + extras
}

func runMigrations(conn *sql.DB) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := migrate_sqlite.WithInstance(conn, &migrate_sqlite.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func parseTime(ns sql.NullString) time.Time {
	if !ns.Valid || ns.String == "" {
		return time.Time{}
	}
	s := ns.String
	for _, layout := range []string{
		time.RFC3339Nano, time.RFC3339,
		"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02T15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

func nullStr(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

func mapUser(u User) db.User {
	return db.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    parseTime(u.CreatedAt),
	}
}

func mapProject(p Project) db.Project {
	return db.Project{
		ID:        p.ID,
		UserID:    nullStr(p.UserID),
		Name:      p.Name,
		GitPath:   p.GitPath,
		CreatedAt: parseTime(p.CreatedAt),
		UpdatedAt: parseTime(p.UpdatedAt),
	}
}

func (r *Repository) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	id := arg.ID
	if id == "" {
		// SQLite has no uuidv7() default — mirror Postgres DEFAULT uuidv7() in app code.
		u7, err := uuid.NewV7()
		if err != nil {
			return db.User{}, fmt.Errorf("generate user id: %w", err)
		}
		id = u7.String()
	}
	u, err := r.queries.CreateUser(ctx, CreateUserParams{
		ID: id, Email: arg.Email, PasswordHash: arg.PasswordHash,
	})
	if err != nil {
		return db.User{}, err
	}
	return mapUser(u), nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	u, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return db.User{}, err
	}
	return mapUser(u), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (db.User, error) {
	u, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return db.User{}, err
	}
	return mapUser(u), nil
}

func (r *Repository) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
	p, err := r.queries.CreateProject(ctx, CreateProjectParams{
		ID: arg.ID, UserID: sql.NullString{String: arg.UserID, Valid: arg.UserID != ""},
		Name: arg.Name, GitPath: arg.GitPath,
	})
	if err != nil {
		return db.Project{}, err
	}
	return mapProject(p), nil
}

func (r *Repository) GetProject(ctx context.Context, id string) (db.Project, error) {
	p, err := r.queries.GetProject(ctx, id)
	if err != nil {
		return db.Project{}, err
	}
	return mapProject(p), nil
}

func (r *Repository) GetUserProjects(ctx context.Context, userID string) ([]db.Project, error) {
	rows, err := r.queries.GetUserProjects(ctx, sql.NullString{String: userID, Valid: userID != ""})
	if err != nil {
		return nil, err
	}
	out := make([]db.Project, len(rows))
	for i, p := range rows {
		out[i] = mapProject(p)
	}
	return out, nil
}

func (r *Repository) UpdateProjectTimestamp(ctx context.Context, id string) error {
	return r.queries.UpdateProjectTimestamp(ctx, id)
}

func (r *Repository) DeleteProject(ctx context.Context, id string) error {
	return r.queries.DeleteProject(ctx, id)
}

// IsUniqueViolation reports unique/primary-key constraint failures via
// modernc *sqlite.Error codes (2067 UNIQUE, 1555 PRIMARYKEY). Bare
// SQLITE_CONSTRAINT (19) is intentionally excluded — it also covers
// CHECK/FK/NOT NULL. String matching on err.Error() is avoided.
func (r *Repository) IsUniqueViolation(err error) bool {
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

func (r *Repository) Close() error {
	return r.db.Close()
}
