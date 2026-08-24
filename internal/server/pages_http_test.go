package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db/sqlite"
	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/paths"
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
	srv := NewServer(repo, dir, project.NewService(dir), auth.NewService(repo))
	srv.hubs.SetCloner(func(dest, _, _ string, _ *igit.HTTPAuth) error {
		if err := igit.InitRepo(dest); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, "main.tex"), []byte("hi\n"), 0o644)
	})
	return srv
}

func signIn(t *testing.T, login string) string {
	t.Helper()
	tok, err := auth.GenerateToken("42", login)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestLandingOK(t *testing.T) {
	t.Parallel()
	srv := testServer(t)
	req := httptest.NewRequest(http.MethodGet, paths.Landing, nil)
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
	if !strings.Contains(string(body), paths.Login()) {
		t.Fatalf("anonymous landing should link to login: %s", truncateForTest(body, 200))
	}
}

func TestLandingLoggedInPointsAtSessions(t *testing.T) {
	t.Setenv("JWT_SECRET", "rod-play-secret")
	t.Setenv("GO_ENV", "development")
	srv := testServer(t)
	tok := signIn(t, "alice")
	land := httptest.NewRequest(http.MethodGet, paths.Landing, nil)
	land.AddCookie(&http.Cookie{Name: auth.AuthCookieName, Value: tok})
	out := httptest.NewRecorder()
	srv.Handler().ServeHTTP(out, land)
	body, _ := io.ReadAll(out.Result().Body)
	if !strings.Contains(string(body), `href="`+paths.Projects()+`"`) {
		t.Fatalf("logged-in landing should link to sessions: %s", truncateForTest(body, 300))
	}
}

func TestCookieWorksOnHTTPWithoutGOEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "rod-play-secret")
	t.Setenv("GO_ENV", "")
	srv := testServer(t)
	tok := signIn(t, "alice")
	follow := httptest.NewRequest(http.MethodGet, paths.Projects(), nil)
	follow.AddCookie(&http.Cookie{Name: auth.AuthCookieName, Value: tok})
	out := httptest.NewRecorder()
	srv.Handler().ServeHTTP(out, follow)
	if out.Code != http.StatusOK {
		t.Fatalf("GET /sessions with session = %d; want 200 (got Location %q)", out.Code, out.Header().Get("Location"))
	}
}

func TestProjectsRedirectsAnonymous(t *testing.T) {
	t.Parallel()
	srv := testServer(t)
	req := httptest.NewRequest(http.MethodGet, paths.Projects(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET /sessions = %d; want 303", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if !strings.HasPrefix(loc, paths.Login()) {
		t.Fatalf("Location = %q", loc)
	}
}

func TestCloneCreatesSessionAndSecondUserFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "rod-play-secret")
	t.Setenv("GO_ENV", "development")
	srv := testServer(t)
	alice := signIn(t, "alice")
	form := strings.NewReader("remote=https://github.com/t/paper&branch=main")
	req := httptest.NewRequest(http.MethodPost, paths.Projects(), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: auth.AuthCookieName, Value: alice})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/editor/") {
		t.Fatalf("Location = %q", loc)
	}
	bob := signIn(t, "bob")
	form2 := strings.NewReader("remote=https://github.com/t/paper&branch=main")
	req2 := httptest.NewRequest(http.MethodPost, paths.Projects(), form2)
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.AddCookie(&http.Cookie{Name: auth.AuthCookieName, Value: bob})
	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusSeeOther {
		t.Fatalf("bob create = %d", rec2.Code)
	}
	if !strings.Contains(rec2.Header().Get("Location"), "sessions.already_live") && !strings.Contains(rec2.Header().Get("Location"), "error=") {
		t.Fatalf("bob should fail unique session: %q", rec2.Header().Get("Location"))
	}
}

func truncateForTest(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
