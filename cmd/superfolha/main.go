package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/server"
)

type root struct {
	stateDir cmd.DataDirArg `long:"state-dir" env:"STATE_DIR" default:"./data" help:"Directory for Git repositories and SQLite"`
	addr     cmd.AddrArg    `long:"addr" env:"PORT" default:"127.0.0.1:8080" help:"Listen address"`
	database db.DBArg       `long:"database" default:"" help:"SQLite path or URL (default: {state-dir}/superfolha.db)"`
	version  *cmd.VersionCmd
}

func (*root) Description() string {
	return "Superfolha - A web-based LaTeX editor with Git version control and collaborative features."
}

type instance struct {
	repo  db.Repository
	srv   *server.Server
	state *path.Root
}

func openInstance(ctx context.Context, stateDir string, database *db.DBArg) (in *instance, err error) {
	state, err := path.Open(stateDir)
	if err != nil {
		return nil, fmt.Errorf("state directory %q: %w", stateDir, err)
	}
	defer func() {
		if in != nil {
			return
		}
		err = errors.Join(err, state.Close())
	}()

	if database.Value() == nil || database.Value().URL() == "" {
		u, err := db.FileURL(filepath.Join(state.Name(), path.New("superfolha.db").String()))
		if err != nil {
			return nil, err
		}
		if err := database.Parse(u); err != nil {
			return nil, err
		}
	}
	repo, err := db.OpenArg(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	dir := state.Name()
	projectService := project.NewService(dir)
	authService := auth.NewService(repo)
	in = &instance{
		repo:  repo,
		srv:   server.NewServer(repo, dir, projectService, authService),
		state: state,
	}
	return in, nil
}

func (in *instance) Close() error {
	if in == nil {
		return nil
	}
	var err error
	if in.repo != nil {
		err = in.repo.Close()
	}
	if in.state != nil {
		err = errors.Join(err, in.state.Close())
	}
	return err
}

func (r *root) Run(ctx context.Context) (err error) {
	in, err := openInstance(ctx, r.stateDir.Value(), &r.database)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, in.Close()) }()

	slog.Info("connected to database", "driver", "sqlite", "path", r.database.Value().URL())
	addr := r.addr.Value()
	slog.Info("starting server", "addr", addr, "version", release.Version())
	return (httpRun{
		server:    newHTTPServer(addr, in.srv.Handler()),
		closeHubs: in.srv.CloseHubs,
	}).run(ctx)
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
}

// httpRun serves until ctx is cancelled.
// A nil listener uses ListenAndServe on server.Addr. closeHubs runs after Shutdown.
type httpRun struct {
	server    *http.Server
	ln        net.Listener
	closeHubs func()
}

func (h httpRun) run(ctx context.Context) error {
	serve := h.server.ListenAndServe
	if h.ln != nil {
		ln := h.ln
		serve = func() error { return h.server.Serve(ln) }
	}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- serve()
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
		if err := h.server.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "err", err)
		}
		if h.closeHubs != nil {
			h.closeHubs()
		}
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server after shutdown: %w", err)
		}
		return nil
	}
}

// wantWindow reports whether this process should open the lewkit app window.
// lewkit release run stamps the app id and version, then starts the binary
// with no arguments. Any argument, including version and --addr, stays on
// the HTTP server CLI. There is no desktop subcommand.
func wantWindow(args []string) bool {
	if len(args) > 0 {
		return false
	}
	return releaseStamped()
}

func releaseStamped() bool {
	if !stampedVersion(release.Version()) {
		return false
	}
	_, err := release.AppID()
	return err == nil
}

func stampedVersion(version string) bool {
	version = strings.TrimSpace(version)
	return version != "" && version != "dev" && !strings.HasPrefix(version, "dev-")
}

func init() { entry.Bind(run) }

func main() {
	if wantWindow(os.Args[1:]) {
		entry.Main(run)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cli, err := cmd.Parse[cmd.App[root]](os.Args[1:]...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := cli.Run(ctx); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

// run is the stamped app entry. entry.Main and the Android host both call it.
func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM)
	defer stop()
	return runWindow(ctx)
}
