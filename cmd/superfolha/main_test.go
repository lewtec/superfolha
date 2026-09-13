package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	xtest "github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/superfolha/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	dir := t.TempDir()
	app, err := cmd.Parse[cmd.App[root]](
		"--state-dir", dir,
		"--addr", "0.0.0.0:9090",
	)
	require.NoError(t, err)
	assert.Equal(t, dir, app.Args.stateDir.Value())
	assert.Equal(t, "0.0.0.0:9090", app.Args.addr.Value())
	assert.Nil(t, app.Args.version)
}

func TestParseVersionCommand(t *testing.T) {
	t.Setenv("STATE_DIR", t.TempDir())
	app, err := cmd.Parse[cmd.App[root]]("version")
	require.NoError(t, err)
	require.NotNil(t, app.Args.version)
}

func TestParseStateDirEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("STATE_DIR", dir)
	app, err := cmd.Parse[cmd.App[root]]()
	require.NoError(t, err)
	assert.Equal(t, dir, app.Args.stateDir.Value())

	flagDir := t.TempDir()
	app, err = cmd.Parse[cmd.App[root]]("--state-dir", flagDir)
	require.NoError(t, err)
	assert.Equal(t, flagDir, app.Args.stateDir.Value())
}

func TestParseAddr(t *testing.T) {
	t.Setenv("STATE_DIR", t.TempDir())
	t.Setenv("PORT", "8081")
	app, err := cmd.Parse[cmd.App[root]]()
	require.NoError(t, err)
	assert.Equal(t, ":8081", app.Args.addr.Value())

	app, err = cmd.Parse[cmd.App[root]]("--addr", "0.0.0.0:9090")
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0:9090", app.Args.addr.Value())
}

func TestParseDatabase(t *testing.T) {
	t.Setenv("STATE_DIR", t.TempDir())
	app, err := cmd.Parse[cmd.App[root]]("--database", "/tmp/x.db")
	require.NoError(t, err)
	require.NotNil(t, app.Args.database.Value())
	assert.Equal(t, "/tmp/x.db", app.Args.database.Value().URL())
}

func TestParseStateDirMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	_, err := cmd.Parse[cmd.App[root]]("--state-dir", missing)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestOpenRepository(t *testing.T) {
	stateDir := t.TempDir()
	wantDB := filepath.Join(stateDir, "superfolha.db")

	repo, err := db.OpenRepository(t.Context(), wantDB)
	require.NoError(t, err)
	require.NotNil(t, repo)
	xtest.CloseOnCleanup(t, repo)

	_, err = os.Stat(wantDB)
	require.NoError(t, err)
}
