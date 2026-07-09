package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	stateDir string
	dbDriver string
	dbDSN    string
	port     string
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Superfolha Server",
	Long:  `Superfolha - A web-based LaTeX editor with Git version control and collaborative features.`,
	Run:   runServer,
}

func init() {
	rootCmd.Flags().StringVar(&stateDir, "state-dir", getEnv("STATE_DIR", "./data"), "Directory for Git repositories (and default SQLite path)")
	rootCmd.Flags().StringVar(&dbDriver, "db-driver", firstEnv("DB_DRIVER", "DATABASE_DRIVER"), "Database driver: sqlite (default) or postgres")
	rootCmd.Flags().StringVar(&dbDSN, "db", firstEnv("DATABASE_URL", "DATABASE_DSN"), "Database DSN (sqlite path or postgres:// URL)")
	rootCmd.Flags().StringVar(&port, "port", getEnv("PORT", "8080"), "Server port")
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

func openRepository(driver, dsn, stateDir string) (db.Repository, error) {
	driver = strings.ToLower(strings.TrimSpace(driver))
	dsn = strings.TrimSpace(dsn)
	if driver == "" {
		driver = db.InferDriver(dsn)
	}

	switch driver {
	case "postgres", "postgresql":
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
		log.Fatalf("Failed to get absolute path for state directory %s: %v", stateDir, err)
	}
	stateDir = absStateDir

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		log.Fatalf("Failed to create state directory: %v", err)
	}

	repo, err := openRepository(dbDriver, dbDSN, stateDir)
	if err != nil {
		log.Fatalf("Unable to open database: %v", err)
	}
	defer repo.Close()

	driver := dbDriver
	if driver == "" {
		driver = db.InferDriver(dbDSN)
	}
	log.Printf("Connected to database (driver=%s)", driver)

	projectService := project.NewService(stateDir)
	authService := auth.NewService(repo)
	srv := server.NewServer(repo, stateDir, projectService, authService)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on http://localhost%s", addr)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
