package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	_ "github.com/lib/pq"
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
	Short: "LaTeX Editor Server",
	Long:  `A web-based LaTeX editor with Git version control and collaborative features.`,
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
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	// Run migrations with dbmate
	log.Println("Running database migrations...")
	if err := runMigrations(dbURL); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
		log.Println("Continuing anyway...")
	}

	// Create state directory
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		log.Fatalf("Failed to create state directory: %v", err)
	}

	// Create server
	srv := server.NewServer(db, stateDir)

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on http://localhost%s", addr)
	log.Printf("GraphQL Playground: http://localhost%s/playground", addr)

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runMigrations(dbURL string) error {
	// Check if dbmate is installed
	if _, err := exec.LookPath("dbmate"); err != nil {
		log.Println("dbmate not found, skipping migrations")
		return nil
	}

	cmd := exec.Command("dbmate", "up")
	cmd.Env = append(os.Environ(), fmt.Sprintf("DATABASE_URL=%s", dbURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
