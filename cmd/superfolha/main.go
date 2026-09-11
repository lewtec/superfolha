package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/db/postgres"
	"github.com/lewtec/superfolha/internal/db/sqlite"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/server"
)

var (
	// Driver configuration errors (errors.Is).
	ErrPostgresDSNRequired = errors.New("postgres driver requires a DSN (--db / DATABASE_URL)")
	ErrUnknownDBDriver     = errors.New("unknown database driver")
)

type root struct {
	stateDir cmd.StringArg `long:"state-dir" help:"Directory for Git repositories (and default SQLite path)"`
	dbDriver cmd.StringArg `long:"db-driver" help:"Database driver: sqlite (default) or postgres"`
	db       cmd.StringArg `long:"db" help:"Database DSN (sqlite path or postgres:// URL)"`
	addr     cmd.StringArg `long:"addr" help:"Listen address (default: :$PORT if set, else 127.0.0.1:8080)"`
	version  *versionCmd
}

func (root) Description() string {
	return "Superfolha - A web-based LaTeX editor with Git version control and collaborative features."
}

func (r *root) applyEnv() {
	if r.stateDir.Value() == "" {
		_ = r.stateDir.Parse(cmp.Or(os.Getenv("STATE_DIR"), "./data"))
	}
	if r.dbDriver.Value() == "" {
		if v := cmp.Or(os.Getenv("DB_DRIVER"), os.Getenv("DATABASE_DRIVER")); v != "" {
			_ = r.dbDriver.Parse(v)
		}
	}
	if r.db.Value() == "" {
		if v := cmp.Or(os.Getenv("DATABASE_URL"), os.Getenv("DATABASE_DSN")); v != "" {
			_ = r.db.Parse(v)
		}
	}
}

type versionCmd struct{}

func (versionCmd) Description() string {
	return "Print version"
}

func (*versionCmd) Run(context.Context) error {
	_, err := fmt.Println(release.Version())
	return err
}

// resolveAddr picks the listen address:
//  1. --addr if set
//  2. else PORT env (":$PORT" if PORT is only a port number)
//  3. else loopback 127.0.0.1:8080
func resolveAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr != "" {
		return addr
	}
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.Contains(port, ":") {
			return port
		}
		return ":" + port
	}
	return "127.0.0.1:8080"
}

func openRepository(ctx context.Context, driver, dsn, stateDir string) (db.Repository, error) {
	driver = strings.ToLower(strings.TrimSpace(driver))
	dsn = strings.TrimSpace(dsn)
	if driver == "" {
		driver = db.InferDriver(dsn)
	}

	switch driver {
	case "postgres", "postgresql":
		slog.Warn("postgres driver is deprecated; Superfolha targets single-instance SQLite — plan to migrate off postgres")
		if dsn == "" {
			return nil, ErrPostgresDSNRequired
		}
		return postgres.NewRepository(ctx, dsn)
	case "sqlite", "sqlite3":
		if dsn == "" {
			dsn = filepath.Join(stateDir, "superfolha.db")
		}
		return sqlite.NewRepository(dsn)
	default:
		return nil, fmt.Errorf("%w %q (want sqlite or postgres)", ErrUnknownDBDriver, driver)
	}
}

func (r *root) Run(ctx context.Context) error {
	r.applyEnv()

	absStateDir, err := filepath.Abs(r.stateDir.Value())
	if err != nil {
		return fmt.Errorf("state directory %q: %w", r.stateDir.Value(), err)
	}
	if err := os.MkdirAll(absStateDir, 0o755); err != nil {
		return fmt.Errorf("create state directory %q: %w", absStateDir, err)
	}

	repo, err := openRepository(ctx, r.dbDriver.Value(), r.db.Value(), absStateDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer repo.Close()

	driver := r.dbDriver.Value()
	if driver == "" {
		driver = db.InferDriver(r.db.Value())
	}
	slog.Info("connected to database", "driver", driver)

	projectService := project.NewService(absStateDir)
	authService := auth.NewService(repo)
	srv := server.NewServer(repo, absStateDir, projectService, authService)

	addr := resolveAddr(r.addr.Value())
	slog.Info("starting server", "addr", addr, "version", release.Version())

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutting down server", "cause", ctx.Err())
		// Parent is already cancelled (signal); WithoutCancel keeps values without inheriting cancel.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "err", err)
		}
		srv.CloseHubs()
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server after shutdown: %w", err)
		}
		return nil
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := cmd.Parse[cmd.App[root]](os.Args[1:]...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.Run(ctx); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
