package auth

import (
	"net/http"
	"strings"
	"time"
)

// AuthCookieName is the HttpOnly session cookie holding the JWT.
const AuthCookieName = "authToken"

// CookieSecure is true when the request arrived over HTTPS (including
// X-Forwarded-Proto from a TLS terminator). Local HTTP must not set Secure
// or the browser drops the session and login/register look like a silent no-op.
func CookieSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	if i := strings.IndexByte(proto, ','); i >= 0 {
		proto = proto[:i]
	}
	return strings.EqualFold(strings.TrimSpace(proto), "https")
}

// authCookieBase returns cookie fields shared by set and clear (flags must match
// or browsers keep stale cookies in mixed Secure contexts).
func authCookieBase(r *http.Request) http.Cookie {
	return http.Cookie{
		Name:     AuthCookieName,
		Path:     "/",
		HttpOnly: true,
		Secure:   CookieSecure(r),
		SameSite: http.SameSiteLaxMode,
	}
}

// SetAuthCookie writes the session JWT as an HttpOnly cookie with TokenTTL.
func SetAuthCookie(w http.ResponseWriter, r *http.Request, token string) {
	c := authCookieBase(r)
	c.Value = token
	c.Expires = time.Now().Add(TokenTTL)
	http.SetCookie(w, &c)
}

// ClearAuthCookie expires the session cookie (logout or invalid token).
func ClearAuthCookie(w http.ResponseWriter, r *http.Request) {
	c := authCookieBase(r)
	c.Value = ""
	c.MaxAge = -1
	c.Expires = time.Unix(0, 0)
	http.SetCookie(w, &c)
}
