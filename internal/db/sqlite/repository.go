package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
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
	dsn, err := buildDSN(path)
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
		_ = conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := runMigrations(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &Repository{db: conn, queries: New(conn)}, nil
}

func buildDSN(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("sqlite path is empty")
	}
	if strings.HasPrefix(path, "file:") {
		return ensurePragmas(path), nil
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create sqlite directory %s: %w", dir, err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("absolute sqlite path: %w", err)
	}
	return ensurePragmas("file:" + abs), nil
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
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
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

func (r *Repository) IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() {
		case sqlite3.SQLITE_CONSTRAINT, sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
			return true
		}
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (r *Repository) Close() error {
	return r.db.Close()
}
