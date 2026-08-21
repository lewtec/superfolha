package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db/sqlite"
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
	return NewServer(repo, dir, project.NewService(dir), auth.NewService(repo))
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
}

func TestGhostJWTTreatedAsAnonymous(t *testing.T) {
	t.Setenv("JWT_SECRET", "rod-play-secret")
	t.Setenv("GO_ENV", "development")
	srv := testServer(t)
	token, err := auth.GenerateToken("01aaaaaaaaaaaaaaaaaaaaaaaa", "ghost@example.com")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, paths.Landing, nil)
	req.AddCookie(&http.Cookie{Name: auth.AuthCookieName, Value: token})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if strings.Contains(string(body), "ghost@example.com") {
		t.Fatal("ghost JWT must not show as signed in")
	}
	if !strings.Contains(string(body), paths.Register()) {
		t.Fatalf("anonymous landing should link to register: %s", truncateForTest(body, 200))
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, auth.AuthCookieName+"=") {
		t.Fatalf("expected ghost cookie cleared, Set-Cookie=%q", setCookie)
	}
}

func TestLandingLoggedInPointsAtProjects(t *testing.T) {
	t.Setenv("JWT_SECRET", "rod-play-secret")
	t.Setenv("GO_ENV", "development")
	srv := testServer(t)
	form := strings.NewReader("email=in@example.com&password=testhorses1&confirm=testhorses1")
	req := httptest.NewRequest(http.MethodPost, paths.Register(), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("register status = %d", rec.Code)
	}
	cookie := rec.Result().Header.Get("Set-Cookie")
	land := httptest.NewRequest(http.MethodGet, paths.Landing, nil)
	land.Header.Set("Cookie", cookie)
	out := httptest.NewRecorder()
	srv.Handler().ServeHTTP(out, land)
	body, _ := io.ReadAll(out.Result().Body)
	if !strings.Contains(string(body), `href="`+paths.Projects()+`"`) {
		t.Fatalf("logged-in landing should link to projects: %s", truncateForTest(body, 300))
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
		t.Fatalf("GET /projects = %d; want 303", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if !strings.HasPrefix(loc, paths.Login()) {
		t.Fatalf("Location = %q", loc)
	}
}

func truncateForTest(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
