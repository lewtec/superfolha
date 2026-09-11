package main

import (
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
		"--addr",
		"env: STATE_DIR",
		"env: PORT",
		"version",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("usage missing %q\n%s", want, text)
		}
	}
	for _, drop := range []string{"--db-driver", "--db "} {
		if strings.Contains(text, drop) {
			t.Errorf("usage still has %q\n%s", drop, text)
		}
	}
}

func TestParseRootFlags(t *testing.T) {
	app, err := cmd.Parse[cmd.App[root]](
		"--state-dir", "/data",
		"--addr", "0.0.0.0:9090",
	)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := app.Args.stateDir.Value(), "/data"; got != want {
		t.Errorf("stateDir = %q, want %q", got, want)
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

func unsetenv(t *testing.T, key string) {
	t.Helper()
	prev, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !ok {
			if err := os.Unsetenv(key); err != nil {
				t.Errorf("unset %s: %v", key, err)
			}
			return
		}
		if err := os.Setenv(key, prev); err != nil {
			t.Errorf("restore %s: %v", key, err)
		}
	})
}

func TestParseStateDirEnv(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		unsetenv(t, "STATE_DIR")
		app, err := cmd.Parse[cmd.App[root]]()
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if got, want := app.Args.stateDir.Value(), "./data"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
	})

	t.Run("STATE_DIR", func(t *testing.T) {
		t.Setenv("STATE_DIR", "/var/sf")
		app, err := cmd.Parse[cmd.App[root]]()
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if got, want := app.Args.stateDir.Value(), "/var/sf"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
	})

	t.Run("flag wins", func(t *testing.T) {
		t.Setenv("STATE_DIR", "/from-env")
		app, err := cmd.Parse[cmd.App[root]]("--state-dir", "/from-flag")
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if got, want := app.Args.stateDir.Value(), "/from-flag"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
	})
}

func TestParseAddr(t *testing.T) {
	tests := []struct {
		name string
		args []string
		port string
		want string
	}{
		{name: "flag wins over PORT", args: []string{"--addr", "0.0.0.0:9090"}, port: "1234", want: "0.0.0.0:9090"},
		{name: "PORT as port number", port: "8081", want: ":8081"},
		{name: "PORT as host:port", port: "0.0.0.0:9090", want: "0.0.0.0:9090"},
		{name: "default loopback", want: "127.0.0.1:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.port == "" {
				unsetenv(t, "PORT")
			} else {
				t.Setenv("PORT", tt.port)
			}
			app, err := cmd.Parse[cmd.App[root]](tt.args...)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := app.Args.addr.Value(); got != tt.want {
				t.Fatalf("addr = %q, want %q (PORT=%q)", got, tt.want, tt.port)
			}
		})
	}
}

func TestOpenRepository(t *testing.T) {
	stateDir := t.TempDir()
	wantDB := filepath.Join(stateDir, "superfolha.db")

	repo, err := openRepository(t.Context(), stateDir)
	if err != nil {
		t.Fatalf("openRepository(stateDir): %v", err)
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
}
