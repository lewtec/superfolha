package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/server"
)

type root struct {
	stateDir cmd.StringArg `long:"state-dir" env:"STATE_DIR" default:"./data" help:"Directory for Git repositories and SQLite"`
	addr     cmd.AddrArg   `long:"addr" env:"PORT" default:"127.0.0.1:8080" help:"Listen address"`
	version  *versionCmd
}

func (root) Description() string {
	return "Superfolha - A web-based LaTeX editor with Git version control and collaborative features."
}

type versionCmd struct{}

func (versionCmd) Description() string {
	return "Print version"
}

func (*versionCmd) Run(context.Context) error {
	_, err := fmt.Println(release.Version())
	return err
}

func openRepository(ctx context.Context, stateDir string) (db.Repository, error) {
	return db.OpenRepository(ctx, filepath.Join(stateDir, "superfolha.db"))
}

func (r *root) Run(ctx context.Context) error {
	absStateDir, err := filepath.Abs(r.stateDir.Value())
	if err != nil {
		return fmt.Errorf("state directory %q: %w", r.stateDir.Value(), err)
	}
	if err := os.MkdirAll(absStateDir, 0o755); err != nil {
		return fmt.Errorf("create state directory %q: %w", absStateDir, err)
	}

	repo, err := openRepository(ctx, absStateDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer repo.Close()

	slog.Info("connected to database", "driver", "sqlite", "path", filepath.Join(absStateDir, "superfolha.db"))

	projectService := project.NewService(absStateDir)
	authService := auth.NewService(repo)
	srv := server.NewServer(repo, absStateDir, projectService, authService)

	addr := r.addr.Value()
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
