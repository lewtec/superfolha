package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db/sqlite"
	"github.com/lewtec/superfolha/internal/project"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	repo, err := sqlite.NewRepository(dir + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return NewServer(repo, dir, project.NewService(dir), auth.NewService(repo))
}

func TestLandingOK(t *testing.T) {
	t.Parallel()
	srv := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET / = %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "Superfolha") {
		t.Fatalf("landing missing brand: %s", truncateForTest(body, 200))
	}
}

func TestProjectsRedirectsAnonymous(t *testing.T) {
	t.Parallel()
	srv := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET /projects = %d; want 303", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if !strings.HasPrefix(loc, "/login") {
		t.Fatalf("Location = %q", loc)
	}
}

func truncateForTest(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
