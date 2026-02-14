package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath" // Added import for filepath

	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/project"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/lewtec/superfolha/internal/server"
	"github.com/spf13/cobra"
)

var (
	stateDir string
	dbURL    string
	port     string
)

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Superfolha Server",
	Long:  `Superfolha - A web-based LaTeX editor with Git version control and collaborative features.`,
	Run:   runServer,
}

func init() {
	rootCmd.Flags().StringVar(&stateDir, "state-dir", getEnv("STATE_DIR", "./data"), "Directory for Git repositories")
	rootCmd.Flags().StringVar(&dbURL, "db", getEnv("DATABASE_URL", ""), "PostgreSQL connection string")
	rootCmd.Flags().StringVar(&port, "port", getEnv("PORT", "8080"), "Server port")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runServer(cmd *cobra.Command, args []string) {
	if dbURL == "" {
		log.Fatal("Database URL is required (--db or DATABASE_URL)")
	}

	// Connect to database
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	// Run migrations with golang-migrate
	log.Println("Running database migrations...")
	sqlDB := stdlib.OpenDBFromPool(dbpool)
	if err := db.RunMigrations(sqlDB); err != nil {
		panic(fmt.Sprintf("Error: Failed to run migrations: %v", err))
	}

	// Create state directory
	absStateDir, err := filepath.Abs(stateDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for state directory %s: %v", stateDir, err)
	}
	stateDir = absStateDir

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		log.Fatalf("Failed to create state directory: %v", err)
	}

	// Create server
	projectService := project.NewService(dbpool, stateDir)
	authService := auth.NewService(dbpool)
	srv := server.NewServer(dbpool, stateDir, projectService, authService)

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on http://localhost%s", addr)
	log.Printf("GraphQL Playground: http://localhost%s/playground", addr)

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
