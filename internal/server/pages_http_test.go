package server

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/superfolha/internal/auth"
	"github.com/lewtec/superfolha/internal/db"
	igit "github.com/lewtec/superfolha/internal/git"
	"github.com/lewtec/superfolha/internal/paths"
	"github.com/lewtec/superfolha/internal/project"
	"github.com/lewtec/superfolha/internal/session"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	var a db.DBArg
	if err := a.Parse(dir + "/t.db"); err != nil {
		t.Fatal(err)
	}
	repo, err := db.OpenArg(t.Context(), &a)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	srv := NewServer(repo, dir, project.NewService(dir), auth.NewService(repo))
	srv.hubs.SetCloner(func(dest, _, _ string, _ igit.SessionSSH) error {
		if err := igit.InitRepo(dest); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, "main.tex"), []byte("hi\n"), 0o644)
	})
	srv.hubs.SetProber(func(string, string, igit.SessionSSH) error { return nil })
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

func devEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "rod-play-secret")
	t.Setenv("GO_ENV", "development")
}

func mustKey(t *testing.T) *igit.SSHKey {
	t.Helper()
	k, err := igit.NewSessionSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func authCookie(value string) *http.Cookie {
	return &http.Cookie{Name: auth.AuthCookieName, Value: value}
}

func serve(srv *Server, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func withAuth(req *http.Request, cookie string) *http.Request {
	req.AddCookie(authCookie(cookie))
	return req
}

func postForm(srv *Server, cookie, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return serve(srv, withAuth(req, cookie))
}

func paperForm(pub string) string {
	return "remote=git@github.com:t/paper&branch=main&ssh_public=" + url.QueryEscape(pub)
}

func readyPaper(t *testing.T, srv *Server, pub string) *session.Live {
	t.Helper()
	live, err := srv.hubs.Create("alice", "git@github.com:t/paper", "main", pub)
	if err != nil {
		t.Fatal(err)
	}
	live.Ready = true
	return live
}

func TestLandingOK(t *testing.T) {
	t.Parallel()
	srv := testServer(t)
	res := serve(srv, httptest.NewRequest(http.MethodGet, paths.Landing, nil)).Result()
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
	devEnv(t)
	srv := testServer(t)
	tok := signIn(t, "alice")
	out := serve(srv, withAuth(httptest.NewRequest(http.MethodGet, paths.Landing, nil), tok))
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
	out := serve(srv, withAuth(httptest.NewRequest(http.MethodGet, paths.Projects(), nil), tok))
	if out.Code != http.StatusOK {
		t.Fatalf("GET /sessions with session = %d; want 200 (got Location %q)", out.Code, out.Header().Get("Location"))
	}
}

func TestProjectsRedirectsAnonymous(t *testing.T) {
	t.Parallel()
	srv := testServer(t)
	res := serve(srv, httptest.NewRequest(http.MethodGet, paths.Projects(), nil)).Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("GET /sessions = %d; want 303", res.StatusCode)
	}
	loc := res.Header.Get("Location")
	if !strings.HasPrefix(loc, paths.Login()) {
		t.Fatalf("Location = %q", loc)
	}
}

func TestChallengeSignLoginSetsCookie(t *testing.T) {
	devEnv(t)
	auth.ResetChallengeStateForTest()
	srv := testServer(t)
	chRec := serve(srv, httptest.NewRequest(http.MethodGet, paths.LoginChallenge(), nil))
	if chRec.Code != http.StatusOK {
		t.Fatalf("challenge = %d", chRec.Code)
	}
	var chBody struct {
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(chRec.Body).Decode(&chBody); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, []byte(chBody.Challenge))
	payload, err := json.Marshal(map[string]string{
		"challenge":  chBody.Challenge,
		"public_key": base64.RawURLEncoding.EncodeToString(pub),
		"signature":  base64.RawURLEncoding.EncodeToString(sig),
		"next":       paths.Projects(),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, paths.LoginVerify(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := serve(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify = %d body %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Set-Cookie") == "" {
		t.Fatal("expected auth cookie")
	}
}

func TestSSHCreateStaysOnSessionsWithKey(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	k := mustKey(t)
	rec := postForm(srv, alice, paths.Projects(), paperForm(k.Authorized))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d", rec.Code)
	}
	if strings.HasPrefix(rec.Header().Get("Location"), "/editor/") {
		t.Fatalf("SSH create must not open editor before deploy key: %q", rec.Header().Get("Location"))
	}
	out := serve(srv, withAuth(httptest.NewRequest(http.MethodGet, paths.Projects(), nil), alice))
	body, _ := io.ReadAll(out.Body)
	if !strings.Contains(string(body), "ssh-ed25519") && !strings.Contains(string(body), "ssh-") {
		t.Fatalf("sessions page should show deploy public key: %s", truncateForTest(body, 400))
	}
}

func TestCreateUsesPostedPublic(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	k := mustKey(t)
	rec := postForm(srv, alice, paths.Projects(), paperForm(k.Authorized))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d", rec.Code)
	}
	out := serve(srv, withAuth(httptest.NewRequest(http.MethodGet, paths.Projects(), nil), alice))
	body, _ := io.ReadAll(out.Body)
	if !strings.Contains(string(body), "ssh-ed25519") {
		t.Fatalf("sessions page missing posted key: %s", truncateForTest(body, 400))
	}
}

func TestCreateRejectsBadPublic(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	rec := postForm(srv, alice, paths.Projects(), "remote=git@github.com:t/paper&branch=main&ssh_public=not-a-key")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "sessions.ssh_seed_invalid") {
		t.Fatalf("Location = %q", rec.Header().Get("Location"))
	}
}

func TestCloneCreatesSessionAndSecondUserFails(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	k := mustKey(t)
	rec := postForm(srv, alice, paths.Projects(), paperForm(k.Authorized))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d", rec.Code)
	}
	if strings.HasPrefix(rec.Header().Get("Location"), "/editor/") {
		t.Fatalf("create must stay on sessions: %q", rec.Header().Get("Location"))
	}
	bob := signIn(t, "bob")
	rec2 := postForm(srv, bob, paths.Projects(), paperForm(k.Authorized))
	if rec2.Code != http.StatusSeeOther {
		t.Fatalf("bob create = %d", rec2.Code)
	}
	if !strings.Contains(rec2.Header().Get("Location"), "sessions.already_live") && !strings.Contains(rec2.Header().Get("Location"), "error=") {
		t.Fatalf("bob should fail unique session: %q", rec2.Header().Get("Location"))
	}
}

