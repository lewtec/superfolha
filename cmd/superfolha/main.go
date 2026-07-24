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
	"strings"
	"syscall"
	"time"

	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/db/postgres"
	"github.com/lewtec/superfolha/internal/db/sqlite"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/server"
	"github.com/spf13/cobra"
)

var (
	// version is set by GoReleaser (-X main.version=...).
	version    = "dev"
	stateDir   string
	dbDriver   string
	dbDSN      string
	listenAddr string
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Superfolha Server",
	Long:  `Superfolha - A web-based LaTeX editor with Git version control and collaborative features.`,
	Run:   runServer,
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	})
	rootCmd.Flags().StringVar(&stateDir, "state-dir", getEnv("STATE_DIR", "./data"), "Directory for Git repositories (and default SQLite path)")
	rootCmd.Flags().StringVar(&dbDriver, "db-driver", firstEnv("DB_DRIVER", "DATABASE_DRIVER"), "Database driver: sqlite (default) or postgres")
	rootCmd.Flags().StringVar(&dbDSN, "db", firstEnv("DATABASE_URL", "DATABASE_DSN"), "Database DSN (sqlite path or postgres:// URL)")
	// Empty default: resolveAddr picks $PORT or 127.0.0.1:8080.
	rootCmd.Flags().StringVar(&listenAddr, "addr", "", "Listen address (default: :$PORT if set, else 127.0.0.1:8080)")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
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

func openRepository(driver, dsn, stateDir string) (db.Repository, error) {
	driver = strings.ToLower(strings.TrimSpace(driver))
	dsn = strings.TrimSpace(dsn)
	if driver == "" {
		driver = db.InferDriver(dsn)
	}

	switch driver {
	case "postgres", "postgresql":
		slog.Warn("postgres driver is deprecated; Superfolha targets single-instance SQLite — plan to migrate off postgres")
		if dsn == "" {
			return nil, fmt.Errorf("postgres driver requires a DSN (--db / DATABASE_URL)")
		}
		return postgres.NewRepository(dsn)
	case "sqlite", "sqlite3":
		if dsn == "" {
			dsn = filepath.Join(stateDir, "superfolha.db")
		}
		return sqlite.NewRepository(dsn)
	default:
		return nil, fmt.Errorf("unknown database driver %q (want sqlite or postgres)", driver)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	absStateDir, err := filepath.Abs(stateDir)
	if err != nil {
		slog.Error("failed to get absolute path for state directory", "state_dir", stateDir, "err", err)
		os.Exit(1)
	}
	stateDir = absStateDir

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		slog.Error("failed to create state directory", "err", err)
		os.Exit(1)
	}

	repo, err := openRepository(dbDriver, dbDSN, stateDir)
	if err != nil {
		slog.Error("unable to open database", "err", err)
		os.Exit(1)
	}
	defer repo.Close()

	driver := dbDriver
	if driver == "" {
		driver = db.InferDriver(dbDSN)
	}
	slog.Info("connected to database", "driver", driver)

	projectService := project.NewService(stateDir)
	authService := auth.NewService(repo)
	srv := server.NewServer(repo, stateDir, projectService, authService)

	addr := resolveAddr(listenAddr)
	slog.Info("starting server", "addr", addr, "version", version)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	case sig := <-quit:
		slog.Info("shutting down server", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("HTTP server shutdown error", "err", err)
		}
		srv.CloseHubs()
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error after shutdown", "err", err)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
