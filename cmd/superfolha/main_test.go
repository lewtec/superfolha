package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/superfolha/internal/db"
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
		"--database",
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

func TestParseStateDirEnv(t *testing.T) {
	t.Setenv("STATE_DIR", "/var/sf")
	app, err := cmd.Parse[cmd.App[root]]()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := app.Args.stateDir.Value(), "/var/sf"; got != want {
		t.Errorf("stateDir = %q, want %q", got, want)
	}

	app, err = cmd.Parse[cmd.App[root]]("--state-dir", "/from-flag")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := app.Args.stateDir.Value(), "/from-flag"; got != want {
		t.Errorf("stateDir = %q, want %q", got, want)
	}
}

func TestParseAddr(t *testing.T) {
	t.Setenv("PORT", "8081")
	app, err := cmd.Parse[cmd.App[root]]()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := app.Args.addr.Value(), ":8081"; got != want {
		t.Errorf("addr = %q, want %q", got, want)
	}

	app, err = cmd.Parse[cmd.App[root]]("--addr", "0.0.0.0:9090")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := app.Args.addr.Value(), "0.0.0.0:9090"; got != want {
		t.Errorf("addr = %q, want %q", got, want)
	}
}

func TestParseDatabase(t *testing.T) {
	app, err := cmd.Parse[cmd.App[root]]("--database", "/tmp/x.db")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if app.Args.database.Value() == nil {
		t.Fatal("expected database arg")
	}
	if got, want := app.Args.database.Value().URL(), "/tmp/x.db"; got != want {
		t.Errorf("database = %q, want %q", got, want)
	}
}

func TestOpenRepository(t *testing.T) {
	stateDir := t.TempDir()
	wantDB := filepath.Join(stateDir, "superfolha.db")

	repo, err := db.OpenRepository(t.Context(), wantDB)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
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