func TestLocalPathCreateOpensEditor(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	rec := postForm(srv, alice, paths.Projects(), "remote="+url.QueryEscape("/tmp/sf-local-paper")+"&branch=main")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/editor/") {
		t.Fatalf("local create must open editor: %q", loc)
	}
}

func TestHostRetryOfReadySessionOpensEditor(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	k := mustKey(t)
	live := readyPaper(t, srv, k.Authorized)
	rec := postForm(srv, alice, paths.Projects(), paperForm(k.Authorized))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("retry = %d", rec.Code)
	}
	want := paths.Editor(live.ID)
	if rec.Header().Get("Location") != want {
		t.Fatalf("Location = %q; want %q", rec.Header().Get("Location"), want)
	}
}

func TestHostInviteRedirectsToPreauthLink(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	k := mustKey(t)
	live := readyPaper(t, srv, k.Authorized)
	rec := serve(srv, withAuth(httptest.NewRequest(http.MethodPost, paths.SessionPreauth(live.ID), nil), alice))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("invite = %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, paths.Editor(live.ID)+"?preauth=") {
		t.Fatalf("Location = %q", loc)
	}
}

func TestHostInviteJSONReturnsLink(t *testing.T) {
	devEnv(t)
	srv := testServer(t)
	alice := signIn(t, "alice")
	k := mustKey(t)
	live := readyPaper(t, srv, k.Authorized)
	req := withAuth(httptest.NewRequest(http.MethodPost, paths.SessionPreauth(live.ID), nil), alice)
	req.Header.Set("Accept", "application/json")
	rec := serve(srv, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("invite json = %d", rec.Code)
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.URL, paths.Editor(live.ID)+"?preauth=") {
		t.Fatalf("url = %q", out.URL)
	}
}

func truncateForTest(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
