package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgmigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lewtec/superfolha/internal/db"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Repository struct {
	pool    *pgxpool.Pool
	queries *Queries
}

func NewRepository(dsn string) (*Repository, error) {
	migrationDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open migration database: %w", err)
	}
	if err := runMigrations(migrationDB); err != nil {
		if closeErr := migrationDB.Close(); closeErr != nil {
			slog.Error("failed to close migration database on error", "err", closeErr)
		}
		return nil, err
	}
	if closeErr := migrationDB.Close(); closeErr != nil {
		return nil, fmt.Errorf("close migration database: %w", closeErr)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Repository{pool: pool, queries: New(pool)}, nil
}

func runMigrations(conn *sql.DB) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := pgmigrate.WithInstance(conn, &pgmigrate.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func toUUID(s string) (pgtype.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// tsTime maps original TIMESTAMP columns (pgtype.Timestamp).
func tsTime(t pgtype.Timestamp) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func mapUser(u User) db.User {
	return db.User{
		ID:           uuidString(u.ID),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    tsTime(u.CreatedAt),
	}
}

func mapProject(p Project) db.Project {
	return db.Project{
		ID:        uuidString(p.ID),
		UserID:    uuidString(p.UserID),
		Name:      p.Name,
		GitPath:   p.GitPath,
		CreatedAt: tsTime(p.CreatedAt),
		UpdatedAt: tsTime(p.UpdatedAt),
	}
}

func (r *Repository) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	// Prefer explicit id when provided (matches other backends / git paths).
	// Empty id uses original INSERT (email, password_hash) → DEFAULT uuidv7().
	var (
		u   User
		err error
	)
	if arg.ID == "" {
		u, err = r.queries.CreateUser(ctx, CreateUserParams{
			Email:        arg.Email,
			PasswordHash: arg.PasswordHash,
		})
	} else {
		id, idErr := toUUID(arg.ID)
		if idErr != nil {
			return db.User{}, fmt.Errorf("invalid user id: %w", idErr)
		}
		u, err = r.queries.CreateUserWithID(ctx, CreateUserWithIDParams{
			ID:           id,
			Email:        arg.Email,
			PasswordHash: arg.PasswordHash,
		})
	}
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
	uid, err := toUUID(id)
	if err != nil {
		return db.User{}, fmt.Errorf("invalid user id: %w", err)
	}
	u, err := r.queries.GetUserByID(ctx, uid)
	if err != nil {
		return db.User{}, err
	}
	return mapUser(u), nil
}

func (r *Repository) CreateProject(ctx context.Context, arg db.CreateProjectParams) (db.Project, error) {
	id, err := toUUID(arg.ID)
	if err != nil {
		return db.Project{}, fmt.Errorf("invalid project id: %w", err)
	}
	userID, err := toUUID(arg.UserID)
	if err != nil {
		return db.Project{}, fmt.Errorf("invalid user id: %w", err)
	}
	p, err := r.queries.CreateProject(ctx, CreateProjectParams{
		ID: id, UserID: userID, Name: arg.Name, GitPath: arg.GitPath,
	})
	if err != nil {
		return db.Project{}, err
	}
	return mapProject(p), nil
}

func (r *Repository) GetProject(ctx context.Context, id string) (db.Project, error) {
	pid, err := toUUID(id)
	if err != nil {
		return db.Project{}, fmt.Errorf("invalid project id: %w", err)
	}
	p, err := r.queries.GetProject(ctx, pid)
	if err != nil {
		return db.Project{}, err
	}
	return mapProject(p), nil
}

func (r *Repository) GetUserProjects(ctx context.Context, userID string) ([]db.Project, error) {
	uid, err := toUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	rows, err := r.queries.GetUserProjects(ctx, uid)
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
	pid, err := toUUID(id)
	if err != nil {
		return fmt.Errorf("invalid project id: %w", err)
	}
	return r.queries.UpdateProjectTimestamp(ctx, pid)
}

func (r *Repository) DeleteProject(ctx context.Context, id string) error {
	pid, err := toUUID(id)
	if err != nil {
		return fmt.Errorf("invalid project id: %w", err)
	}
	return r.queries.DeleteProject(ctx, pid)
}

func (r *Repository) IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *Repository) Close() error {
	r.pool.Close()
	return nil
}
