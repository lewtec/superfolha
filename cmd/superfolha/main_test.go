package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
		port string // "" clears PORT; non-empty sets PORT
		want string
	}{
		{
			name: "explicit addr wins over PORT",
			addr: "0.0.0.0:9090",
			port: "1234",
			want: "0.0.0.0:9090",
		},
		{
			name: "explicit addr with surrounding spaces",
			addr: "  :3000  ",
			port: "",
			want: ":3000",
		},
		{
			name: "PORT as port number",
			addr: "",
			port: "8081",
			want: ":8081",
		},
		{
			name: "PORT as host:port",
			addr: "",
			port: "0.0.0.0:9090",
			want: "0.0.0.0:9090",
		},
		{
			name: "PORT with surrounding spaces",
			addr: "",
			port: "  4000  ",
			want: ":4000",
		},
		{
			name: "empty addr and empty PORT defaults to loopback",
			addr: "",
			port: "",
			want: "127.0.0.1:8080",
		},
		{
			name: "whitespace addr treated as empty",
			addr: "   ",
			port: "",
			want: "127.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always clear first so ambient PORT cannot leak into the default case.
			t.Setenv("PORT", "")
			if tt.port != "" {
				t.Setenv("PORT", tt.port)
			}

			got := resolveAddr(tt.addr)
			if got != tt.want {
				t.Fatalf("resolveAddr(%q) with PORT=%q: got %q, want %q", tt.addr, tt.port, got, tt.want)
			}
		})
	}
}

func TestOpenRepository(t *testing.T) {
	t.Run("unknown driver", func(t *testing.T) {
		repo, err := openRepository(t.Context(), "mysql", "unused", t.TempDir())
		if err == nil {
			if repo != nil {
				_ = repo.Close()
			}
			t.Fatal("expected error for unknown driver")
		}
		if !strings.Contains(err.Error(), "unknown database driver") {
			t.Fatalf("error = %q, want substring %q", err.Error(), "unknown database driver")
		}
		if repo != nil {
			t.Fatal("expected nil repository on error")
		}
	})

	t.Run("postgres without DSN", func(t *testing.T) {
		repo, err := openRepository(t.Context(), "postgres", "", t.TempDir())
		if err == nil {
			if repo != nil {
				_ = repo.Close()
			}
			t.Fatal("expected error for postgres without DSN")
		}
		if !strings.Contains(err.Error(), "postgres driver requires a DSN") {
			t.Fatalf("error = %q, want substring %q", err.Error(), "postgres driver requires a DSN")
		}
		if repo != nil {
			t.Fatal("expected nil repository on error")
		}
	})

	t.Run("postgresql alias without DSN", func(t *testing.T) {
		_, err := openRepository(t.Context(), "postgresql", "  ", t.TempDir())
		if err == nil {
			t.Fatal("expected error for postgresql without DSN")
		}
		if !strings.Contains(err.Error(), "postgres driver requires a DSN") {
			t.Fatalf("error = %q, want substring %q", err.Error(), "postgres driver requires a DSN")
		}
	})

	t.Run("sqlite empty DSN uses stateDir path", func(t *testing.T) {
		stateDir := t.TempDir()
		wantDB := filepath.Join(stateDir, "superfolha.db")

		repo, err := openRepository(t.Context(), "sqlite", "", stateDir)
		if err != nil {
			t.Fatalf("openRepository(sqlite, \"\", stateDir): %v", err)
		}
		if repo == nil {
			t.Fatal("expected non-nil repository")
		}
		t.Cleanup(func() {
			if err := repo.Close(); err != nil {
				t.Errorf("repo.Close(): %v", err)
			}
		})

		if _, err := os.Stat(wantDB); err != nil {
			t.Fatalf("expected sqlite file at %s: %v", wantDB, err)
		}
	})

	t.Run("sqlite3 alias empty DSN uses stateDir path", func(t *testing.T) {
		stateDir := t.TempDir()
		wantDB := filepath.Join(stateDir, "superfolha.db")

		repo, err := openRepository(t.Context(), "sqlite3", "  ", stateDir)
		if err != nil {
			t.Fatalf("openRepository(sqlite3, whitespace DSN, stateDir): %v", err)
		}
		t.Cleanup(func() {
			if err := repo.Close(); err != nil {
				t.Errorf("repo.Close(): %v", err)
			}
		})

		if _, err := os.Stat(wantDB); err != nil {
			t.Fatalf("expected sqlite file at %s: %v", wantDB, err)
		}
	})
}
