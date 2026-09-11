package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
)

func TestRootUsage(t *testing.T) {
	text, err := cmd.Usage[cmd.App[root]]("superfolha")
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	for _, want := range []string{
		"Superfolha",
		"--state-dir",
		"--db-driver",
		"--db",
		"--addr",
		"version",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("usage missing %q\n%s", want, text)
		}
	}
}

func TestParseRootFlags(t *testing.T) {
	app, err := cmd.Parse[cmd.App[root]](
		"--state-dir", "/data",
		"--db-driver", "sqlite",
		"--db", "/data/superfolha.db",
		"--addr", "0.0.0.0:9090",
	)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := app.Args.stateDir.Value(), "/data"; got != want {
		t.Errorf("stateDir = %q, want %q", got, want)
	}
	if got, want := app.Args.dbDriver.Value(), "sqlite"; got != want {
		t.Errorf("dbDriver = %q, want %q", got, want)
	}
	if got, want := app.Args.db.Value(), "/data/superfolha.db"; got != want {
		t.Errorf("db = %q, want %q", got, want)
	}
	if got, want := app.Args.addr.Value(), "0.0.0.0:9090"; got != want {
		t.Errorf("addr = %q, want %q", got, want)
	}
	if app.Args.version != nil {
		t.Fatal("version command should be unset")
	}
}

func TestParseVersionCommand(t *testing.T) {
	app, err := cmd.Parse[cmd.App[root]]("version")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if app.Args.version == nil {
		t.Fatal("expected version command")
	}
}

func TestRootApplyEnv(t *testing.T) {
	t.Run("defaults when env empty", func(t *testing.T) {
		t.Setenv("STATE_DIR", "")
		t.Setenv("DB_DRIVER", "")
		t.Setenv("DATABASE_DRIVER", "")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("DATABASE_DSN", "")

		var r root
		r.applyEnv()
		if got, want := r.stateDir.Value(), "./data"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
		if got := r.dbDriver.Value(); got != "" {
			t.Errorf("dbDriver = %q, want empty", got)
		}
		if got := r.db.Value(); got != "" {
			t.Errorf("db = %q, want empty", got)
		}
	})

	t.Run("env fills empty flags", func(t *testing.T) {
		t.Setenv("STATE_DIR", "/var/sf")
		t.Setenv("DB_DRIVER", "postgres")
		t.Setenv("DATABASE_URL", "postgres://x")

		var r root
		r.applyEnv()
		if got, want := r.stateDir.Value(), "/var/sf"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
		if got, want := r.dbDriver.Value(), "postgres"; got != want {
			t.Errorf("dbDriver = %q, want %q", got, want)
		}
		if got, want := r.db.Value(), "postgres://x"; got != want {
			t.Errorf("db = %q, want %q", got, want)
		}
	})

	t.Run("alternate env keys", func(t *testing.T) {
		t.Setenv("STATE_DIR", "")
		t.Setenv("DB_DRIVER", "")
		t.Setenv("DATABASE_DRIVER", "sqlite")
		t.Setenv("DATABASE_URL", "")
		t.Setenv("DATABASE_DSN", "/tmp/sf.db")

		var r root
		r.applyEnv()
		if got, want := r.dbDriver.Value(), "sqlite"; got != want {
			t.Errorf("dbDriver = %q, want %q", got, want)
		}
		if got, want := r.db.Value(), "/tmp/sf.db"; got != want {
			t.Errorf("db = %q, want %q", got, want)
		}
	})

	t.Run("flags win over env", func(t *testing.T) {
		t.Setenv("STATE_DIR", "/from-env")
		t.Setenv("DB_DRIVER", "postgres")
		t.Setenv("DATABASE_URL", "postgres://env")

		var r root
		if err := r.stateDir.Parse("/from-flag"); err != nil {
			t.Fatalf("stateDir.Parse: %v", err)
		}
		if err := r.dbDriver.Parse("sqlite"); err != nil {
			t.Fatalf("dbDriver.Parse: %v", err)
		}
		if err := r.db.Parse("/from-flag.db"); err != nil {
			t.Fatalf("db.Parse: %v", err)
		}
		r.applyEnv()
		if got, want := r.stateDir.Value(), "/from-flag"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
		if got, want := r.dbDriver.Value(), "sqlite"; got != want {
			t.Errorf("dbDriver = %q, want %q", got, want)
		}
		if got, want := r.db.Value(), "/from-flag.db"; got != want {
			t.Errorf("db = %q, want %q", got, want)
		}
	})
}

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
	ctx := t.Context()

	t.Run("unknown driver", func(t *testing.T) {
		repo, err := openRepository(ctx, "mysql", "unused", t.TempDir())
		if err == nil {
			if repo != nil {
				if closeErr := repo.Close(); closeErr != nil {
					t.Errorf("repo.Close(): %v", closeErr)
				}
			}
			t.Fatal("expected error for unknown driver")
		}
		if !errors.Is(err, ErrUnknownDBDriver) {
			t.Fatalf("error = %v, want %v", err, ErrUnknownDBDriver)
		}
		if repo != nil {
			t.Fatal("expected nil repository on error")
		}
	})

	t.Run("postgres without DSN", func(t *testing.T) {
		repo, err := openRepository(ctx, "postgres", "", t.TempDir())
		if err == nil {
			if repo != nil {
				if closeErr := repo.Close(); closeErr != nil {
					t.Errorf("repo.Close(): %v", closeErr)
				}
			}
			t.Fatal("expected error for postgres without DSN")
		}
		if !errors.Is(err, ErrPostgresDSNRequired) {
			t.Fatalf("error = %v, want %v", err, ErrPostgresDSNRequired)
		}
		if repo != nil {
			t.Fatal("expected nil repository on error")
		}
	})

	t.Run("postgresql alias without DSN", func(t *testing.T) {
		_, err := openRepository(ctx, "postgresql", "  ", t.TempDir())
		if err == nil {
			t.Fatal("expected error for postgresql without DSN")
		}
		if !errors.Is(err, ErrPostgresDSNRequired) {
			t.Fatalf("error = %v, want %v", err, ErrPostgresDSNRequired)
		}
	})

	t.Run("sqlite empty DSN uses stateDir path", func(t *testing.T) {
		stateDir := t.TempDir()
		wantDB := filepath.Join(stateDir, "superfolha.db")

		repo, err := openRepository(ctx, "sqlite", "", stateDir)
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

		repo, err := openRepository(ctx, "sqlite3", "  ", stateDir)
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
