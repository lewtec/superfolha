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

func TestRootApplyEnv(t *testing.T) {
	t.Run("defaults when env empty", func(t *testing.T) {
		t.Setenv("STATE_DIR", "")

		var r root
		r.applyEnv()
		if got, want := r.stateDir.Value(), "./data"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
	})

	t.Run("env fills empty flag", func(t *testing.T) {
		t.Setenv("STATE_DIR", "/var/sf")

		var r root
		r.applyEnv()
		if got, want := r.stateDir.Value(), "/var/sf"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
		}
	})

	t.Run("flag wins over env", func(t *testing.T) {
		t.Setenv("STATE_DIR", "/from-env")

		var r root
		if err := r.stateDir.Parse("/from-flag"); err != nil {
			t.Fatalf("stateDir.Parse: %v", err)
		}
		r.applyEnv()
		if got, want := r.stateDir.Value(), "/from-flag"; got != want {
			t.Errorf("stateDir = %q, want %q", got, want)
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
	stateDir := t.TempDir()
	wantDB := filepath.Join(stateDir, "superfolha.db")

	repo, err := openRepository(stateDir)
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
