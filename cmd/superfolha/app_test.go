package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/lewtec/superfolha/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWantWindow(t *testing.T) {
	assert.False(t, wantWindow(nil), "unstamped binary")
	for _, args := range [][]string{
		{"--help"},
		{"version"},
		{"--addr", "127.0.0.1:9"},
	} {
		assert.False(t, wantWindow(args), "args %q", args)
	}
}

func TestStampedVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{name: "empty", version: "", want: false},
		{name: "dev", version: "dev", want: false},
		{name: "dev suffix", version: "dev-abcdef12", want: false},
		{name: "release", version: "0.1.0", want: true},
		{name: "release vcs", version: "0.1.0-abcdef12", want: true},
		{name: "tag", version: "v1.2.3", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stampedVersion(tt.version))
		})
	}
}

func TestHeadlessHost(t *testing.T) {
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	assert.False(t, headlessHost())
	t.Setenv("LEWKIT_NO_UI", "1")
	assert.True(t, headlessHost())
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "yes")
	assert.True(t, headlessHost())
}

func TestOpenInstanceServesLanding(t *testing.T) {
	dir := t.TempDir()
	var database db.DBArg
	in, err := openInstance(t.Context(), dir, &database)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, in.Close())
	})
	rec := httptest.NewRecorder()
	in.srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "Superfolha")
}

func TestWindowPageQuotesTarget(t *testing.T) {
	target := `http://127.0.0.1:9/").alert(1)`
	rec := httptest.NewRecorder()
	windowPage(target).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	body := rec.Body.String()
	assert.Contains(t, body, strconv.Quote(target))
	assert.NotContains(t, body, target)
	assert.Contains(t, body, "location.replace(")
}
