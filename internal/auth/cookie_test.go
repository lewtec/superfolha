package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetAndClearAuthCookie_FlagsMatch(t *testing.T) {
	t.Setenv("GO_ENV", "development")

	rec := httptest.NewRecorder()
	SetAuthCookie(rec, "tok-value")
	setHdr := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setHdr, AuthCookieName+"=tok-value") {
		t.Fatalf("set cookie missing value: %q", setHdr)
	}
	if !strings.Contains(setHdr, "HttpOnly") {
		t.Fatalf("set cookie missing HttpOnly: %q", setHdr)
	}
	if strings.Contains(setHdr, "Secure") {
		t.Fatalf("development set cookie should not be Secure: %q", setHdr)
	}
	if !strings.Contains(setHdr, "Path=/") {
		t.Fatalf("set cookie missing Path=/: %q", setHdr)
	}

	rec = httptest.NewRecorder()
	ClearAuthCookie(rec)
	clearHdr := rec.Header().Get("Set-Cookie")
	if !strings.Contains(clearHdr, AuthCookieName+"=") {
		t.Fatalf("clear cookie missing name: %q", clearHdr)
	}
	if !(strings.Contains(clearHdr, "Max-Age=0") || strings.Contains(clearHdr, "Max-Age=-1")) {
		t.Fatalf("clear cookie missing Max-Age: %q", clearHdr)
	}
	if strings.Contains(clearHdr, "Secure") {
		t.Fatalf("development clear cookie should not be Secure: %q", clearHdr)
	}
}

func TestSetAuthCookie_SecureOutsideDevelopment(t *testing.T) {
	t.Setenv("GO_ENV", "production")

	rec := httptest.NewRecorder()
	SetAuthCookie(rec, "tok")
	setHdr := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setHdr, "Secure") {
		t.Fatalf("production set cookie should be Secure: %q", setHdr)
	}
	if !strings.Contains(setHdr, "Expires=") {
		t.Fatalf("expected Expires on set cookie: %q", setHdr)
	}
}
