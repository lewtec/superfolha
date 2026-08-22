package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetAndClearAuthCookie_HTTPNotSecure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/", nil)
	rec := httptest.NewRecorder()
	SetAuthCookie(rec, req, "tok-value")
	setHdr := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setHdr, AuthCookieName+"=tok-value") {
		t.Fatalf("set cookie missing value: %q", setHdr)
	}
	if !strings.Contains(setHdr, "HttpOnly") {
		t.Fatalf("set cookie missing HttpOnly: %q", setHdr)
	}
	if strings.Contains(setHdr, "Secure") {
		t.Fatalf("HTTP set cookie must not be Secure: %q", setHdr)
	}
	if !strings.Contains(setHdr, "Path=/") {
		t.Fatalf("set cookie missing Path=/: %q", setHdr)
	}

	rec = httptest.NewRecorder()
	ClearAuthCookie(rec, req)
	clearHdr := rec.Header().Get("Set-Cookie")
	if !strings.Contains(clearHdr, AuthCookieName+"=") {
		t.Fatalf("clear cookie missing name: %q", clearHdr)
	}
	if !(strings.Contains(clearHdr, "Max-Age=0") || strings.Contains(clearHdr, "Max-Age=-1")) {
		t.Fatalf("clear cookie missing Max-Age: %q", clearHdr)
	}
	if strings.Contains(clearHdr, "Secure") {
		t.Fatalf("HTTP clear cookie must not be Secure: %q", clearHdr)
	}
}

func TestSetAuthCookie_SecureOnHTTPS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	SetAuthCookie(rec, req, "tok")
	setHdr := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setHdr, "Secure") {
		t.Fatalf("HTTPS set cookie should be Secure: %q", setHdr)
	}
}

func TestSetAuthCookie_SecureOnForwardedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("X-Forwarded-Proto", "https, http")
	rec := httptest.NewRecorder()
	SetAuthCookie(rec, req, "tok")
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Secure") {
		t.Fatalf("X-Forwarded-Proto=https should be Secure: %q", rec.Header().Get("Set-Cookie"))
	}
}
